// notesvc — микросервис заметок Groove Work.
//
// Владеет личными заметками пользователей: rich-текст (документ TipTap),
// группы-фильтры и публичные ссылки в режимах «чтение» / «чтение и
// редактирование». Заметка принадлежит одному пользователю и не зависит от
// компании (кросс-компанийная, как ежедневник). Схему таблиц ведёт
// migrate-контейнер (goose).
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/notes/* (за nginx).
// Сокет-события клиентам — Redis-канал gw2:notes:events (доставляет
// gatewaysvc). Картинки редактора — pkg/storage (local-том в dev, S3 в prod),
// отдаются по /uploads/. Межсервисных вызовов нет: авторизация локальная (PASETO).
package main

import (
	"os"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/notes/internal/repository/postgres"
	redisrepo "github.com/DmitriyODS/gw2/back-go/notes/internal/repository/redis"
	"github.com/DmitriyODS/gw2/back-go/notes/internal/service"
	httptransport "github.com/DmitriyODS/gw2/back-go/notes/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/pkg/storage"
)

// sharedWriteLimit — троттлинг анонимных правок по коду edit-ссылки (в минуту).
const sharedWriteLimit = 30

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	uploadFolder := bootstrap.Env("UPLOAD_FOLDER", "../../uploads")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8103")

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
		Repo:    repo,
		Files:   records.NewFileStore(storage.FromEnv(log, uploadFolder), "notes"),
		Bus:     events.NewPublisher(rdb, log, "gw2:notes:events"),
		Limiter: redisrepo.NewWriteLimiter(rdb, sharedWriteLimit),
		Log:     log,
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
