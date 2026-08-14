package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

// Лимит тарифа на число реестров. Скоуп — ЧЕЛОВЕК: реестр принадлежит ему, а
// не компании, поэтому и тариф спрашиваем его собственный.
// Биллинг не подключён или недоступен — ограничений нет (fail-open).

// WithBilling — подключить проверку лимитов тарифа.
func (s *Service) WithBilling(billing *billingclient.Client) *Service {
	s.billing = billing
	return s
}

// ensureLimit — влезает ли ещё один реестр в тариф владельца.
func (s *Service) ensureLimit(ctx context.Context, ownerID int64) error {
	if s.billing == nil {
		return nil
	}
	ent := s.billing.Entitlements(ctx, ownerID, 0)
	limit := int(ent.Limits.GetRegistries())
	if limit == billingclient.Unlimited {
		return nil
	}
	current, err := s.repo.CountOwned(ctx, ownerID)
	if err != nil {
		return err
	}
	return billingclient.EnsureCount("registries", limit, current, ent.PlanName)
}
