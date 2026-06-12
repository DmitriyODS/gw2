// Package grpc — gRPC-транспорт мессенджера (его дёргают Flask и groovesvc:
// плашки звонков, бот Грувика, контекст pet-чата). Бизнес-ошибки уезжают
// полем Error в ответе — транспорт всегда отвечает OK.
package grpc

import (
	"context"
	"encoding/json"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/messengerpb"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/endpoint"
)

type Server struct {
	messengerpb.UnimplementedMessengerServiceServer
	eps endpoint.Endpoints
}

func NewServer(eps endpoint.Endpoints) *Server {
	return &Server{eps: eps}
}

func (s *Server) EnsureDialog(ctx context.Context, req *messengerpb.EnsureDialogRequest) (*messengerpb.EnsureDialogResponse, error) {
	resp, err := s.eps.EnsureDialog(ctx, endpoint.PairRequest{
		UserAID: req.GetUserAId(), UserBID: req.GetUserBId(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &messengerpb.EnsureDialogResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &messengerpb.EnsureDialogResponse{ConversationId: resp.(int64)}, nil
}

func (s *Server) CreateCallMessage(ctx context.Context, req *messengerpb.CreateCallMessageRequest) (*messengerpb.CreateCallMessageResponse, error) {
	resp, err := s.eps.CreateCallMessage(ctx, endpoint.CreateCallMessageRequest{
		ConversationID: req.GetConversationId(),
		SenderID:       req.GetSenderId(),
		CallID:         req.GetCallId(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &messengerpb.CreateCallMessageResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(endpoint.CallMessageResponse)
	raw, err := messageJSON(r.Message)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &messengerpb.CreateCallMessageResponse{
		MessageJson:   raw,
		NotifyUserIds: r.NotifyUserIDs,
	}, nil
}

func (s *Server) GetCallMessage(ctx context.Context, req *messengerpb.GetCallMessageRequest) (*messengerpb.GetCallMessageResponse, error) {
	resp, err := s.eps.GetCallMessage(ctx, req.GetCallId())
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &messengerpb.GetCallMessageResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(endpoint.CallMessageResponse)
	raw, err := messageJSON(r.Message)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &messengerpb.GetCallMessageResponse{
		ConversationId: r.ConversationID,
		MessageJson:    raw,
		NotifyUserIds:  r.NotifyUserIDs,
	}, nil
}

func (s *Server) PostBotMessage(ctx context.Context, req *messengerpb.PostBotMessageRequest) (*messengerpb.PostBotMessageResponse, error) {
	resp, err := s.eps.PostBotMessage(ctx, endpoint.PostBotMessageRequest{
		ConversationID: req.GetConversationId(), Text: req.GetText(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &messengerpb.PostBotMessageResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &messengerpb.PostBotMessageResponse{MessageId: resp.(int64)}, nil
}

func (s *Server) ListRecentMessages(ctx context.Context, req *messengerpb.ListRecentMessagesRequest) (*messengerpb.ListRecentMessagesResponse, error) {
	resp, err := s.eps.ListRecentMessages(ctx, endpoint.ListRecentRequest{
		ConversationID: req.GetConversationId(), Limit: int(req.GetLimit()),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &messengerpb.ListRecentMessagesResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	msgs := resp.([]*domain.Message)
	out := &messengerpb.ListRecentMessagesResponse{
		Messages: make([]*messengerpb.ChatMessage, 0, len(msgs)),
	}
	for _, m := range msgs {
		cm := &messengerpb.ChatMessage{
			Id:        m.ID,
			IsBot:     m.IsBot,
			CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339Nano),
		}
		if m.SenderID != nil {
			cm.SenderId = *m.SenderID
		}
		if m.Text != nil {
			cm.Text = *m.Text
		}
		out.Messages = append(out.Messages, cm)
	}
	return out, nil
}

// messageJSON — снапшот сообщения в ТОЧНОЙ форме REST-ответа: вызывающий
// эмитит его в Socket.IO как payload message:new/message:updated.
func messageJSON(m *dto.Message) (string, error) {
	raw, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func pbError(e *domain.Error) *messengerpb.Error {
	return &messengerpb.Error{Code: e.Code, Message: e.Message, HttpStatus: int32(e.HTTPStatus)}
}
