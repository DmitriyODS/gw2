// registrysvc — микросервис реестров Groove Work.
//
// Владеет реестрами компаний (настраиваемыми таблицами-справочниками): их
// структурой (поля разных типов с раскладкой карточки) и записями. Структуру
// правит администратор компании, записи — любой её участник. Схему таблиц ведёт
// migrate-контейнер (goose, back-go/migrate).
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/registries/* (за nginx).
// Сокет-события клиентам — Redis-канал gw2:registry:events (доставляет
// gatewaysvc). Загруженные файлы/картинки — общий uploads-том (раздаёт nginx
// /uploads/). Межсервисных вызовов нет: авторизация локальная (PASETO).
package main

import (
	"os"

	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/filestore"
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
	svc := service.New(service.Deps{
		Repo:  repo,
		Files: filestore.New(uploadFolder),
		Bus:   events.NewPublisher(rdb, log, "gw2:registry:events"),
		Log:   log,
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
