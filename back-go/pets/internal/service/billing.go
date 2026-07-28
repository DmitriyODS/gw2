package service

import (
	"context"
	"net/http"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

// Премиум-контент витрины (скины, декор домика, товары) по тарифной линейке
// доступен только на старшем тарифе: он виден всем, но покупается по подписке.
// Скоуп ЛИЧНЫЙ — питомец принадлежит человеку, а не компании.
//
// Биллинг не подключён или недоступен — ограничений нет (fail-open).

// errPremiumOnly — покупка премиум-позиции на младшем тарифе.
var errPremiumOnly = domain.NewError("PREMIUM_ONLY",
	"Этот товар доступен на старшем тарифе — оформите подписку в магазине.",
	http.StatusPaymentRequired)

// WithBilling — подключить проверку тарифа.
func (s *Service) WithBilling(billing *billingclient.Client) *Service {
	s.billing = billing
	return s
}

// ensurePremium — можно ли купить премиум-позицию. kind различает раздел
// витрины: скины питомца, декор домика и прочие товары включаются тарифом
// по отдельности.
func (s *Service) ensurePremium(ctx context.Context, userID int64, premium bool, kind string) error {
	if !premium || s.billing == nil {
		return nil
	}
	limits := s.billing.Entitlements(ctx, userID, 0).Limits
	allowed := limits.GetPremiumPetGoods()
	switch kind {
	case "skin", "species":
		allowed = limits.GetPremiumPetSkins()
	case "house":
		allowed = limits.GetPremiumPetHouse()
	}
	if allowed {
		return nil
	}
	return errPremiumOnly
}
