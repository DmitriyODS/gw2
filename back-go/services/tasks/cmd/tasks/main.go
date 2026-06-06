package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DmitriyODS/gw2/back-go/services/tasks/internal/bootstrap"
	"github.com/DmitriyODS/gw2/back-go/services/tasks/internal/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", slog.String("err", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.New(rootCtx, cfg)
	if err != nil {
		return err
	}

	app.Logger.Info("starting service",
		slog.String("http_addr", cfg.HTTPAddr),
		slog.String("grpc_addr", cfg.GRPCAddr),
	)

	httpErr := make(chan error, 1)
	grpcErr := make(chan error, 1)

	go func() { httpErr <- app.HTTPApp.Listen(cfg.HTTPAddr) }()
	go func() { grpcErr <- app.GRPCServer.Serve(app.ServeGRPC()) }()

	select {
	case <-rootCtx.Done():
		app.Logger.Info("shutdown signal received")
	case err := <-httpErr:
		if err != nil {
			return err
		}
	case err := <-grpcErr:
		if err != nil {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.HTTPApp.ShutdownWithContext(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		app.Logger.Warn("http shutdown", slog.String("err", err.Error()))
	}
	app.GRPCServer.GracefulStop()
	return nil
}
