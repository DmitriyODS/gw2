// boardsvc — микросервис досок Groove Work.
//
// Владеет личными досками пользователей: сцена рисования (объекты холста в
// JSONB), иерархические папки, теги-метки, публичные ссылки в режимах «чтение» /
// «чтение и редактирование» и адресный шаринг пользователям и компаниям. Доска
// принадлежит одному пользователю и не зависит от компании (кросс-компанийная,
// как заметка). Схему таблиц ведёт migrate-контейнер (goose).
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/boards/* (за nginx).
// Сокет-события клиентам — Redis-канал gw2:board:events (доставляет
// gatewaysvc). Картинки холста и превью досок — pkg/storage (local-том в dev,
// S3 в prod), отдаются по /uploads/. Межсервисных вызовов нет: авторизация
// локальная (PASETO).
package main

import (
	"os"

	"github.com/DmitriyODS/gw2/back-go/board/internal/repository/postgres"
	redisrepo "github.com/DmitriyODS/gw2/back-go/board/internal/repository/redis"
	"github.com/DmitriyODS/gw2/back-go/board/internal/service"
	httptransport "github.com/DmitriyODS/gw2/back-go/board/internal/transport/http"
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
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8105")

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

	users := postgres.NewUserReader(pool)
	svc := service.New(service.Deps{
		Repo:    postgres.NewRepo(pool),
		Users:   users,
		Files:   records.NewFileStore(storage.FromEnv(log, uploadFolder), "boards"),
		Bus:     events.NewPublisher(rdb, log, "gw2:board:events"),
		Limiter: redisrepo.NewWriteLimiter(rdb, sharedWriteLimit),
		Log:     log,
	})
	httpServer := httptransport.NewServer(svc, users, verifier, log)

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
