// Package bootstrap — общий каркас запуска Go-микросервисов Groove Work:
// env-конфиг, slog JSON, подключения PostgreSQL/Redis, graceful shutdown.
//
// Фатальные ошибки конфигурации/подключений завершают процесс (os.Exit(1)):
// в docker-compose сервис перезапустится, healthcheck не пройдёт — это
// осознанный fail-fast на старте.
package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Env — значение переменной окружения или fallback, если пусто.
func Env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// MustEnv — обязательная переменная окружения; пустая — фатал.
func MustEnv(log *slog.Logger, key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Error(key + " не задан")
		os.Exit(1)
	}
	return v
}

// Logger — slog с JSON-выводом в stdout, ставится дефолтным.
func Logger() *slog.Logger {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)
	return log
}

// SignalContext — контекст, отменяемый SIGINT/SIGTERM.
func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

// MustPostgres — пул pgx к общей PostgreSQL платформы (схему ведёт Alembic).
func MustPostgres(ctx context.Context, log *slog.Logger, url string) *pgxpool.Pool {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Error("postgres.connect_failed", "error", err)
		os.Exit(1)
	}
	return pool
}

// MustRedis — клиент Redis по URL.
func MustRedis(log *slog.Logger, url string) *redis.Client {
	opts, err := redis.ParseURL(url)
	if err != nil {
		log.Error("redis.bad_url", "error", err)
		os.Exit(1)
	}
	return redis.NewClient(opts)
}

// Component — запускаемый сервер: Run блокируется до ошибки/остановки,
// Stop — graceful shutdown (ошибки остановки логирует сам).
type Component struct {
	Name string
	Run  func() error
	Stop func()
}

// Run — поднять компоненты, дождаться сигнала или первой ошибки,
// остановить все. Повторяет прежний select{ctx.Done, errCh} каждого main.
func Run(ctx context.Context, log *slog.Logger, components ...Component) {
	errCh := make(chan error, len(components))
	for _, comp := range components {
		comp := comp
		go func() {
			errCh <- comp.Run()
		}()
	}

	select {
	case <-ctx.Done():
		log.Info("shutdown.signal")
	case err := <-errCh:
		log.Error("server.failed", "error", err)
	}

	for _, comp := range components {
		comp.Stop()
	}
	log.Info("shutdown.done")
}
