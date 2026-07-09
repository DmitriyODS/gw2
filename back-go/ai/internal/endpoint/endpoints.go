// Package endpoint — go-kit endpoints поверх сервисного слоя.
//
// Каждый use-case обёрнут в endpoint.Endpoint: транспорты (gRPC, Fiber)
// декодируют свои запросы, зовут endpoint и кодируют ответ обратно.
// Бизнес-ошибки (*domain.Error) пролетают через error-канал endpoint'а и
// мапятся транспортом: gRPC — в поле Error ответа, HTTP — в статус + JSON.
package endpoint

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/ai/internal/service"
)

// Endpoints — все use-case'ы aisvc.
type Endpoints struct {
	GetSettings    endpoint.Endpoint
	UpdateSettings endpoint.Endpoint
	TestSettings   endpoint.Endpoint
	IndexingStatus endpoint.Endpoint
	StartReindex   endpoint.Endpoint
	GetTVFact      endpoint.Endpoint

	Status         endpoint.Endpoint
	Chat           endpoint.Endpoint
	Embed          endpoint.Endpoint
	SemanticSearch endpoint.Endpoint
	ReindexTask    endpoint.Endpoint

	// ИИ-ассистент (REST /api/ai/assistant/*).
	SendAssistantMessage  endpoint.Endpoint
	GetAssistantHistory   endpoint.Endpoint
	SendAssistantFeedback endpoint.Endpoint

	// ИИ-инструменты текста заметок (REST /api/ai/text-tools).
	TransformText endpoint.Endpoint
}

// ── Транспорт-независимые запросы/ответы ─────────────────────────

type SettingsRequest struct {
	Actor     *domain.User
	CompanyID int64
}

type UpdateSettingsRequest struct {
	Actor     *domain.User
	CompanyID int64
	Update    dto.AiSettingsUpdate
}

type EmbedRequest struct {
	CompanyID int64
	Text      string
}

type SearchRequest struct {
	CompanyID int64
	Query     string
}

type EmbedResponse struct {
	Vector []float32
	Model  string
}

// ── ИИ-ассистент ───────────────────────────────────────────────────

type SendAssistantMessageRequest struct {
	UserID    int64
	CompanyID int64
	Text      string
}

type GetAssistantHistoryRequest struct {
	UserID    int64
	CompanyID int64
	Limit     int
	Before    *time.Time
}

type SendAssistantFeedbackRequest struct {
	UserID    int64
	CompanyID int64
	MessageID int64
	Verdict   string
	Reason    *string
}

type TransformTextRequest struct {
	CompanyID int64
	Action    string
	Style     string
	Text      string
}

func New(svc service.AiService) Endpoints {
	return Endpoints{
		GetSettings: func(ctx context.Context, request any) (any, error) {
			req := request.(SettingsRequest)
			return svc.GetSettings(ctx, req.Actor, req.CompanyID)
		},
		UpdateSettings: func(ctx context.Context, request any) (any, error) {
			req := request.(UpdateSettingsRequest)
			return svc.UpdateSettings(ctx, req.Actor, req.CompanyID, req.Update)
		},
		TestSettings: func(ctx context.Context, request any) (any, error) {
			req := request.(SettingsRequest)
			return svc.TestSettings(ctx, req.Actor, req.CompanyID)
		},
		IndexingStatus: func(ctx context.Context, request any) (any, error) {
			req := request.(SettingsRequest)
			return svc.IndexingStatus(ctx, req.Actor, req.CompanyID)
		},
		StartReindex: func(ctx context.Context, request any) (any, error) {
			req := request.(SettingsRequest)
			return svc.StartReindex(ctx, req.Actor, req.CompanyID)
		},
		GetTVFact: func(ctx context.Context, request any) (any, error) {
			return svc.GetTVFact(ctx, request.(int64))
		},

		Status: func(ctx context.Context, request any) (any, error) {
			return svc.Status(ctx, request.(int64))
		},
		Chat: func(ctx context.Context, request any) (any, error) {
			return svc.Chat(ctx, request.(service.ChatArgs))
		},
		Embed: func(ctx context.Context, request any) (any, error) {
			req := request.(EmbedRequest)
			vector, model, err := svc.Embed(ctx, req.CompanyID, req.Text)
			if err != nil {
				return nil, err
			}
			return EmbedResponse{Vector: vector, Model: model}, nil
		},
		SemanticSearch: func(ctx context.Context, request any) (any, error) {
			req := request.(SearchRequest)
			return svc.SemanticSearch(ctx, req.CompanyID, req.Query)
		},
		ReindexTask: func(ctx context.Context, request any) (any, error) {
			svc.ScheduleReindexTask(request.(int64))
			return nil, nil
		},

		SendAssistantMessage: func(ctx context.Context, request any) (any, error) {
			req := request.(SendAssistantMessageRequest)
			return svc.SendAssistantMessage(ctx, req.UserID, req.CompanyID, req.Text)
		},
		GetAssistantHistory: func(ctx context.Context, request any) (any, error) {
			req := request.(GetAssistantHistoryRequest)
			return svc.GetAssistantHistory(ctx, req.UserID, req.CompanyID, req.Limit, req.Before)
		},
		SendAssistantFeedback: func(ctx context.Context, request any) (any, error) {
			req := request.(SendAssistantFeedbackRequest)
			return nil, svc.SendAssistantFeedback(ctx, req.UserID, req.CompanyID, req.MessageID, req.Verdict, req.Reason)
		},
		TransformText: func(ctx context.Context, request any) (any, error) {
			req := request.(TransformTextRequest)
			return svc.TransformText(ctx, req.CompanyID, req.Action, req.Style, req.Text)
		},
	}
}
