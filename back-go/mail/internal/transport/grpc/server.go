// Package grpc — gRPC-транспорт mailsvc. Его дёргает authsvc (письма
// подтверждения email). Бизнес-ошибка уезжает полем Error; транспорт OK.
package grpc

import (
	"context"
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/mail/internal/service"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/mailpb"
)

type Server struct {
	mailpb.UnimplementedMailServiceServer
	svc *service.Service
	log *slog.Logger
}

func NewServer(svc *service.Service, log *slog.Logger) *Server {
	return &Server{svc: svc, log: log}
}

func (s *Server) Send(ctx context.Context, req *mailpb.SendRequest) (*mailpb.SendResponse, error) {
	if err := s.svc.Send(ctx, req.GetTo(), req.GetToName(), req.GetTemplate(), req.GetParams()); err != nil {
		s.log.Error("mail.send_failed", "to", req.GetTo(), "template", req.GetTemplate(), "error", err)
		return &mailpb.SendResponse{Error: &mailpb.Error{
			Code: "MAIL_SEND_FAILED", Message: err.Error(), HttpStatus: 502,
		}}, nil
	}
	s.log.Info("mail.sent", "to", req.GetTo(), "template", req.GetTemplate())
	return &mailpb.SendResponse{}, nil
}
