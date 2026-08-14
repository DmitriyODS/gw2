// portalsvc — микросервис корпоративного портала Groove Work.
//
// Владеет порталом компании: постами (с вложениями), плоскими комментариями,
// реакциями, закреплением (лимит 10 на компанию) и тематическими разделами.
// Полностью независим от питомцев-грувиков (petsvc). Пересылка поста в
// мессенджер — единственный межсервисный вызов, gRPC к msgsvc
// (CreatePostMessage). Схему таблиц ведёт migrate-контейнер (goose).
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/portal/* (за nginx) и узкий
// gRPC (GRPC_ADDR) — только контракт владельца файлов для раздела
// «Настройки → Хранилище» (его зовёт биллинг).
//
// Сокет-события клиентам — Redis-канал gw2:portal:events (доставляет
// gatewaysvc). Вложения — общий uploads-том/S3 (pkg/storage, префикс "portal").
package main

import (
	"net"
	"os"

	googrpc "google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
	"github.com/DmitriyODS/gw2/back-go/pkg/companydata"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/pkg/storage"
	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
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
	// Лимиты тарифа и учёт занятого места — gRPC billingsvc. Пустой адрес
	// выключает проверки, недоступный биллинг их не блокирует (fail-open).
	billing, err := billingclient.New(bootstrap.Env("BILLING_GRPC_ADDR", ""), log)
	if err != nil {
		log.Error("billing.dial_failed", "error", err)
		os.Exit(1)
	}
	defer billing.Close()
	fileStore := records.NewFileStore(storage.FromEnv(log, uploadFolder), "portal").
		WithQuota(billing, "portal")

	svc := service.New(service.Deps{
		Repo:      repo,
		Files:     fileStore,
		Bus:       events.NewPublisher(rdb, log, "gw2:portal:events"),
		Messenger: messenger,
		Log:       log,
	})
	svc.WithBilling(billing)

	eps := endpoint.New(svc)

	// Приём файлов частями — общий движок платформы.
	uploads := chunkupload.New(pool, fileStore, "portal", log)
	go uploads.Sweep(ctx)

	httpServer := httptransport.NewServer(eps, svc, users, uploads, verifier, log)

	// gRPC — единственный: биллинг спрашивает про файлы для раздела
	// «Настройки → Хранилище» (вложения публикаций).
	grpcAddr := bootstrap.Env("GRPC_ADDR", ":9102")
	grpcServer := googrpc.NewServer(googrpc.MaxRecvMsgSize(companydata.MaxMessageBytes))
	storagefiles.Register(grpcServer, svc)
	// Перенос компании: архив собирает authsvc, портал отдаёт свою часть.
	companydata.Register(grpcServer, repo)
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc.listen_failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	log.Info("listening", "http", httpAddr, "grpc", grpcAddr)
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
		bootstrap.Component{
			Name: "grpc",
			Run:  func() error { return grpcServer.Serve(listener) },
			Stop: grpcServer.GracefulStop,
		},
	)
}
