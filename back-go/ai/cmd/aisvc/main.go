// aisvc — микросервис работы с ИИ Groove Work (LLM-шлюз).
//
// Владеет AI-ключами компаний (Fernet, AI_KEY_ENCRYPTION_KEY) и вызовами
// ProxyAPI/OpenAI (chat completions + embeddings), таблицей task_embeddings
// (pgvector: индексация + семантический поиск задач), а также ТВ-фактом дня
// (фоновая генерация раз в час, кэш в Redis gw2:ai:tv_fact:{cid}).
//
// Транспорты:
//   - HTTP/Fiber (HTTP_ADDR) — REST /api/companies/:id/ai-settings*,
//     /api/ai/tv-fact и /api/ai/assistant/* (деловой ИИ-ассистент, свой
//     tools-цикл и хранилище диалога — внутри aisvc, см. internal/service/assistant.go);
//   - gRPC (GRPC_ADDR) — Status/Chat/Embed/SemanticSearch/ReindexTask
//     (SemanticSearch/Embed зовёт tasksvc; Chat используется и снаружи, и
//     внутрипроцессно самим ассистентом).
//
// Зависимости: общая PostgreSQL платформы (схему ведёт goose-migrate) + Redis
// (кэш ТВ-фактов).
package main

import (
	"net"
	"os"

	googrpc "google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/clients"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/llm"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/repository/redisx"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/secret"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/service"
	grpctransport "github.com/DmitriyODS/gw2/back-go/ai/internal/transport/grpc"
	httptransport "github.com/DmitriyODS/gw2/back-go/ai/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/aipb"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	baseURL := bootstrap.Env("AI_API_BASE_URL", llm.DefaultBaseURL)
	// Fernet-ключ шифрования AI-ключей компаний. Пустой не фатален на старте:
	// расшифровка тихо выключит AI-фичи, шифрование при PUT отдаст
	// AI_KEY_NOT_CONFIGURED — ровно как во Flask.
	encKey := os.Getenv("AI_KEY_ENCRYPTION_KEY")
	if encKey == "" {
		log.Warn("AI_KEY_ENCRYPTION_KEY не задан — AI-фичи будут выключены")
	}
	// tasksvc — gRPC-клиент для инструментов ИИ-ассистента (статистика,
	// поиск/ссылки на задачи).
	tasksAddr := bootstrap.Env("TASKS_GRPC_ADDR", "localhost:9095")
	// APP_PUBLIC_BASE_URL — тот же паттерн формирования абсолютных ссылок,
	// что уже использует authsvc для писем (verify-email/reset-password).
	appBaseURL := bootstrap.Env("APP_PUBLIC_BASE_URL", "http://localhost:5173")
	// Платформенный LLM техподдержки (dev-чат мессенджера): ключ НЕ компанийный.
	// Пустой — ИИ поддержки выключен (msgsvc откатится на канированный автоответ).
	support := service.SupportConfig{
		APIKey: os.Getenv("SUPPORT_AI_API_KEY"),
		Model:  os.Getenv("SUPPORT_AI_MODEL"),
	}
	if support.APIKey == "" {
		log.Warn("SUPPORT_AI_API_KEY не задан — ИИ техподдержки выключен")
	}
	// Публичный ключ access-токенов PASETO (v4.public): токены выпускает
	// authsvc, мы только проверяем подпись.
	verifier, err := pasetoauth.NewVerifier(bootstrap.MustEnv(log, "PASETO_PUBLIC_KEY"))
	if err != nil {
		log.Error("paseto.bad_public_key", "error", err)
		os.Exit(1)
	}

	ctx, stop := bootstrap.SignalContext()
	defer stop()

	pool := bootstrap.MustPostgres(ctx, log, dbURL)
	defer pool.Close()
	rdb := bootstrap.MustRedis(log, redisURL)
	defer rdb.Close()

	repo := postgres.NewRepo(pool)
	users := postgres.NewUserReader(pool)
	facts := redisx.NewFactCache(rdb, log)
	assistants := postgres.NewAssistantRepo(pool)
	tasksClient, err := clients.NewTasks(tasksAddr, log)
	if err != nil {
		log.Error("tasks.client_failed", "error", err)
		os.Exit(1)
	}
	defer tasksClient.Close()
	svc := service.New(repo, llm.New(baseURL, log), secret.New(encKey), facts,
		assistants, tasksClient, appBaseURL, support, log)
	eps := endpoint.New(svc)

	// Фоновый цикл ТВ-фактов: стартовый проход + тик раз в час.
	go svc.RunTVFactsLoop(ctx)

	grpcAddr := bootstrap.Env("GRPC_ADDR", ":9093")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8093")

	grpcServer := googrpc.NewServer()
	aipb.RegisterAiServiceServer(grpcServer, grpctransport.NewServer(eps))
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc.listen_failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	httpServer := httptransport.NewServer(eps, users, verifier, log)

	log.Info("listening", "grpc", grpcAddr, "http", httpAddr)
	bootstrap.Run(ctx, log,
		bootstrap.Component{
			Name: "grpc",
			Run:  func() error { return grpcServer.Serve(listener) },
			Stop: grpcServer.GracefulStop,
		},
		bootstrap.Component{
			Name: "http",
			Run:  func() error { return httpServer.Listen(httpAddr) },
			Stop: func() {
				if err := httpServer.Shutdown(); err != nil {
					log.Warn("http.shutdown_failed", "error", err)
				}
			},
		},
	)
}
