// groovesvc — микросервис «Мой Groove» Groove Work: лента активности,
// реакции, комментарии, кудосы, заряды, питомцы-Грувики, зоопарк, магазин,
// недельные рейды и AI-механики Грувика.
//
// Транспорты:
//   - HTTP/Fiber (HTTP_ADDR) — REST /api/groove/* (за nginx);
//   - gRPC (GRPC_ADDR) — хуки доменных событий (Flask — юниты/задачи,
//     msgsvc — pet-чат).
//
// Зависимости: общая PostgreSQL платформы (схему ведёт goose-migrate), Redis
// (дневные капы, кэши, события Socket.IO через канал gw2:groove:events),
// gRPC-клиенты aisvc (LLM) и msgsvc (pet-чат).
package main

import (
	"net"
	"os"

	googrpc "google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/clients"
	"github.com/DmitriyODS/gw2/back-go/groove/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/groove/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/groove/internal/repository/redisx"
	"github.com/DmitriyODS/gw2/back-go/groove/internal/service"
	grpctransport "github.com/DmitriyODS/gw2/back-go/groove/internal/transport/grpc"
	httptransport "github.com/DmitriyODS/gw2/back-go/groove/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/groove/internal/weather"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/groovepb"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	aiAddr := bootstrap.Env("AI_GRPC_ADDR", "localhost:9093")
	messengerAddr := bootstrap.Env("MESSENGER_GRPC_ADDR", "localhost:9092")
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

	aiClient, err := clients.NewAI(aiAddr, log)
	if err != nil {
		log.Error("ai_grpc.bad_addr", "error", err)
		os.Exit(1)
	}
	defer aiClient.Close()
	msgrClient, err := clients.NewMessenger(messengerAddr, log)
	if err != nil {
		log.Error("messenger_grpc.bad_addr", "error", err)
		os.Exit(1)
	}
	defer msgrClient.Close()

	feedRepo := postgres.NewFeedRepo(pool)
	petRepo := postgres.NewPetRepo(pool)
	platform := postgres.NewPlatformRepo(pool)
	locRepo := postgres.NewLocationRepo(pool)
	daily := redisx.New(rdb, log)
	pub := events.NewPublisher(rdb, log, "gw2:groove:events")
	weatherCli := weather.New(log)

	svc := service.New(feedRepo, petRepo, platform, platform, platform,
		platform, locRepo, daily, pub, aiClient, msgrClient, weatherCli, log)
	eps := endpoint.New(svc)

	// Фоновые циклы: забота (болезни, характеры), AI (фразы, дайджест)
	// и погода (Open-Meteo → реплики Грувика).
	go svc.RunCareLoop(ctx)
	go svc.RunAILoop(ctx)
	go svc.RunWeatherLoop(ctx)

	grpcAddr := bootstrap.Env("GRPC_ADDR", ":9094")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8094")

	grpcServer := googrpc.NewServer()
	groovepb.RegisterGrooveServiceServer(grpcServer, grpctransport.NewServer(svc, log))
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc.listen_failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	httpServer := httptransport.NewServer(eps, platform, verifier, log)

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
