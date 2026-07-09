// portalsvc — микросервис корпоративного портала Groove Work.
//
// Владеет порталом компании: постами (с вложениями), плоскими комментариями,
// реакциями, закреплением (лимит 10 на компанию) и тематическими разделами.
// Полностью независим от питомцев-грувиков (petsvc). Пересылка поста в
// мессенджер — единственный межсервисный вызов, gRPC к msgsvc
// (CreatePostMessage). Схему таблиц ведёт migrate-контейнер (goose).
//
// Транспорт один — HTTP/Fiber (HTTP_ADDR): REST /api/portal/* (за nginx);
// своего gRPC-сервера у portalsvc нет.
//
// Сокет-события клиентам — Redis-канал gw2:portal:events (доставляет
// gatewaysvc). Вложения — общий uploads-том/S3 (pkg/storage, префикс "portal").
package main

import (
	"os"

	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/pkg/storage"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/clients"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/service"
	httptransport "github.com/DmitriyODS/gw2/back-go/portal/internal/transport/http"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	uploadFolder := bootstrap.Env("UPLOAD_FOLDER", "../../uploads")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8102")
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

	messenger, err := clients.NewMessenger(messengerAddr, log)
	if err != nil {
		log.Error("messenger.client_init_failed", "error", err)
		os.Exit(1)
	}
	defer messenger.Close()

	repo := postgres.NewRepo(pool)
	users := postgres.NewUserReader(pool)
	svc := service.New(service.Deps{
		Repo:      repo,
		Files:     records.NewFileStore(storage.FromEnv(log, uploadFolder), "portal"),
		Bus:       events.NewPublisher(rdb, log, "gw2:portal:events"),
		Messenger: messenger,
		Log:       log,
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
