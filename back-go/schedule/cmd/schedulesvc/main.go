// schedulesvc — микросервис расписаний Groove Work.
//
// Владеет регулярными расписаниями пользователей: занятия стоят на дне недели и
// времени, а какая сейчас неделя цикла (1..4), считается из даты — вхождений на
// конкретные даты в базе нет. Расписание принадлежит одному человеку и не
// зависит от компании; другим оно доступно ТОЛЬКО НА ЧТЕНИЕ — публичной ссылкой
// или адресно. Схему таблиц ведёт migrate-контейнер (goose).
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/schedules/* (за nginx).
// Сокет-события клиентам — Redis-канал gw2:schedule:events (доставляет
// gatewaysvc). Межсервисных вызовов нет: авторизация локальная (PASETO),
// файлов раздел не держит.
package main

import (
	"os"

	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/schedule/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/schedule/internal/service"
	httptransport "github.com/DmitriyODS/gw2/back-go/schedule/internal/transport/http"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8110")

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
		Repo:  postgres.NewRepo(pool),
		Users: users,
		Bus:   events.NewPublisher(rdb, log, "gw2:schedule:events"),
		Log:   log,
	})

	httpServer := httptransport.NewServer(svc, users, verifier, log)

	log.Info("listening", "http", httpAddr)
	bootstrap.Run(ctx, log, bootstrap.Component{
		Name: "http",
		Run:  func() error { return httpServer.Listen(httpAddr) },
		Stop: func() {
			if err := httpServer.Shutdown(); err != nil {
				log.Warn("http.shutdown_failed", "error", err)
			}
		},
	})
}
