package service

import (
	"strings"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

// Showcase — всё, что нужно вкладке «Подписки»: текущий тариф с расходом
// места и токенов, линейка тарифов и аддоны.
type Showcase struct {
	Entitlements *domain.Entitlements   `json:"entitlements"`
	Subscription *domain.Subscription   `json:"subscription"`
	Plans        []*domain.Plan         `json:"plans"`
	Addons       []*domain.Addon        `json:"addons"`
	MyAddons     []*domain.UserAddon    `json:"my_addons"`
	Storage      []*domain.StorageEntry `json:"storage"`
	Settings     *domain.Settings       `json:"settings"`
	// PlanLimits — лимиты всей линейки: карточка тарифа показывает, что даёт
	// каждый (значения конечны и приходят с сервера, а не дублируются на фронте).
	PlanLimits map[string]domain.Limits `json:"plan_limits"`
}

func (s *Service) Showcase(ctx domain.Ctx, userID int64) (*Showcase, error) {
	ent, err := s.Entitlements(ctx, userID, 0)
	if err != nil {
		return nil, err
	}
	sub, err := s.Subs.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	plans, err := s.Catalog.ListPlans(ctx, true)
	if err != nil {
		return nil, err
	}
	addons, err := s.Catalog.ListAddons(ctx, true)
	if err != nil {
		return nil, err
	}
	mine, err := s.Subs.ListUserAddons(ctx, userID)
	if err != nil {
		return nil, err
	}
	usage, err := s.Storage.Usage(ctx, userID)
	if err != nil {
		return nil, err
	}
	settings, err := s.Settings.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	limits := map[string]domain.Limits{}
	for _, code := range domain.PlanCodes {
		limits[code] = domain.LimitsFor(code)
	}
	return &Showcase{
		Entitlements: ent,
		Subscription: sub,
		Plans:        plans,
		Addons:       addons,
		MyAddons:     mine,
		Storage:      usage,
		Settings:     settings,
		PlanLimits:   limits,
	}, nil
}

// SetAutoRenew — включить или выключить автопродление подписки.
func (s *Service) SetAutoRenew(ctx domain.Ctx, userID int64, on bool) (*domain.Subscription, error) {
	sub, err := s.Subs.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, domain.ErrNotFound
	}
	sub.AutoRenew = on
	now := s.now()
	if on {
		sub.CancelledAt = nil
	} else {
		sub.CancelledAt = &now
	}
	if err := s.Subs.SaveSubscription(ctx, sub); err != nil {
		return nil, err
	}
	s.publish(ctx, "billing:subscription", userID, map[string]any{"auto_renew": on})
	return sub, nil
}

// CancelAddon — отказаться от докупки (действует до конца оплаченного срока).
func (s *Service) CancelAddon(ctx domain.Ctx, userID, addonID int64) error {
	return s.Subs.CancelAddon(ctx, addonID, userID)
}

// ActivatePromo — промокод-подарок: бесплатные дни тарифа или пачка токенов.
// Скидочные коды (percent/amount) активируются не здесь, а в момент покупки.
func (s *Service) ActivatePromo(ctx domain.Ctx, userID int64, code string) (map[string]any, error) {
	promo, err := s.Promos.GetPromoByCode(ctx, strings.TrimSpace(code))
	if err != nil {
		return nil, err
	}
	if promo == nil {
		return nil, domain.ErrPromoUnknown
	}
	if !promo.Usable(s.now()) {
		return nil, domain.ErrPromoExpired
	}
	if promo.Kind != domain.PromoDays && promo.Kind != domain.PromoTokens {
		return nil, domain.ErrPromoMismatch
	}
	used, err := s.Promos.CountRedemptions(ctx, promo.ID, userID)
	if err != nil {
		return nil, err
	}
	if used >= promo.PerUserLimit {
		return nil, domain.ErrPromoUsed
	}
	ok, err := s.Promos.Redeem(ctx, promo.ID, userID, nil)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrPromoUsed
	}

	if promo.Kind == domain.PromoTokens {
		if err := s.AI.AddExtraTokens(ctx, userID, promo.Value); err != nil {
			return nil, err
		}
		s.publish(ctx, "billing:tokens", userID, map[string]any{"added": promo.Value})
		return map[string]any{"kind": promo.Kind, "tokens": promo.Value}, nil
	}

	plan := domain.PlanMiddle
	if promo.PlanCode != nil && *promo.PlanCode != "" {
		plan = *promo.PlanCode
	}
	until, err := s.grantPlan(ctx, userID, plan, int(promo.Value), domain.SourceGrant,
		"Промокод "+promo.Code)
	if err != nil {
		return nil, err
	}
	return map[string]any{"kind": promo.Kind, "plan": plan, "expires_at": until}, nil
}

// grantPlan — выдать тариф на days дней (промокод и выдача супер-админом).
// Действующая подписка того же тарифа продлевается, младшая — заменяется.
func (s *Service) grantPlan(ctx domain.Ctx, userID int64, plan string, days int, source, note string) (any, error) {
	if _, ok := domain.PlanLimits[plan]; !ok {
		return nil, domain.ErrPlanUnknown
	}
	now := s.now()
	sub, err := s.Subs.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	from := now
	if sub != nil && sub.Active(now) && sub.ExpiresAt != nil &&
		domain.PlanRank(sub.PlanCode) >= domain.PlanRank(plan) {
		from = *sub.ExpiresAt
	}
	until := from.AddDate(0, 0, days)
	next := &domain.Subscription{
		UserID:    userID,
		PlanCode:  plan,
		Period:    domain.PeriodMonth,
		Source:    source,
		StartedAt: now,
		ExpiresAt: &until,
		AutoRenew: false,
		Note:      note,
	}
	if err := s.Subs.SaveSubscription(ctx, next); err != nil {
		return nil, err
	}
	quota, periodEnd := domain.AIQuota(domain.LimitsFor(plan), now)
	if _, err := s.AI.EnsureBalance(ctx, userID, quota, now, periodEnd); err != nil {
		s.Log.Warn("billing.ai_balance_refresh_failed", "user_id", userID, "error", err)
	}
	s.publish(ctx, "billing:subscription", userID, map[string]any{"plan": plan, "expires_at": until})
	return until, nil
}
