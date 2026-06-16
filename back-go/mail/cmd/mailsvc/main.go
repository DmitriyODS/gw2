// mailsvc — микросервис рассылки писем Groove Work.
//
// Транспорты:
//   - gRPC (GRPC_ADDR) — Send(шаблон + параметры): зовёт authsvc для писем
//     подтверждения email;
//   - HTTP/Fiber (HTTP_ADDR) — только /healthz для docker healthcheck.
//
// Сервис stateless: без PostgreSQL/Redis. Доставка — SMTP (env SMTP_*).
package main

import (
	"net"
	"os"

	googrpc "google.golang.org/grpc"

	"github.com/DmitriyODS/gw2/back-go/mail/internal/service"
	"github.com/DmitriyODS/gw2/back-go/mail/internal/smtp"
	grpctransport "github.com/DmitriyODS/gw2/back-go/mail/internal/transport/grpc"
	httptransport "github.com/DmitriyODS/gw2/back-go/mail/internal/transport/http"
	"github.com/DmitriyODS/gw2/back-go/pkg/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/mailpb"
)

func main() {
	log := bootstrap.Logger()

	grpcAddr := bootstrap.Env("GRPC_ADDR", ":9098")
	httpAddr := bootstrap.Env("HTTP_ADDR", ":8098")

	smtpCli := smtp.New(smtp.Config{
		Host:     bootstrap.MustEnv(log, "SMTP_HOST"),
		Port:     bootstrap.Env("SMTP_PORT", "587"),
		User:     os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     bootstrap.MustEnv(log, "SMTP_FROM"),
		FromName: bootstrap.Env("SMTP_FROM_NAME", "Groove Work"),
		TLSMode:  bootstrap.Env("SMTP_TLS", "starttls"),
	}, log)

	svc := service.New(smtpCli, log)

	grpcServer := googrpc.NewServer()
	mailpb.RegisterMailServiceServer(grpcServer, grpctransport.NewServer(svc, log))
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("grpc.listen_failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	httpServer := httptransport.NewServer()

	ctx, stop := bootstrap.SignalContext()
	defer stop()

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
