package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

// Лимиты тарифа в authsvc: сколько компаний пользователь может создать и
// сколько человек помещается в компанию. Компанийные лимиты считаются по
// тарифу СОЗДАТЕЛЯ компании — это делает сам биллинг, здесь достаточно
// передать company_id.
//
// Биллинг недоступен или не подключён (billing == nil) — ограничений нет:
// платформа продолжает работать, см. billingclient (fail-open).

// WithBilling — подключить проверку лимитов тарифа.
func (s *Service) WithBilling(billing *billingclient.Client) *Service {
	s.billing = billing
	return s
}

// ensureCompanyLimit — влезает ли ещё одна СВОЯ компания в тариф пользователя.
func (s *Service) ensureCompanyLimit(ctx context.Context, userID int64) error {
	if s.billing == nil {
		return nil
	}
	ent := s.billing.Entitlements(ctx, userID, 0)
	limit := int(ent.Limits.GetCompanies())
	if limit == billingclient.Unlimited {
		return nil
	}
	current, err := s.companies.CountCompaniesCreatedBy(ctx, userID)
	if err != nil {
		return err
	}
	return billingclient.EnsureCount("companies", limit, current, ent.PlanName)
}

// ensureMemberLimit — влезает ли ещё один участник в компанию. Лимит берётся
// по тарифу создателя компании, текущее число — из счётчиков компании.
func (s *Service) ensureMemberLimit(ctx context.Context, companyID int64) error {
	if s.billing == nil || companyID <= 0 {
		return nil
	}
	ent := s.billing.Entitlements(ctx, 0, companyID)
	limit := int(ent.Limits.GetMembers())
	if limit == billingclient.Unlimited {
		return nil
	}
	stats, err := s.companies.CompanyStats(ctx, []int64{companyID})
	if err != nil {
		return err
	}
	return billingclient.EnsureCount("members", limit, stats[companyID].Employees, ent.PlanName)
}

// planAllowsStatuses — доступны ли пользовательские статусы на тарифе.
func (s *Service) planAllowsStatuses(ctx context.Context, userID int64) error {
	if s.billing == nil {
		return nil
	}
	ent := s.billing.Entitlements(ctx, userID, 0)
	if ent.Limits.GetUserStatuses() {
		return nil
	}
	return billingclient.FeatureError("user_statuses", ent.PlanName)
}
