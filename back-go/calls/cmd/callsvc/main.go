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
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	googrpc "google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/calls/gen/callspb"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/events"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/livekit"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/ringstate"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/service"
	grpctransport "github.com/DmitriyODS/gw2/back-go/calls/internal/transport/grpc"
	httptransport "github.com/DmitriyODS/gw2/back-go/calls/internal/transport/http"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	dbURL := env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := env("REDIS_URL", "redis://localhost:6379/0")
	// Публичный ключ access-токенов PASETO (v4.public): токены выпускает
	// authsvc, мы только проверяем подпись.
	pasetoPublicHex := os.Getenv("PASETO_PUBLIC_KEY")
	if pasetoPublicHex == "" {
		log.Error("PASETO_PUBLIC_KEY не задан")
		os.Exit(1)
	}
	pasetoPublic, err := paseto.NewV4AsymmetricPublicKeyFromHex(pasetoPublicHex)
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Error("postgres.connect_failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Error("redis.bad_url", "error", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(redisOpts)
	defer rdb.Close()

	lk := livekit.New(livekit.Config{
		APIKey:    env("LIVEKIT_API_KEY", "devkey"),
		APISecret: env("LIVEKIT_API_SECRET", "dev_livekit_secret_min_32_chars_ok"),
		APIURL:    env("LIVEKIT_URL", "http://localhost:7880"),
		ClientURL: env("LIVEKIT_CLIENT_URL", "/livekit"),
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

	grpcAddr := env("GRPC_ADDR", ":9090")
	httpAddr := env("HTTP_ADDR", ":8090")

	grpcServer := googrpc.NewServer()
	callspb.RegisterCallServiceServer(grpcServer, grpctransport.NewServer(eps))
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc.listen_failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	httpServer := httptransport.NewServer(eps, svc, lk, users, pasetoPublic, log)

	errCh := make(chan error, 2)
	go func() {
		log.Info("grpc.listening", "addr", grpcAddr)
		errCh <- grpcServer.Serve(listener)
	}()
	go func() {
		log.Info("http.listening", "addr", httpAddr)
		errCh <- httpServer.Listen(httpAddr)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown.signal")
	case err := <-errCh:
		log.Error("server.failed", "error", err)
	}

	grpcServer.GracefulStop()
	if err := httpServer.Shutdown(); err != nil {
		log.Warn("http.shutdown_failed", "error", err)
	}
	log.Info("shutdown.done")
}
