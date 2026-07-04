// Package endpoint — go-kit endpoints поверх сервисного слоя.
//
// Каждый use-case обёрнут в endpoint.Endpoint: транспорты (gRPC, Fiber)
// декодируют свои запросы, зовут endpoint и кодируют ответ обратно.
// Бизнес-ошибки (*domain.Error) пролетают через error-канал endpoint'а и
// мапятся транспортом: gRPC — в поле Error ответа, HTTP — в статус + JSON.
package endpoint

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/service"
)

// Endpoints — все use-case'ы groovesvc.
type Endpoints struct {
	GetFeed        endpoint.Endpoint
	ToggleReaction endpoint.Endpoint
	ListComments   endpoint.Endpoint
	AddComment     endpoint.Endpoint
	DeleteComment  endpoint.Endpoint
	SendKudos      endpoint.Endpoint
	GetLive        endpoint.Endpoint

	GetMyPet      endpoint.Endpoint
	FeedPet       endpoint.Endpoint
	RenamePet     endpoint.Endpoint
	EquipItem     endpoint.Endpoint
	GetShop       endpoint.Endpoint
	BuyItem       endpoint.Endpoint
	BuySpecies    endpoint.Endpoint
	SwitchSpecies endpoint.Endpoint
	ClaimQuest    endpoint.Endpoint

	GetZoo    endpoint.Endpoint
	GetRaid   endpoint.Endpoint
	GetRating endpoint.Endpoint

	GetWrapped   endpoint.Endpoint
	ShareWrapped endpoint.Endpoint
	Morning      endpoint.Endpoint
	GrooveTV     endpoint.Endpoint

	GetLocation    endpoint.Endpoint
	SetLocation    endpoint.Endpoint
	DeleteLocation endpoint.Endpoint
	GeoSearch      endpoint.Endpoint
}

// ── Транспорт-независимые запросы ─────────────────────────────────

// Scope — пользователь + компания запроса (после auth и company-scope).
type Scope struct {
	UserID    int64
	CompanyID int64
	UserLevel int
}

type GetFeedRequest struct {
	Scope
	BeforeID int64
	Limit    int
}

type EventRequest struct {
	Scope
	EventID int64
}

type ToggleReactionRequest struct {
	EventRequest
	Emoji string
}

type AddCommentRequest struct {
	EventRequest
	Text      string
	ReplyToID *int64
}

type DeleteCommentRequest struct {
	Scope
	CommentID int64
}

type KudosRequest struct {
	Scope
	ToUserID int64
	Category string
	Text     string
}

type NameRequest struct {
	Scope
	Name string
}

type ItemRequest struct {
	Scope
	Item string
}

type EquipRequest struct {
	Scope
	Item *string
}

type MorningRequest struct {
	Scope
	Part string
}

type LocationRequest struct {
	Scope
	Lat  float64
	Lon  float64
	City *string
}

type GeoSearchRequest struct {
	Scope
	Query string
}

func New(svc *service.Service) Endpoints {
	return Endpoints{
		GetFeed: func(ctx context.Context, request any) (any, error) {
			r := request.(GetFeedRequest)
			return svc.GetFeedPage(ctx, r.CompanyID, r.UserID, r.BeforeID, r.Limit)
		},
		ToggleReaction: func(ctx context.Context, request any) (any, error) {
			r := request.(ToggleReactionRequest)
			return svc.ToggleReaction(ctx, r.EventID, r.UserID, r.CompanyID, r.Emoji)
		},
		ListComments: func(ctx context.Context, request any) (any, error) {
			r := request.(EventRequest)
			return svc.ListComments(ctx, r.EventID, r.CompanyID)
		},
		AddComment: func(ctx context.Context, request any) (any, error) {
			r := request.(AddCommentRequest)
			return svc.AddComment(ctx, r.EventID, r.UserID, r.CompanyID, r.Text, r.ReplyToID)
		},
		DeleteComment: func(ctx context.Context, request any) (any, error) {
			r := request.(DeleteCommentRequest)
			return nil, svc.DeleteComment(ctx, r.CommentID, r.UserID, r.UserLevel)
		},
		SendKudos: func(ctx context.Context, request any) (any, error) {
			r := request.(KudosRequest)
			return nil, svc.SendKudos(ctx, r.CompanyID, r.UserID, r.ToUserID, r.Category, r.Text)
		},
		GetLive: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetLive(ctx, r.CompanyID)
		},

		GetMyPet: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetMyPet(ctx, r.UserID, r.CompanyID)
		},
		FeedPet: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.FeedPet(ctx, r.UserID, r.CompanyID)
		},
		RenamePet: func(ctx context.Context, request any) (any, error) {
			r := request.(NameRequest)
			return svc.RenamePet(ctx, r.UserID, r.CompanyID, r.Name)
		},
		EquipItem: func(ctx context.Context, request any) (any, error) {
			r := request.(EquipRequest)
			return svc.EquipItem(ctx, r.UserID, r.CompanyID, r.Item)
		},
		GetShop: func(ctx context.Context, request any) (any, error) {
			return svc.GetShopState(), nil
		},
		BuyItem: func(ctx context.Context, request any) (any, error) {
			r := request.(ItemRequest)
			return svc.BuyItem(ctx, r.UserID, r.CompanyID, r.Item)
		},
		BuySpecies: func(ctx context.Context, request any) (any, error) {
			r := request.(ItemRequest)
			return svc.BuySpecies(ctx, r.UserID, r.CompanyID, r.Item)
		},
		SwitchSpecies: func(ctx context.Context, request any) (any, error) {
			r := request.(ItemRequest)
			return svc.SwitchSpecies(ctx, r.UserID, r.CompanyID, r.Item)
		},
		ClaimQuest: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.ClaimQuest(ctx, r.UserID, r.CompanyID)
		},

		GetZoo: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetZoo(ctx, r.CompanyID)
		},
		GetRaid: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetRaidState(ctx, r.CompanyID, r.UserID)
		},
		GetRating: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetRating(ctx, r.CompanyID, r.UserID)
		},

		GetWrapped: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetWrapped(ctx, r.CompanyID, r.UserID)
		},
		ShareWrapped: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return nil, svc.ShareWrapped(ctx, r.CompanyID, r.UserID)
		},
		Morning: func(ctx context.Context, request any) (any, error) {
			r := request.(MorningRequest)
			return svc.MorningBriefing(ctx, r.CompanyID, r.UserID, r.Part)
		},
		GrooveTV: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetGrooveTV(ctx, r.CompanyID)
		},

		GetLocation: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetUserLocation(ctx, r.UserID)
		},
		SetLocation: func(ctx context.Context, request any) (any, error) {
			r := request.(LocationRequest)
			return svc.SetUserLocation(ctx, r.UserID, r.Lat, r.Lon, r.City)
		},
		DeleteLocation: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return nil, svc.DeleteUserLocation(ctx, r.UserID)
		},
		GeoSearch: func(ctx context.Context, request any) (any, error) {
			r := request.(GeoSearchRequest)
			return svc.SearchCities(ctx, r.Query)
		},
	}
}
