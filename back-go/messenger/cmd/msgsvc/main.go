// msgsvc — микросервис мессенджера Groove Work.
//
// Транспорты:
//   - HTTP/Fiber (HTTP_ADDR) — REST /api/messenger/* (за nginx; кроме
//     exact /api/messenger/presence — он остаётся во Flask);
//   - gRPC (GRPC_ADDR) — плашки звонков (Flask) и бот Грувика (groove).
//
// Зависимости: общая PostgreSQL платформы (схему ведёт Alembic), Redis
// (публикация событий Socket.IO для Flask-моста, канал gw2:messenger:events),
// общий uploads-volume (файлы вложений).
package main

import (
	"net"
	"os"

	googrpc "google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/messengerpb"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/clients"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/files"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/service"
	grpctransport "github.com/DmitriyODS/gw2/back-go/messenger/internal/transport/grpc"
	httptransport "github.com/DmitriyODS/gw2/back-go/messenger/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	uploadFolder := bootstrap.Env("UPLOAD_FOLDER", "/app/uploads")
	grooveAddr := bootstrap.Env("GROOVE_GRPC_ADDR", "localhost:9094")
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
	store := files.NewStore(uploadFolder, log)
	pub := events.NewPublisher(rdb, log, "gw2:messenger:events")
	// Ответ Грувика в pet-чате генерирует groovesvc (gRPC, fire-and-forget).
	groove, err := clients.NewGroove(grooveAddr, log)
	if err != nil {
		log.Error("groove_grpc.bad_addr", "error", err)
		os.Exit(1)
	}
	defer groove.Close()
	svc := service.New(repo, users, store, pub, groove, log)
	eps := endpoint.New(svc)

	grpcAddr := bootstrap.Env("GRPC_ADDR", ":9092")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8092")

	grpcServer := googrpc.NewServer()
	messengerpb.RegisterMessengerServiceServer(grpcServer, grpctransport.NewServer(eps))
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc.listen_failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	httpServer := httptransport.NewServer(eps, users, verifier, log)

	log.Info("listening", "grpc", grpcAddr, "http", httpAddr)
	bootstrap.Run(ctx, log,
		bootstrap.Component{
			Name: "grpc",
			Run:  func() error { return grpcServer.Serve(listener) },
			Stop: grpcServer.GracefulStop,
		},
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
