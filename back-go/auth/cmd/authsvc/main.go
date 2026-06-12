// authsvc — микросервис авторизации и пользователей Groove Work.
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/auth/* и /api/users/*
// (за nginx, мимо Flask). Выпускает PASETO-токены: access — v4.public
// (Ed25519, PASETO_PRIVATE_KEY), проверяется Flask и callsvc по публичному
// ключу; refresh — v4.local (PASETO_REFRESH_KEY) в HttpOnly-cookie.
//
// Зависимости: общая PostgreSQL платформы (схему ведёт Alembic; пароли —
// pgcrypto), Redis (анти-brute-force), общий uploads-volume (аватарки).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/avatar"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/repository/redisx"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/service"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/token"
	httptransport "github.com/DmitriyODS/gw2/back-go/auth/internal/transport/http"
)

const (
	accessTTL  = 15 * time.Minute
	refreshTTL = 30 * 24 * time.Hour
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
	uploadFolder := env("UPLOAD_FOLDER", "/app/uploads")

	privateKey := os.Getenv("PASETO_PRIVATE_KEY")
	refreshKey := os.Getenv("PASETO_REFRESH_KEY")
	if privateKey == "" || refreshKey == "" {
		log.Error("PASETO_PRIVATE_KEY/PASETO_REFRESH_KEY не заданы")
		os.Exit(1)
	}
	issuer, err := token.NewIssuer(privateKey, refreshKey, accessTTL, refreshTTL)
	if err != nil {
		log.Error("paseto.bad_keys", "error", err)
		os.Exit(1)
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

	repo := postgres.NewUserRepository(pool)
	throttle := redisx.NewLoginThrottle(rdb, log)
	avatars := avatar.NewStorage(uploadFolder)
	svc := service.New(repo, throttle, issuer, avatars, log)
	eps := endpoint.New(svc)

	httpAddr := env("HTTP_ADDR", ":8091")
	httpServer := httptransport.NewServer(eps, token.VerifierFromIssuer(issuer), repo, log)

	errCh := make(chan error, 1)
	go func() {
		log.Info("http.listening", "addr", httpAddr, "public_key", issuer.PublicKeyHex())
		errCh <- httpServer.Listen(httpAddr)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown.signal")
	case err := <-errCh:
		log.Error("server.failed", "error", err)
	}

	if err := httpServer.Shutdown(); err != nil {
		log.Warn("http.shutdown_failed", "error", err)
	}
	log.Info("shutdown.done")
}
