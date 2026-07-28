package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

// Лимит тарифа на число ежедневников. Скоуп — пользователя: считается личный тариф владельца.
// Биллинг не подключён или недоступен — ограничений нет (fail-open).

// WithBilling — подключить проверку лимитов тарифа.
func (s *Service) WithBilling(billing *billingclient.Client) *Service {
	s.billing = billing
	return s
}

// ensureLimit — влезает ли ещё один diary в тариф.
func (s *Service) ensureLimit(ctx context.Context, scopeID int64) error {
	if s.billing == nil {
		return nil
	}
	ent := s.billing.Entitlements(ctx, scopeID, 0)
	limit := int(ent.Limits.GetDiaries())
	if limit == billingclient.Unlimited {
		return nil
	}
	current, err := s.repo.CountDiaries(ctx, scopeID)
	if err != nil {
		return err
	}
	return billingclient.EnsureCount("diaries", limit, current, ent.PlanName)
}
