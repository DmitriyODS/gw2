// Package grpc — gRPC-транспорт petsvc: хуки доменных событий tasksvc
// (юниты/задачи).
//
// Семантика fire-and-forget: вызывающий не ждёт результата геймификации,
// поэтому ошибки здесь только логируются, транспорт всегда отвечает OK.
package grpc

import (
	"context"
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/petspb"
)

type Server struct {
	petspb.UnimplementedPetsServiceServer
	svc *service.Service
	log *slog.Logger
}

func NewServer(svc *service.Service, log *slog.Logger) *Server {
	return &Server{svc: svc, log: log}
}

func (s *Server) OnUnitStarted(ctx context.Context, req *petspb.UnitStartedRequest) (*petspb.HookResponse, error) {
	s.svc.OnUnitStarted(ctx, service.UnitHook{
		CompanyID: req.GetCompanyId(),
		UserID:    req.GetUserId(),
		UnitID:    req.GetUnitId(),
		UnitName:  req.GetUnitName(),
		TaskID:    req.GetTaskId(),
		TaskName:  req.GetTaskName(),
	})
	return &petspb.HookResponse{}, nil
}

func (s *Server) OnUnitStopped(ctx context.Context, req *petspb.UnitStoppedRequest) (*petspb.HookResponse, error) {
	s.svc.OnUnitStopped(ctx, service.UnitHook{
		CompanyID: req.GetCompanyId(),
		UserID:    req.GetUserId(),
		UnitID:    req.GetUnitId(),
		UnitName:  req.GetUnitName(),
		TaskID:    req.GetTaskId(),
		TaskName:  req.GetTaskName(),
		Minutes:   int(req.GetMinutes()),
	})
	return &petspb.HookResponse{}, nil
}

func (s *Server) OnTaskClosed(ctx context.Context, req *petspb.TaskClosedRequest) (*petspb.HookResponse, error) {
	s.svc.OnTaskClosed(ctx, req.GetCompanyId(), req.GetHeroUserId(),
		req.GetTaskId(), req.GetTaskName())
	return &petspb.HookResponse{}, nil
}
