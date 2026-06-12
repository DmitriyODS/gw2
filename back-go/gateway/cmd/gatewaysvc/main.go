// gatewaysvc — realtime-шлюз Groove Work (наследник Flask-SocketIO).
//
// Лёгкий WS-шлюз: PASETO-handshake, комнаты all/user_{id}, presence в Redis
// (visibility + heartbeat + sweeper, last_seen_at в users) и ринг-фаза
// звонков (WS-команды call:* → gRPC callsvc). Сам подписывается на все
// Redis-каналы gw2:<svc>:events (общий envelope) и доставляет события
// клиентам — отдельных мостов больше нет.
//
// Транспорт: HTTP/Fiber (HTTP_ADDR) — /ws (WebSocket), exact
// /api/messenger/presence (REST, за nginx) и /healthz.
package main

import (
	"os"

	"github.com/DmitriyODS/gw2/back-go/gateway/internal/bridge"
	"github.com/DmitriyODS/gw2/back-go/gateway/internal/hub"
	"github.com/DmitriyODS/gw2/back-go/gateway/internal/presence"
	"github.com/DmitriyODS/gw2/back-go/gateway/internal/repository/postgres"
	"github.com/DmitriyODS/gw2/back-go/gateway/internal/ring"
	httptransport "github.com/DmitriyODS/gw2/back-go/gateway/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

func main() {
	log := bootstrap.Logger()

	dbURL := bootstrap.Env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")
	redisURL := bootstrap.Env("REDIS_URL", "redis://localhost:6379/0")
	callsAddr := bootstrap.Env("CALLS_GRPC_ADDR", "localhost:9090")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8096")

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

	calls, conn, err := ring.Dial(callsAddr)
	if err != nil {
		log.Error("calls.client_failed", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	h := hub.New()
	bus := events.NewPublisher(rdb, log, "gw2:gateway:events")
	pres := presence.New(rdb, presence.PGLastSeen{Pool: pool}, bus, log)
	rng := ring.New(calls, bus, log)
	users := postgres.NewUserReader(pool)
	br := bridge.New(rdb, h, log)

	server := httptransport.NewServer(httptransport.Deps{
		Hub:      h,
		Presence: pres,
		Ring:     rng,
		Verifier: verifier,
		Auth:     users.AuthInfo,
		Log:      log,
	})

	log.Info("listening", "http", httpAddr)
	bootstrap.Run(ctx, log,
		bootstrap.Component{
			Name: "bridge",
			Run: func() error {
				br.Run(ctx)
				return nil
			},
			Stop: func() {},
		},
		bootstrap.Component{
			Name: "presence-sweeper",
			Run: func() error {
				pres.RunSweeper(ctx)
				return nil
			},
			Stop: func() {},
		},
		bootstrap.Component{
			Name: "http",
			Run:  func() error { return server.Listen(httpAddr) },
			Stop: func() {
				if err := server.Shutdown(); err != nil {
					log.Warn("http.shutdown_failed", "error", err)
				}
			},
		},
	)
}
