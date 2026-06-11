// Package endpoint — go-kit endpoints поверх сервисного слоя.
//
// Каждый use-case обёрнут в endpoint.Endpoint: транспорты (gRPC, Fiber)
// декодируют свои запросы в dto, зовут endpoint и кодируют ответ обратно.
// Бизнес-ошибки (*domain.Error) пролетают через error-канал endpoint'а и
// мапятся транспортом: gRPC — в поле Error ответа, HTTP — в статус + JSON.
package endpoint

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/service"
)

// Endpoints — все use-cases сервиса звонков.
type Endpoints struct {
	StartCall    endpoint.Endpoint
	InviteToCall endpoint.Endpoint
	AcceptCall   endpoint.Endpoint
	DeclineCall  endpoint.Endpoint
	LeaveCall    endpoint.Endpoint
	EndCall      endpoint.Endpoint

	RejoinToken endpoint.Endpoint
	ActiveCall  endpoint.Endpoint
	History     endpoint.Endpoint
	JoinInfo    endpoint.Endpoint
	JoinByCode  endpoint.Endpoint
}

func New(svc service.CallService) Endpoints {
	return Endpoints{
		StartCall: func(ctx context.Context, request any) (any, error) {
			return svc.StartCall(ctx, request.(dto.StartCallRequest))
		},
		InviteToCall: func(ctx context.Context, request any) (any, error) {
			return svc.InviteToCall(ctx, request.(dto.InviteRequest))
		},
		AcceptCall: func(ctx context.Context, request any) (any, error) {
			return svc.AcceptCall(ctx, request.(dto.AcceptRequest))
		},
		DeclineCall: func(ctx context.Context, request any) (any, error) {
			return svc.DeclineCall(ctx, request.(dto.HangupRequest))
		},
		LeaveCall: func(ctx context.Context, request any) (any, error) {
			return svc.LeaveCall(ctx, request.(dto.HangupRequest))
		},
		EndCall: func(ctx context.Context, request any) (any, error) {
			return svc.EndCall(ctx, request.(dto.HangupRequest))
		},
		RejoinToken: func(ctx context.Context, request any) (any, error) {
			req := request.(RejoinTokenRequest)
			return svc.RejoinToken(ctx, req.CallID, req.UserID)
		},
		ActiveCall: func(ctx context.Context, request any) (any, error) {
			return svc.ActiveCall(ctx, request.(int64))
		},
		History: func(ctx context.Context, request any) (any, error) {
			req := request.(HistoryRequest)
			return svc.History(ctx, req.UserID, req.Limit)
		},
		JoinInfo: func(ctx context.Context, request any) (any, error) {
			return svc.JoinInfo(ctx, request.(string))
		},
		JoinByCode: func(ctx context.Context, request any) (any, error) {
			return svc.JoinByCode(ctx, request.(dto.JoinByCodeRequest))
		},
	}
}

// RejoinTokenRequest / HistoryRequest — транспорт-независимые запросы
// REST-эндпоинтов, у которых нет своего dto.
type RejoinTokenRequest struct {
	CallID int64
	UserID int64
}

type HistoryRequest struct {
	UserID int64
	Limit  int
}
