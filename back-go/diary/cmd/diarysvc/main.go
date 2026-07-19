// diarysvc — микросервис ежедневников Groove Work.
//
// Владеет личными ежедневниками пользователей: списками записей-задач,
// привязанных к дню (с опциональным временем начала/конца, описанием, отметкой
// «выполнено» → архив и связью с задачей tasksvc). Ежедневник принадлежит
// одному пользователю и не зависит от компании (кросс-компанийный, как
// мессенджер); другим он доступен только через шаринг (read-only) — публичной
// ссылкой или адресно. Схему таблиц ведёт migrate-контейнер (goose).
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — REST /api/diaries/* (за nginx).
// Сокет-события клиентам — Redis-канал gw2:diary:events (доставляет
// gatewaysvc). Межсервисных вызовов нет: авторизация локальная (PASETO).
package main

import (
	"net"
	"os"

	googrpc "google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/diary/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/diary/internal/service"
	grpctransport "github.com/DmitriyODS/gw2/back-go/diary/internal/transport/grpc"
	httptransport "github.com/DmitriyODS/gw2/back-go/diary/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/diarypb"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8101")
	grpcAddr := bootstrap.Env("GRPC_ADDR", ":9101")

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
		Users: users,
		Bus:   events.NewPublisher(rdb, log, "gw2:diary:events"),
		Log:   log,
	})
	eps := endpoint.New(svc)

	httpServer := httptransport.NewServer(eps, users, verifier, log)

	// gRPC — голосовые операции навыка Алисы (зовёт alicesvc).
	grpcServer := googrpc.NewServer()
	diarypb.RegisterDiaryServiceServer(grpcServer, grpctransport.NewServer(svc))
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
