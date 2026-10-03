package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

// Лимит тарифа на число досок. Скоуп — пользователя: считается личный тариф владельца.
// Биллинг не подключён или недоступен — ограничений нет (fail-open).

// WithBilling — подключить проверку лимитов тарифа.
func (s *Service) WithBilling(billing *billingclient.Client) *Service {
	s.billing = billing
	return s
}

// ensureLimit — влезают ли ещё extra досок в тариф.
func (s *Service) ensureLimit(ctx context.Context, scopeID int64, extra int) error {
	if s.billing == nil {
		return nil
	}
	ent := s.billing.Entitlements(ctx, scopeID, 0)
	limit := int(ent.Limits.GetBoards())
	if limit == billingclient.Unlimited {
		return nil
	}
	current, err := s.repo.CountBoards(ctx, scopeID)
	if err != nil {
		return err
	}
	return billingclient.EnsureCount("boards", limit, current+extra-1, ent.PlanName)
}
