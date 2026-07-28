package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

// Лимиты тарифа в tasksvc: сколько задач помещается в компанию, доступны ли
// расширенная статистика и выгрузка данных. Все три — КОМПАНИЙНЫЕ: тариф берётся
// у создателя компании (это делает биллинг, здесь достаточно company_id).
//
// Биллинг не подключён или недоступен — ограничений нет (fail-open).

// WithBilling — подключить проверку лимитов тарифа.
func (s *Service) WithBilling(billing *billingclient.Client) *Service {
	s.billing = billing
	return s
}

// ensureTaskLimit — влезает ли ещё одна задача в тариф компании.
func (s *Service) ensureTaskLimit(ctx context.Context, companyID int64) error {
	if s.billing == nil {
		return nil
	}
	ent := s.billing.Entitlements(ctx, 0, companyID)
	limit := ent.Limits.GetTasks()
	if limit == billingclient.Unlimited {
		return nil
	}
	current, err := s.tasks.CountCompanyTasks(ctx, companyID)
	if err != nil {
		return err
	}
	if current < limit {
		return nil
	}
	return billingclient.LimitError("tasks", limit, current, ent.PlanName)
}

// ensureAdvancedStats — расширенная статистика компании доступна с платного
// тарифа.
func (s *Service) ensureAdvancedStats(ctx context.Context, companyID *int64) error {
	if s.billing == nil || companyID == nil {
		return nil
	}
	ent := s.billing.Entitlements(ctx, 0, *companyID)
	if ent.Limits.GetAdvancedStats() {
		return nil
	}
	return billingclient.FeatureError("advanced_stats", ent.PlanName)
}

// ensureDataTransfer — выгрузка данных компании (xlsx) доступна с платного
// тарифа.
func (s *Service) ensureDataTransfer(ctx context.Context, companyID *int64) error {
	if s.billing == nil || companyID == nil {
		return nil
	}
	ent := s.billing.Entitlements(ctx, 0, *companyID)
	if ent.Limits.GetDataTransfer() {
		return nil
	}
	return billingclient.FeatureError("data_transfer", ent.PlanName)
}
