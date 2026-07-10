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

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
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
	ResetSpecies   endpoint.Endpoint
	ClaimQuest     endpoint.Endpoint
	StartAdventure endpoint.Endpoint
	PrestigePet    endpoint.Endpoint

	GetSeason         endpoint.Endpoint
	ClaimSeasonReward endpoint.Endpoint

	GetHouse      endpoint.Endpoint
	BuyHouseDecor endpoint.Endpoint
	ArrangeHouse  endpoint.Endpoint

	WalkPet   endpoint.Endpoint
	HealPet   endpoint.Endpoint
	StrokePet endpoint.Endpoint

	GetZoo         endpoint.Endpoint
	DeleteZooPet   endpoint.Endpoint
	GetRating      endpoint.Endpoint
	GetLive        endpoint.Endpoint
	GetActivityLog endpoint.Endpoint

	GetBank        endpoint.Endpoint
	GetBankLedger  endpoint.Endpoint
	TransferKudos  endpoint.Endpoint
	BankDeposit    endpoint.Endpoint
	BankWithdraw   endpoint.Endpoint
	BankTakeLoan   endpoint.Endpoint
	BankRepayLoan  endpoint.Endpoint
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

// ZooDeleteRequest — удаление питомца сотрудника администратором компании:
// TargetUserID — владелец удаляемого питомца из пути.
type ZooDeleteRequest struct {
	Scope
	TargetUserID int64
}

// SeasonClaimRequest — забрать награду порога сезонного трека.
type SeasonClaimRequest struct {
	Scope
	Threshold int
}

// ArrangeRequest — свободная расстановка декора домика (координаты — % сцены).
type ArrangeRequest struct {
	Scope
	Placed []domain.HouseItem
}

// TransferRequest — перевод кудосов коллеге по компании.
type TransferRequest struct {
	Scope
	ToUserID int64
	Amount   int
	Comment  string
}

// BankAmountRequest — вклад/снятие/кредит/погашение (одна сумма).
type BankAmountRequest struct {
	Scope
	Amount int
}

// LedgerRequest — страница выписки (keyset вниз от BeforeID; 0 — с начала).
type LedgerRequest struct {
	Scope
	BeforeID int64
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
		ResetSpecies: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.ResetSpecies(ctx, r.UserID, r.CompanyID)
		},
		ClaimQuest: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.ClaimQuest(ctx, r.UserID, r.CompanyID)
		},
		StartAdventure: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.StartAdventure(ctx, r.UserID, r.CompanyID)
		},
		PrestigePet: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.PrestigePet(ctx, r.UserID, r.CompanyID)
		},

		GetSeason: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetSeason(ctx, r.UserID, r.CompanyID)
		},
		ClaimSeasonReward: func(ctx context.Context, request any) (any, error) {
			r := request.(SeasonClaimRequest)
			return svc.ClaimSeasonReward(ctx, r.UserID, r.CompanyID, r.Threshold)
		},

		GetHouse: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetHouse(ctx, r.UserID, r.CompanyID)
		},
		BuyHouseDecor: func(ctx context.Context, request any) (any, error) {
			r := request.(ItemRequest)
			return svc.BuyHouseDecor(ctx, r.UserID, r.CompanyID, r.Item)
		},
		ArrangeHouse: func(ctx context.Context, request any) (any, error) {
			r := request.(ArrangeRequest)
			return svc.ArrangeHouse(ctx, r.UserID, r.CompanyID, r.Placed)
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
			return svc.GetZoo(ctx, r.CompanyID, r.UserID)
		},
		DeleteZooPet: func(ctx context.Context, request any) (any, error) {
			r := request.(ZooDeleteRequest)
			return nil, svc.DeleteColleaguePet(ctx, r.UserLevel, r.TargetUserID, r.CompanyID)
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

		GetBank: func(ctx context.Context, request any) (any, error) {
			r := request.(Scope)
			return svc.GetBank(ctx, r.UserID, r.CompanyID)
		},
		GetBankLedger: func(ctx context.Context, request any) (any, error) {
			r := request.(LedgerRequest)
			return svc.GetBankLedger(ctx, r.UserID, r.BeforeID)
		},
		TransferKudos: func(ctx context.Context, request any) (any, error) {
			r := request.(TransferRequest)
			return svc.TransferKudos(ctx, r.UserID, r.ToUserID, r.CompanyID, r.Amount, r.Comment)
		},
		BankDeposit: func(ctx context.Context, request any) (any, error) {
			r := request.(BankAmountRequest)
			return svc.BankDeposit(ctx, r.UserID, r.CompanyID, r.Amount)
		},
		BankWithdraw: func(ctx context.Context, request any) (any, error) {
			r := request.(BankAmountRequest)
			return svc.BankWithdraw(ctx, r.UserID, r.CompanyID, r.Amount)
		},
		BankTakeLoan: func(ctx context.Context, request any) (any, error) {
			r := request.(BankAmountRequest)
			return svc.BankTakeLoan(ctx, r.UserID, r.CompanyID, r.Amount)
		},
		BankRepayLoan: func(ctx context.Context, request any) (any, error) {
			r := request.(BankAmountRequest)
			return svc.BankRepayLoan(ctx, r.UserID, r.CompanyID, r.Amount)
		},
	}
}
