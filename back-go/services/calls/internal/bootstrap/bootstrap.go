// Package bootstrap собирает граф зависимостей calls-сервиса.
// На Этапе 1 — только REST /v1/health и gRPC stub. Реальные репозитории,
// конкретная бизнес-логика добавляются на Этапе 3.
package bootstrap

import (
	"context"
	"log/slog"
	"net"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"google.golang.org/grpc"

	callsv1 "github.com/DmitriyODS/gw2/back-go/gen/proto/calls/v1"
	"github.com/DmitriyODS/gw2/back-go/services/calls/internal/config"
	"github.com/DmitriyODS/gw2/back-go/shared/pkg/logger"
)

type App struct {
	Logger     *slog.Logger
	Config     config.Config
	HTTPApp    *fiber.App
	GRPCServer *grpc.Server
	grpcLis    net.Listener
}

func New(_ context.Context, cfg config.Config) (*App, error) {
	log := logger.New(cfg.LogLevel).With(slog.String("service", cfg.ServiceName))

	httpApp := fiber.New(fiber.Config{
		AppName:      cfg.ServiceName,
		ServerHeader: "groovework-" + cfg.ServiceName,
	})
	httpApp.Use(requestid.New())
	httpApp.Use(recover.New())

	httpApp.Get("/v1/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": cfg.ServiceName,
		})
	})

	grpcSrv := grpc.NewServer()
	callsv1.RegisterCallsServiceServer(grpcSrv, &callsStubServer{})

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return nil, err
	}

	return &App{
		Logger:     log,
		Config:     cfg,
		HTTPApp:    httpApp,
		GRPCServer: grpcSrv,
		grpcLis:    lis,
	}, nil
}

func (a *App) ServeGRPC() net.Listener { return a.grpcLis }

type callsStubServer struct {
	callsv1.UnimplementedCallsServiceServer
}
