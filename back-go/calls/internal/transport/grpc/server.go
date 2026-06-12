// Package grpc — gRPC-транспорт ринг-фазы (его дёргает Flask-шлюз из
// Socket.IO-хендлеров). Бизнес-ошибки уезжают полем Error в ответе (transport
// всегда отвечает OK) — Flask транслирует их в call:error как раньше.
package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/callspb"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/calls/internal/endpoint"
)

type Server struct {
	callspb.UnimplementedCallServiceServer
	eps endpoint.Endpoints
}

func NewServer(eps endpoint.Endpoints) *Server {
	return &Server{eps: eps}
}

func (s *Server) StartCall(ctx context.Context, req *callspb.StartCallRequest) (*callspb.StartCallResponse, error) {
	resp, err := s.eps.StartCall(ctx, dto.StartCallRequest{
		InitiatorID:    req.GetInitiatorId(),
		InviteeIDs:     req.GetInviteeIds(),
		Media:          req.GetMedia(),
		ConversationID: req.GetConversationId(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &callspb.StartCallResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(*dto.StartCallResponse)
	return &callspb.StartCallResponse{
		Call:    pbCall(r.Call),
		Livekit: pbLivekit(r.Livekit),
	}, nil
}

func (s *Server) InviteToCall(ctx context.Context, req *callspb.InviteToCallRequest) (*callspb.InviteToCallResponse, error) {
	resp, err := s.eps.InviteToCall(ctx, dto.InviteRequest{
		CallID:     req.GetCallId(),
		InviterID:  req.GetInviterId(),
		InviteeIDs: req.GetInviteeIds(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &callspb.InviteToCallResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(*dto.InviteResponse)
	return &callspb.InviteToCallResponse{
		Call:          pbCall(r.Call),
		NewInviteeIds: r.NewInviteeIDs,
		NotifyUserIds: r.NotifyUserIDs,
	}, nil
}

func (s *Server) AcceptCall(ctx context.Context, req *callspb.AcceptCallRequest) (*callspb.AcceptCallResponse, error) {
	resp, err := s.eps.AcceptCall(ctx, dto.AcceptRequest{
		CallID: req.GetCallId(),
		UserID: req.GetUserId(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &callspb.AcceptCallResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(*dto.AcceptResponse)
	return &callspb.AcceptCallResponse{
		Call:    pbCall(r.Call),
		Livekit: pbLivekit(r.Livekit),
	}, nil
}

func (s *Server) DeclineCall(ctx context.Context, req *callspb.DeclineCallRequest) (*callspb.DeclineCallResponse, error) {
	r, err := s.hangup(ctx, s.eps.DeclineCall, req.GetCallId(), req.GetUserId())
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &callspb.DeclineCallResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &callspb.DeclineCallResponse{
		Call:          pbCall(r.Call),
		Ended:         r.Ended,
		NotifyUserIds: r.NotifyUserIDs,
	}, nil
}

func (s *Server) LeaveCall(ctx context.Context, req *callspb.LeaveCallRequest) (*callspb.LeaveCallResponse, error) {
	r, err := s.hangup(ctx, s.eps.LeaveCall, req.GetCallId(), req.GetUserId())
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &callspb.LeaveCallResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &callspb.LeaveCallResponse{
		Call:          pbCall(r.Call),
		Ended:         r.Ended,
		NotifyUserIds: r.NotifyUserIDs,
	}, nil
}

func (s *Server) EndCall(ctx context.Context, req *callspb.EndCallRequest) (*callspb.EndCallResponse, error) {
	r, err := s.hangup(ctx, s.eps.EndCall, req.GetCallId(), req.GetUserId())
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &callspb.EndCallResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &callspb.EndCallResponse{
		Call:          pbCall(r.Call),
		Ended:         r.Ended,
		NotifyUserIds: r.NotifyUserIDs,
	}, nil
}

func (s *Server) hangup(ctx context.Context, ep func(context.Context, any) (any, error),
	callID, userID int64) (*dto.HangupResponse, error) {
	resp, err := ep(ctx, dto.HangupRequest{CallID: callID, UserID: userID})
	if err != nil {
		return nil, err
	}
	return resp.(*dto.HangupResponse), nil
}

// ── Мапинг dto → pb ──────────────────────────────────────────────

func pbError(e *domain.Error) *callspb.Error {
	return &callspb.Error{Code: e.Code, Message: e.Message, HttpStatus: int32(e.HTTPStatus)}
}

func pbLivekit(l dto.LivekitDTO) *callspb.LivekitInfo {
	return &callspb.LivekitInfo{Token: l.Token, Url: l.URL}
}

func pbCall(c *dto.CallDTO) *callspb.Call {
	if c == nil {
		return nil
	}
	out := &callspb.Call{
		Id:          c.ID,
		Kind:        c.Kind,
		Status:      c.Status,
		Media:       c.Media,
		StartedAt:   c.StartedAt,
		EndedAt:     strOrEmpty(c.EndedAt),
		InitiatorId: c.InitiatorID,
		ShareCode:   strOrEmpty(c.ShareCode),
		DurationSec: c.DurationSec,
	}
	if c.InitiatorFIO != nil {
		out.InitiatorFio = *c.InitiatorFIO
	}
	if c.ConversationID != nil {
		out.ConversationId = *c.ConversationID
	}
	for _, p := range c.Participants {
		out.Participants = append(out.Participants, &callspb.Participant{
			UserId:     p.UserID,
			Fio:        p.FIO,
			AvatarPath: strOrEmpty(p.AvatarPath),
			Role:       p.Role,
			JoinedAt:   strOrEmpty(p.JoinedAt),
			LeftAt:     strOrEmpty(p.LeftAt),
			Declined:   p.Declined,
		})
	}
	return out
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
