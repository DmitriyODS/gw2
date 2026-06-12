// Package grpc — gRPC-транспорт aisvc (его дёргают Flask и groovesvc:
// LLM-шлюз, эмбеддинги, семантический поиск, переиндексация). Бизнес-ошибки
// уезжают полем Error в ответе — транспорт всегда отвечает OK.
package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/aipb"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/service"
)

type Server struct {
	aipb.UnimplementedAiServiceServer
	eps endpoint.Endpoints
}

func NewServer(eps endpoint.Endpoints) *Server {
	return &Server{eps: eps}
}

func (s *Server) Status(ctx context.Context, req *aipb.StatusRequest) (*aipb.StatusResponse, error) {
	resp, err := s.eps.Status(ctx, req.GetCompanyId())
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &aipb.StatusResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(*service.StatusResult)
	return &aipb.StatusResponse{
		Enabled:        r.Enabled,
		ModelChat:      r.ModelChat,
		ModelEmbedding: r.ModelEmbedding,
	}, nil
}

func (s *Server) Chat(ctx context.Context, req *aipb.ChatRequest) (*aipb.ChatResponse, error) {
	resp, err := s.eps.Chat(ctx, service.ChatArgs{
		CompanyID:    req.GetCompanyId(),
		MessagesJSON: req.GetMessagesJson(),
		ToolsJSON:    req.GetToolsJson(),
		MaxTokens:    int(req.GetMaxTokens()),
		Temperature:  req.GetTemperature(),
		TimeoutSec:   req.GetTimeoutSec(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &aipb.ChatResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(*domain.ChatResult)
	return &aipb.ChatResponse{
		Content:       r.Content,
		ToolCallsJson: r.ToolCallsJSON,
	}, nil
}

func (s *Server) Embed(ctx context.Context, req *aipb.EmbedRequest) (*aipb.EmbedResponse, error) {
	resp, err := s.eps.Embed(ctx, endpoint.EmbedRequest{
		CompanyID: req.GetCompanyId(), Text: req.GetText(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &aipb.EmbedResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(endpoint.EmbedResponse)
	return &aipb.EmbedResponse{Vector: r.Vector, Model: r.Model}, nil
}

func (s *Server) SemanticSearch(ctx context.Context, req *aipb.SemanticSearchRequest) (*aipb.SemanticSearchResponse, error) {
	resp, err := s.eps.SemanticSearch(ctx, endpoint.SearchRequest{
		CompanyID: req.GetCompanyId(), Query: req.GetQuery(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &aipb.SemanticSearchResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	hits := resp.([]domain.SearchHit)
	out := &aipb.SemanticSearchResponse{Hits: make([]*aipb.SearchHit, 0, len(hits))}
	for _, h := range hits {
		out.Hits = append(out.Hits, &aipb.SearchHit{TaskId: h.TaskID, Score: h.Score})
	}
	return out, nil
}

func (s *Server) ReindexTask(ctx context.Context, req *aipb.ReindexTaskRequest) (*aipb.ReindexTaskResponse, error) {
	// Работа уходит в фон внутри сервиса (fail-open) — RPC отвечает сразу OK.
	if _, err := s.eps.ReindexTask(ctx, req.GetTaskId()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &aipb.ReindexTaskResponse{}, nil
}

func pbError(e *domain.Error) *aipb.Error {
	return &aipb.Error{Code: e.Code, Message: e.Message, HttpStatus: int32(e.HTTPStatus)}
}
