// Package grpc — gRPC-транспорт portalsvc (его дёргает petsvc: системные
// celebrating-посты вида 'pet_evolved'). Бизнес-ошибки уезжают полем Error
// в ответе — транспорт всегда отвечает OK.
package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/portalpb"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/endpoint"
)

type Server struct {
	portalpb.UnimplementedPortalServiceServer
	eps endpoint.Endpoints
}

func NewServer(eps endpoint.Endpoints) *Server {
	return &Server{eps: eps}
}

func (s *Server) CreateSystemPost(ctx context.Context, req *portalpb.CreateSystemPostRequest) (*portalpb.CreateSystemPostResponse, error) {
	resp, err := s.eps.CreateSystemPost(ctx, endpoint.CreateSystemPostReq{
		CompanyID:  req.GetCompanyId(),
		AuthorID:   req.GetAuthorUserId(),
		SystemKind: req.GetSystemKind(),
		Title:      req.GetTitle(),
		Body:       req.GetBody(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &portalpb.CreateSystemPostResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	p := resp.(*domain.Post)
	return &portalpb.CreateSystemPostResponse{PostId: p.ID}, nil
}

func pbError(e *domain.Error) *portalpb.Error {
	return &portalpb.Error{Code: e.Code, Message: e.Message, HttpStatus: int32(e.HTTPStatus)}
}
