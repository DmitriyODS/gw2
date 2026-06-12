// tasksvc — микросервис задач Groove Work (ядро платформы).
//
// Владеет задачами, юнитами, типами юнитов, этапами, отделами,
// комментариями, избранным, личными цветами, статистикой (включая
// xlsx-экспорт) и YouGile-интеграцией (личные ключи пользователей,
// настройки компаний, импорт/экспорт задач, двусторонняя синхра через
// вебхук). Схему таблиц ведёт migrate-контейнер (goose, back-go/migrate).
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/tasks|units|unit-types|
// departments|stages|stats|yougile (за nginx).
//
// Межсервисное: groovesvc (gRPC, хуки геймификации), aisvc (gRPC,
// семантический поиск + реиндекс эмбеддингов). Сокет-события клиентам —
// Redis-канал gw2:tasks:events (доставляет gatewaysvc).
package main

import (
	"os"

	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/clients"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/service"
	httptransport "github.com/DmitriyODS/gw2/back-go/tasks/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/yougile"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	grooveAddr := bootstrap.Env("GROOVE_GRPC_ADDR", "localhost:9094")
	aiAddr := bootstrap.Env("AI_GRPC_ADDR", "localhost:9093")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8095")

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

	groove, err := clients.NewGroove(grooveAddr, log)
	if err != nil {
		log.Error("groove.client_failed", "error", err)
		os.Exit(1)
	}
	defer groove.Close()
	ai, err := clients.NewAI(aiAddr, log)
	if err != nil {
		log.Error("ai.client_failed", "error", err)
		os.Exit(1)
	}
	defer ai.Close()

	repo := postgres.NewRepo(pool)
	users := postgres.NewUserReader(pool)
	svc := service.New(service.Deps{
		Tasks: repo, Units: repo, UnitTypes: repo, Depts: repo, Stages: repo,
		Comments: repo, Stats: repo, Users: users, Companies: users,
		Groove: groove, AI: ai,
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

	log.Info("listening", "http", httpAddr)
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
	)
}
