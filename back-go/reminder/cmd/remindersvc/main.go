// remindersvc — микросервис напоминаний Groove Work.
//
// Владеет личными напоминаниями пользователей: разовыми и повторяющимися
// (ежедневно / по рабочим дням / по дням недели / ежемесячно / ежегодно), со
// свободным текстом либо привязкой к записи ежедневника или календаря.
// Напоминание принадлежит одному пользователю и не зависит от компании
// (кросс-компанийное, как ежедневник). Схему таблиц ведёт migrate-контейнер
// (goose).
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/reminders/* (за nginx) плюс
// внутренний планировщик: раз в полминуты забирает наступившие сроки и шлёт
// reminder:fire в Redis-канал gw2:reminder:events. Дальше событие расходится
// само: gatewaysvc доставляет его открытым вкладкам (тост, системное и
// десктопное уведомление), pushsvc — FCM-пушем тем, кто офлайн. Межсервисных
// вызовов нет: авторизация локальная (PASETO).
package main

import (
	"os"

	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
	"github.com/DmitriyODS/gw2/back-go/reminder/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/reminder/internal/service"
	httptransport "github.com/DmitriyODS/gw2/back-go/reminder/internal/transport/http"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8106")

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
		Repo: postgres.NewRepo(pool),
		Bus:  events.NewPublisher(rdb, log, "gw2:reminder:events"),
		Log:  log,
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
		bootstrap.Component{
			Name: "scheduler",
			Run:  func() error { return svc.RunScheduler(ctx) },
			Stop: stop,
		},
	)
}
