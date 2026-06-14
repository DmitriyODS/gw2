// pushsvc — микросервис пуш-уведомлений Groove Work (FCM).
//
// Подписан на Redis-каналы событий (gw2:messenger:events — новые сообщения,
// gw2:tasks:events — новые задачи, gw2:gateway:events — входящие звонки) и
// шлёт пуши через FCM HTTP v1 тем получателям, кого нет в онлайне (FCM-first:
// открытое приложение получит событие по WS, фоновое — пушем). Токены
// устройств регистрируются по REST /api/push/register|unregister.
//
// Без service-account ключа Firebase сервис стартует, но отправка отключена.
package main

import (
	"log/slog"
	"os"

	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/push/internal/consumer"
	"github.com/DmitriyODS/gw2/back-go/push/internal/fcm"
	"github.com/DmitriyODS/gw2/back-go/push/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/push/internal/repository/redisx"
	"github.com/DmitriyODS/gw2/back-go/push/internal/service"
	httptransport "github.com/DmitriyODS/gw2/back-go/push/internal/transport/http"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8097")

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

	sender, err := fcm.New(ctx, firebaseCredentials(log), log)
	if err != nil {
		log.Error("fcm.init_failed", "error", err)
		os.Exit(1)
	}

	svc := service.New(service.Deps{
		Tokens:   postgres.NewTokenStore(pool),
		Users:    postgres.NewUserDirectory(pool),
		Presence: redisx.NewPresence(rdb),
		Sender:   sender,
		Log:      log,
	})

	cons := consumer.New(rdb, svc, log)
	server := httptransport.NewServer(svc, verifier, log)

	log.Info("listening", "http", httpAddr)
	bootstrap.Run(ctx, log,
		bootstrap.Component{
			Name: "consumer",
			Run:  func() error { cons.Run(ctx); return nil },
			Stop: func() {},
		},
		bootstrap.Component{
			Name: "http",
			Run:  func() error { return server.Listen(httpAddr) },
			Stop: func() {
				if err := server.Shutdown(); err != nil {
					log.Warn("http.shutdown_failed", "error", err)
				}
			},
		},
	)
}

// firebaseCredentials — service-account JSON из env FIREBASE_CREDENTIALS_JSON
// (содержимое) либо GOOGLE_APPLICATION_CREDENTIALS (путь к файлу). Пусто —
// отправка отключается (no-op sender).
func firebaseCredentials(log *slog.Logger) []byte {
	if raw := os.Getenv("FIREBASE_CREDENTIALS_JSON"); raw != "" {
		return []byte(raw)
	}
	if path := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Warn("fcm.creds_file_unreadable", "path", path, "error", err)
			return nil
		}
		return data
	}
	return nil
}
