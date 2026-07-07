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

	"github.com/DmitriyODS/gw2/back-go/pets/internal/service"
)

// Endpoints — все use-case'ы petsvc.
type Endpoints struct {
	GetMyPet       endpoint.Endpoint
	FeedPet        endpoint.Endpoint
	RenamePet      endpoint.Endpoint
	EquipItem      endpoint.Endpoint
	GetShop        endpoint.Endpoint
	GetMystery     endpoint.Endpoint
	BuyItem        endpoint.Endpoint
	BuySpecies     endpoint.Endpoint
	SwitchSpecies  endpoint.Endpoint
	ClaimQuest     endpoint.Endpoint
	StartAdventure endpoint.Endpoint

	WalkPet   endpoint.Endpoint
	HealPet   endpoint.Endpoint
	StrokePet endpoint.Endpoint

	GetZoo         endpoint.Endpoint
	GetRating      endpoint.Endpoint
	GetLive        endpoint.Endpoint
	GetActivityLog endpoint.Endpoint
}

// ── Транспорт-независимые запросы ─────────────────────────────────

// Scope — пользователь + компания запроса (после auth и company-scope).
type Scope struct {
	UserID    int64
	CompanyID int64
	UserLevel int
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

// StrokeRequest — поглаживание чужого питомца: PetOwnerID — владелец из
// пути, Scope.UserID — гладящий.
type StrokeRequest struct {
	Scope
	PetOwnerID int64
}

func New(svc *service.Service) Endpoints {
	return Endpoints{
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
			r := request.(Scope)
			return svc.GetShopState(ctx, r.UserID, r.CompanyID)
		},
		GetMystery: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetMysteryItem(ctx, r.UserID, r.CompanyID)
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
		StartAdventure: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.StartAdventure(ctx, r.UserID, r.CompanyID)
		},

		WalkPet: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.WalkPet(ctx, r.UserID, r.CompanyID)
		},
		HealPet: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.HealPet(ctx, r.UserID, r.CompanyID)
		},
		StrokePet: func(ctx context.Context, request any) (any, error) {
			r := request.(StrokeRequest)
			return svc.StrokePet(ctx, r.UserID, r.PetOwnerID, r.CompanyID)
		},

		GetZoo: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetZoo(ctx, r.CompanyID)
		},
		GetRating: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetRating(ctx, r.CompanyID, r.UserID)
		},
		GetLive: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetLive(ctx, r.CompanyID)
		},
		GetActivityLog: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetActivityLog(ctx, r.UserID)
		},
	}
}
