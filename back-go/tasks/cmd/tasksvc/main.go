// tasksvc — микросервис задач Groove Work (ядро платформы).
//
// Владеет задачами, юнитами, типами юнитов, этапами, отделами,
// комментариями, избранным, личными цветами, статистикой (включая
// xlsx-экспорт) и YouGile-интеграцией (личные ключи пользователей,
// настройки компаний, импорт/экспорт задач, двусторонняя синхра через
// вебхук). Схему таблиц ведёт migrate-контейнер (goose, back-go/migrate).
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/tasks|units|unit-types|
// departments|stages|stats|yougile (за nginx); gRPC (GRPC_ADDR) —
// TasksService, исходящий контракт для aisvc (статистика/поиск задач для
// инструментов ИИ-ассистента).
//
// Межсервисное: petsvc (gRPC, хуки геймификации), aisvc (gRPC-клиент,
// семантический поиск + реиндекс эмбеддингов; и gRPC-сервер — наоборот,
// aisvc зовёт нас). Сокет-события клиентам — Redis-канал gw2:tasks:events
// (доставляет gatewaysvc).
package main

import (
	"net"
	"os"

	googrpc "google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/taskspb"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/clients"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/service"
	grpctransport "github.com/DmitriyODS/gw2/back-go/tasks/internal/transport/grpc"
	httptransport "github.com/DmitriyODS/gw2/back-go/tasks/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/yougile"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	petsAddr := bootstrap.Env("PETS_GRPC_ADDR", "localhost:9094")
	aiAddr := bootstrap.Env("AI_GRPC_ADDR", "localhost:9093")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8095")
	grpcAddr := bootstrap.Env("GRPC_ADDR", ":9095")

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

	pets, err := clients.NewPets(petsAddr, log)
	if err != nil {
		log.Error("pets.client_failed", "error", err)
		os.Exit(1)
	}
	defer pets.Close()
	ai, err := clients.NewAI(aiAddr, log)
	if err != nil {
		log.Error("ai.client_failed", "error", err)
		os.Exit(1)
	}
	defer ai.Close()

	repo := postgres.NewRepo(pool)
	users := postgres.NewUserReader(pool)
	svc := service.New(service.Deps{
		Tasks: repo, Tags: repo, Units: repo, UnitTypes: repo, Depts: repo,
		Stages: repo, Comments: repo, Stats: repo, Users: users, Companies: users,
		Pets: pets, AI: ai,
		Bus: events.NewPublisher(rdb, log, "gw2:tasks:events"),
		Log: log,
	})
	yg := service.NewYougile(service.YougileDeps{
		Service: svc,
		Repo:    repo,
		Cipher:  yougile.NewCipher(bootstrap.Env("YOUGILE_ENC_KEY", "")),
		NewClient: func(key string) domain.YougileAPI {
			return yougile.NewClient(key)
		},
		PublicBase: bootstrap.Env("YOUGILE_WEBHOOK_PUBLIC_BASE", ""),
		Log:        log,
	})
	eps := endpoint.New(svc, yg)

	httpServer := httptransport.NewServer(eps, users, verifier, log)

	grpcServer := googrpc.NewServer()
	taskspb.RegisterTasksServiceServer(grpcServer, grpctransport.NewServer(eps))
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc.listen_failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	log.Info("listening", "http", httpAddr, "grpc", grpcAddr)
	bootstrap.Run(ctx, log,
		bootstrap.Component{
			Name: "http",
			Run:  func() error { return httpServer.Listen(httpAddr) },
			Stop: func() {
				if err := httpServer.Shutdown(); err != nil {
					log.Warn("http.shutdown_failed", "error", err)
				}
			},
		},
		bootstrap.Component{
			Name: "grpc",
			Run:  func() error { return grpcServer.Serve(listener) },
			Stop: grpcServer.GracefulStop,
		},
	)
}
