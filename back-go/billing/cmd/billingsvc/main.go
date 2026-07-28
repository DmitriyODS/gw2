// billingsvc — подписки, магазин и платежи Groove Work.
//
// Владеет тарифами и их ценами, подписками пользователей и докупками, заказами
// и платежами, промокодами, товарами витрины (платформенными и авторскими),
// балансом токенов доступа к ИИ и учётом занятого хранилища.
//
// Скоуп: подписка принадлежит ПОЛЬЗОВАТЕЛЮ, компания наследует тариф своего
// СОЗДАТЕЛЯ. Лимиты конечны и живут в коде (domain.PlanLimits), цены правит
// супер-админ в разделе «Аудит платформы».
//
// Транспорт: HTTP/Fiber (REST /api/billing/* за nginx, вебхук платёжного шлюза
// публичный) и gRPC — контракт для остальных сервисов: какие лимиты действуют,
// влезает ли файл в хранилище, есть ли токены на обращение к модели. Плюс
// внутренний планировщик продлений.
package main

import (
	"net"
	"os"

	googrpc "google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/payments"
	"github.com/DmitriyODS/gw2/back-go/billing/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/billing/internal/service"
	grpctransport "github.com/DmitriyODS/gw2/back-go/billing/internal/transport/grpc"
	httptransport "github.com/DmitriyODS/gw2/back-go/billing/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/billingpb"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8107")
	grpcAddr := bootstrap.Env("GRPC_ADDR", ":9107")

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
	identity := postgres.NewIdentity(pool)

	// Платёжный шлюз: код провайдера хранится в настройках платформы, но выбор
	// реализации делается на старте — смена провайдера это перезапуск сервиса.
	provider := payments.New(bootstrap.Env("PAYMENT_PROVIDER", "manual"))

	svc := service.New(service.Deps{
		Catalog: repo, Subs: repo, Orders: repo, Promos: repo, Products: repo,
		AI: repo, Storage: repo, Settings: repo, Audit: repo,
		Identity: identity,
		Provider: provider,
		Bus:      events.NewPublisher(rdb, log, "gw2:billing:events"),
		Log:      log,
	})

	httpServer := httptransport.NewServer(svc, identity, verifier, log)

	grpcServer := googrpc.NewServer()
	billingpb.RegisterBillingServiceServer(grpcServer, grpctransport.NewServer(svc))
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc.listen_failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	go svc.RunScheduler(ctx)

	log.Info("listening", "http", httpAddr, "grpc", grpcAddr, "payments", provider.Name())
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
