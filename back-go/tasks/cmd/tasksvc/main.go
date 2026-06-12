// tasksvc — микросервис задач Groove Work (ядро платформы).
//
// Владеет задачами, юнитами, типами юнитов, этапами, отделами,
// комментариями, избранным, личными цветами и статистикой (включая
// xlsx-экспорт). Схему таблиц по-прежнему ведёт Alembic на стороне Flask.
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/tasks|units|unit-types|
// departments|stages|stats (за nginx, мимо Flask).
//
// Межсервисное: groovesvc (gRPC, хуки геймификации), aisvc (gRPC,
// семантический поиск + реиндекс эмбеддингов). Сокет-события клиентам —
// Redis-канал gw2:tasks:events (generic-мост Flask); события с префиксом
// «_» дёргают python-обработчики (YouGile-пуш до фазы 4).
package main

import (
	"os"

	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/clients"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/service"
	httptransport "github.com/DmitriyODS/gw2/back-go/tasks/internal/transport/http"
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
	eps := endpoint.New(svc)

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
