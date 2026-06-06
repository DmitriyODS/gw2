// Package bootstrap собирает граф зависимостей auth-сервиса.
// На Этапе 1 здесь только REST /v1/health и gRPC stub.
// Реальные репозитории, токены и outbox добавляются на Этапе 2.
package bootstrap

import (
	"context"
	"log/slog"
	"net"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"google.golang.org/grpc"

	authv1 "github.com/DmitriyODS/gw2/back-go/gen/proto/auth/v1"
	"github.com/DmitriyODS/gw2/back-go/services/auth/internal/config"
	"github.com/DmitriyODS/gw2/back-go/shared/pkg/logger"
)

// App — собранные зависимости сервиса. main.go запускает HTTPApp.Listen и
// GRPCServer.Serve, по сигналу делает graceful shutdown.
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
	authv1.RegisterAuthServiceServer(grpcSrv, &authStubServer{})

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

// ServeGRPC отдаёт listener, на котором GRPCServer должен слушать.
// main.go вызывает GRPCServer.Serve(app.ServeGRPC()).
func (a *App) ServeGRPC() net.Listener { return a.grpcLis }

// authStubServer — заглушка до Этапа 2. Все методы возвращают Unimplemented
// благодаря встроенному UnimplementedAuthServiceServer.
type authStubServer struct {
	authv1.UnimplementedAuthServiceServer
}
