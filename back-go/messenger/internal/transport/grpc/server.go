// Package grpc — gRPC-транспорт мессенджера (его дёргает callsvc: плашки
// звонков). Бизнес-ошибки уезжают полем Error в ответе — транспорт всегда
// отвечает OK.
package grpc

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/messengerpb"
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

func (s *Server) CreatePostMessage(ctx context.Context, req *messengerpb.CreatePostMessageRequest) (*messengerpb.CreatePostMessageResponse, error) {
	resp, err := s.eps.CreatePostMessage(ctx, endpoint.CreatePostMessageRequest{
		ConversationID: req.GetConversationId(),
		SenderID:       req.GetSenderId(),
		PostID:         req.GetPostId(),
		Title:          req.GetTitle(),
		Excerpt:        req.GetExcerpt(),
		CoverURL:       req.GetCoverUrl(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &messengerpb.CreatePostMessageResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(endpoint.CallMessageResponse)
	raw, err := messageJSON(r.Message)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &messengerpb.CreatePostMessageResponse{
		MessageJson:   raw,
		NotifyUserIds: r.NotifyUserIDs,
	}, nil
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
