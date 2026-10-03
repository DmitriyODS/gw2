package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

// Лимит тарифа на число календарей считается по автору — как у реестров.
// Биллинг не подключён или недоступен — ограничений нет (fail-open).

// WithBilling — подключить проверку лимитов тарифа.
func (s *Service) WithBilling(billing *billingclient.Client) *Service {
	s.billing = billing
	return s
}

// ensureLimit — влезает ли ещё один calendar в тариф.
func (s *Service) ensureLimit(ctx context.Context, userID int64) error {
	if s.billing == nil {
		return nil
	}
	ent := s.billing.Entitlements(ctx, userID, 0)
	limit := int(ent.Limits.GetCalendars())
	if limit == billingclient.Unlimited {
		return nil
	}
	current, err := s.repo.CountOwned(ctx, userID)
	if err != nil {
		return err
	}
	return billingclient.EnsureCount("calendars", limit, current, ent.PlanName)
}
