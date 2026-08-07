package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

// Корпоративный портал — возможность платного тарифа. Скоуп компанийный:
// тариф берётся у создателя компании. Биллинг не подключён или недоступен —
// ограничений нет (fail-open).

// WithBilling — подключить проверку тарифа.
func (s *Service) WithBilling(billing *billingclient.Client) *Service {
	s.billing = billing
	return s
}

// ensurePortal — доступен ли портал компании на её тарифе.
func (s *Service) ensurePortal(ctx context.Context, companyID int64) error {
	if s.billing == nil {
		return nil
	}
	ent := s.billing.Entitlements(ctx, 0, companyID)
	if ent.Limits.GetPortal() {
		return nil
	}
	return billingclient.FeatureError("portal", ent.PlanName)
}
