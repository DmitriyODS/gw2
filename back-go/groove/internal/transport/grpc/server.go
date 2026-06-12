// Package grpc — gRPC-транспорт groovesvc: хуки доменных событий других
// сервисов (Flask — юниты/задачи, msgsvc — pet-чат).
//
// Семантика fire-and-forget: вызывающий не ждёт результата геймификации,
// поэтому ошибки здесь только логируются, транспорт всегда отвечает OK.
package grpc

import (
	"context"
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/groovepb"
)

type Server struct {
	groovepb.UnimplementedGrooveServiceServer
	svc *service.Service
	log *slog.Logger
}

func NewServer(svc *service.Service, log *slog.Logger) *Server {
	return &Server{svc: svc, log: log}
}

func (s *Server) OnUnitStarted(ctx context.Context, req *groovepb.UnitStartedRequest) (*groovepb.HookResponse, error) {
	s.svc.OnUnitStarted(ctx, service.UnitHook{
		CompanyID: req.GetCompanyId(),
		UserID:    req.GetUserId(),
		UnitID:    req.GetUnitId(),
		UnitName:  req.GetUnitName(),
		TaskID:    req.GetTaskId(),
		TaskName:  req.GetTaskName(),
	})
	return &groovepb.HookResponse{}, nil
}

func (s *Server) OnUnitStopped(ctx context.Context, req *groovepb.UnitStoppedRequest) (*groovepb.HookResponse, error) {
	s.svc.OnUnitStopped(ctx, service.UnitHook{
		CompanyID: req.GetCompanyId(),
		UserID:    req.GetUserId(),
		UnitID:    req.GetUnitId(),
		UnitName:  req.GetUnitName(),
		TaskID:    req.GetTaskId(),
		TaskName:  req.GetTaskName(),
		Minutes:   int(req.GetMinutes()),
	})
	return &groovepb.HookResponse{}, nil
}

func (s *Server) OnTaskClosed(ctx context.Context, req *groovepb.TaskClosedRequest) (*groovepb.HookResponse, error) {
	s.svc.OnTaskClosed(ctx, req.GetCompanyId(), req.GetHeroUserId(),
		req.GetTaskId(), req.GetTaskName())
	return &groovepb.HookResponse{}, nil
}

func (s *Server) OnPetMessage(ctx context.Context, req *groovepb.PetMessageRequest) (*groovepb.HookResponse, error) {
	// Ответ Грувика асинхронный (LLM может думать десятки секунд).
	s.svc.SchedulePetReply(req.GetConversationId())
	return &groovepb.HookResponse{}, nil
}
