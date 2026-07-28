// calendarsvc — микросервис календарей Groove Work.
//
// Владеет календарями компаний (настраиваемыми списками записей, привязанных к
// дате/времени): их структурой (поля разных типов с раскладкой карточки, плюс
// встроенное обязательное поле «Дата и время») и записями. Структуру правит
// администратор компании, записи — любой её участник. Схему таблиц ведёт
// migrate-контейнер (goose, back-go/migrate).
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/calendars/* (за nginx).
// Сокет-события клиентам — Redis-канал gw2:calendar:events (доставляет
// gatewaysvc). Загруженные файлы/картинки — общий uploads-том (раздаёт nginx
// /uploads/). Межсервисных вызовов нет: авторизация локальная (PASETO).
package main

import (
	"os"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/calendar/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/calendar/internal/service"
	httptransport "github.com/DmitriyODS/gw2/back-go/calendar/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/pkg/storage"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	uploadFolder := bootstrap.Env("UPLOAD_FOLDER", "../../uploads")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8100")

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
	fileStore := records.NewFileStore(storage.FromEnv(log, uploadFolder), "calendar").
		WithQuota(billing, "calendar")

	svc := service.New(service.Deps{
		Repo:  repo,
		Files: fileStore,
		Bus:   events.NewPublisher(rdb, log, "gw2:calendar:events"),
		Log:   log,
	})
	svc.WithBilling(billing)

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
