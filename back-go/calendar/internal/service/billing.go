package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

// Лимит тарифа на число календарей. Скоуп — компании: тариф берётся у создателя компании.
// Биллинг не подключён или недоступен — ограничений нет (fail-open).

// WithBilling — подключить проверку лимитов тарифа.
func (s *Service) WithBilling(billing *billingclient.Client) *Service {
	s.billing = billing
	return s
}

// ensureLimit — влезает ли ещё один calendar в тариф.
func (s *Service) ensureLimit(ctx context.Context, scopeID int64) error {
	if s.billing == nil {
		return nil
	}
	ent := s.billing.Entitlements(ctx, 0, scopeID)
	limit := int(ent.Limits.GetCalendars())
	if limit == billingclient.Unlimited {
		return nil
	}
	current, err := s.repo.CountCalendars(ctx, scopeID)
	if err != nil {
		return err
	}
	return billingclient.EnsureCount("calendars", limit, current, ent.PlanName)
}
