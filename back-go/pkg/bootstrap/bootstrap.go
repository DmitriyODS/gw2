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
	"strconv"
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

// EnvInt — целочисленная переменная окружения или fallback (пусто/не число).
func EnvInt(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
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
//
// MaxConns — фиксированный потолок (DB_POOL_MAX_CONNS, по умолчанию 10), а не
// дефолт pgxpool "runtime.NumCPU()": тот зависит от числа ядер ХОСТА, а не от
// реальной нагрузки сервиса — на многоядерной машине двадцать сервисов с
// таким дефолтом суммарно легко превышают max_connections одной общей
// Postgres, и все сервисы разом ловят "sorry, too many clients already".
func MustPostgres(ctx context.Context, log *slog.Logger, url string) *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		log.Error("postgres.bad_url", "error", err)
		os.Exit(1)
	}
	cfg.MaxConns = int32(EnvInt("DB_POOL_MAX_CONNS", 10))
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
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
