// registrysvc — микросервис реестров Groove Work.
//
// Владеет реестрами (настраиваемыми таблицами-справочниками): их структурой
// (поля разных типов с раскладкой карточки), записями, учётом выдач и шарингом.
// Реестр принадлежит ЧЕЛОВЕКУ, а коллеги и компании получают доступ трёх
// уровней адресно. Схему таблиц ведёт migrate-контейнер (goose, back-go/migrate).
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/registries/* (за nginx).
// Сокет-события клиентам — Redis-канал gw2:registry:events (доставляет
// gatewaysvc). Загруженные файлы/картинки — общий uploads-том (раздаёт nginx
// /uploads/). Межсервисных вызовов нет: авторизация локальная (PASETO).
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
	"github.com/DmitriyODS/gw2/back-go/registry/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/service"
	httptransport "github.com/DmitriyODS/gw2/back-go/registry/internal/transport/http"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	uploadFolder := bootstrap.Env("UPLOAD_FOLDER", "../../uploads")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8099")

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
	fileStore := records.NewFileStore(storage.FromEnv(log, uploadFolder), "registry").
		WithQuota(billing, "registry")

	// Приём больших файлов частями — общий движок платформы: сессии в БД,
	// части объектами в хранилище (переживает несколько инстансов сервиса).
	uploads := chunkupload.New(pool, fileStore, "registry", log)

	svc := service.New(service.Deps{
		Repo:    repo,
		Users:   users,
		Files:   fileStore,
		Bus:     events.NewPublisher(rdb, log, "gw2:registry:events"),
		Uploads: uploads,
		Log:     log,
	})
	svc.WithBilling(billing)

	// Брошенные загрузки (закрытая вкладка) не должны копить части в хранилище.
	go uploads.Sweep(ctx)

	httpServer := httptransport.NewServer(svc, users, verifier, log)

	// gRPC — единственный: биллинг спрашивает про файлы для раздела
	// «Настройки → Хранилище» (файлы записей).
	grpcAddr := bootstrap.Env("GRPC_ADDR", ":9099")
	grpcServer := googrpc.NewServer(googrpc.MaxRecvMsgSize(companydata.MaxMessageBytes))
	storagefiles.Register(grpcServer, svc)
	// Перенос компании: архив собирает authsvc, раздел отдаёт свою часть.
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
