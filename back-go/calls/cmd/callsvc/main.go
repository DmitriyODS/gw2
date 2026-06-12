// callsvc — микросервис звонков Groove Work.
//
// Транспорты:
//   - gRPC (GRPC_ADDR) — ринг-фаза, дёргается Flask-шлюзом из Socket.IO;
//   - HTTP/Fiber (HTTP_ADDR) — REST /api/calls/* (за nginx) и вебхуки LiveKit.
//
// Зависимости: общая PostgreSQL платформы (схему ведёт Alembic), Redis
// (публикация событий для Flask), LiveKit (медиа).
package main

import (
	"net"
	"os"
	"strconv"
	"time"

	googrpc "google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/callspb"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/events"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/livekit"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/ringstate"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/service"
	grpctransport "github.com/DmitriyODS/gw2/back-go/calls/internal/transport/grpc"
	httptransport "github.com/DmitriyODS/gw2/back-go/calls/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	// Публичный ключ access-токенов PASETO (v4.public): токены выпускает
	// authsvc, мы только проверяем подпись.
	verifier, err := pasetoauth.NewVerifier(bootstrap.MustEnv(log, "PASETO_PUBLIC_KEY"))
	if err != nil {
		log.Error("paseto.bad_public_key", "error", err)
		os.Exit(1)
	}
	tokenTTL := 6 * time.Hour
	if raw := os.Getenv("LIVEKIT_TOKEN_TTL"); raw != "" {
		if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
			tokenTTL = time.Duration(sec) * time.Second
		}
	}

	ctx, stop := bootstrap.SignalContext()
	defer stop()

	pool := bootstrap.MustPostgres(ctx, log, dbURL)
	defer pool.Close()
	rdb := bootstrap.MustRedis(log, redisURL)
	defer rdb.Close()

	lk := livekit.New(livekit.Config{
		APIKey:    bootstrap.Env("LIVEKIT_API_KEY", "devkey"),
		APISecret: bootstrap.Env("LIVEKIT_API_SECRET", "dev_livekit_secret_min_32_chars_ok"),
		APIURL:    bootstrap.Env("LIVEKIT_URL", "http://localhost:7880"),
		ClientURL: bootstrap.Env("LIVEKIT_CLIENT_URL", "/livekit"),
		TokenTTL:  tokenTTL,
	}, log)

	repo := postgres.NewCallRepository(pool)
	users := postgres.NewUserReader(pool)
	ring := ringstate.New()
	pub := events.NewPublisher(rdb, log)
	svc := service.New(repo, users, ring, lk, pub, log)
	eps := endpoint.New(svc)

	// Зависшие с прошлого запуска звонки: живые комнаты восстанавливаем,
	// мёртвые финализируем. Ошибка не фатальна (БД могла ещё не мигрировать
	// при самом первом деплое) — сервис поднимется, состояние догонят вебхуки.
	if err := svc.ReconcileStartup(ctx); err != nil {
		log.Warn("calls.startup_reconcile_failed", "error", err)
	}

	grpcAddr := bootstrap.Env("GRPC_ADDR", ":9090")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8090")

	grpcServer := googrpc.NewServer()
	callspb.RegisterCallServiceServer(grpcServer, grpctransport.NewServer(eps))
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc.listen_failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	httpServer := httptransport.NewServer(eps, svc, lk, users, verifier, log)

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
