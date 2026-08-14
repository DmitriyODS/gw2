// drivesvc — микросервис «Диск» Groove Work.
//
// Владеет личным файловым хранилищем пользователей: папки, файлы, корзина,
// избранное и шаринг (публичные ссылки + адресный доступ человеку или
// компании, каскадом по папке). Диск принадлежит ОДНОМУ пользователю и не
// зависит от компании — как заметки и доски. Схему таблиц ведёт
// migrate-контейнер (goose).
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/drive/* (за nginx) и узкий
// gRPC (GRPC_ADDR) — контракт владельца файлов для раздела
// «Настройки → Хранилище» (его зовёт биллинг).
//
// Сокет-события клиентам — Redis-канал gw2:drive:events (доставляет
// gatewaysvc). Сами файлы — pkg/storage (local-том в dev, S3 в prod), место
// считается в квоте владельца (billingsvc).
package main

import (
	"net"
	"os"

	googrpc "google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/drive/internal/service"
	httptransport "github.com/DmitriyODS/gw2/back-go/drive/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/pkg/storage"
	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	uploadFolder := bootstrap.Env("UPLOAD_FOLDER", "../../uploads")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8108")
	grpcAddr := bootstrap.Env("GRPC_ADDR", ":9108")

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

	repo := postgres.NewRepo(pool)
	users := postgres.NewUserReader(pool)

	// Учёт занятого места — gRPC billingsvc: файл сверх квоты не сохраняется
	// вовсе. Пустой адрес выключает учёт, недоступный биллинг его не
	// блокирует (fail-open).
	billing, err := billingclient.New(bootstrap.Env("BILLING_GRPC_ADDR", ""), log)
	if err != nil {
		log.Error("billing.dial_failed", "error", err)
		os.Exit(1)
	}
	defer billing.Close()
	fileStore := records.NewFileStore(storage.FromEnv(log, uploadFolder), "drive").
		WithQuota(billing, "drive")

	svc := service.New(service.Deps{
		Repo:  repo,
		Users: users,
		Files: fileStore,
		Bus:   events.NewPublisher(rdb, log, "gw2:drive:events"),
		Log:   log,
	})

	// Приём больших файлов частями — общий движок платформы: сессии в БД,
	// части объектами в хранилище (переживает несколько инстансов сервиса).
	uploads := chunkupload.New(pool, fileStore, "drive", log)

	httpServer := httptransport.NewServer(svc, users, uploads, verifier, log)

	grpcServer := googrpc.NewServer()
	storagefiles.Register(grpcServer, svc)
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc.listen_failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	// Корзина чистится сама: пролежавшее дольше срока удаляется вместе с
	// объектами, и место возвращается владельцу.
	go svc.RunTrashCleaner(ctx)
	// Брошенные загрузки по частям: закрытая вкладка не должна оставлять
	// куски в хранилище.
	go uploads.Sweep(ctx)

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
