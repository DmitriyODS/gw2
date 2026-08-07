package service

import (
	"time"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

// Entitlements — что доступно здесь и сейчас. companyID > 0 означает
// «лимиты компании»: применяется тариф её СОЗДАТЕЛЯ (платит владелец — его
// компания и получает участников, портал, статистику, календари и реестры).
// Компания без создателя (историческая) остаётся на бесплатных лимитах.
func (s *Service) Entitlements(ctx domain.Ctx, userID, companyID int64) (*domain.Entitlements, error) {
	ownerID := userID
	if companyID > 0 {
		owner, err := s.Identity.CompanyOwner(ctx, companyID)
		if err != nil {
			return nil, err
		}
		ownerID = owner
	}

	ent := &domain.Entitlements{
		UserID:   userID,
		OwnerID:  ownerID,
		Plan:     domain.PlanJunior,
		PlanName: "Джун",
		Limits:   domain.LimitsFor(domain.PlanJunior),
	}
	if ownerID == 0 {
		return ent, nil
	}

	now := s.now()
	sub, err := s.Subs.GetSubscription(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if sub != nil && sub.Active(now) {
		ent.Plan = sub.PlanCode
		ent.Source = sub.Source
		ent.ExpiresAt = sub.ExpiresAt
		ent.AutoRenew = sub.AutoRenew
	}
	ent.Limits = domain.LimitsFor(ent.Plan)
	if plan, err := s.Catalog.GetPlan(ctx, ent.Plan); err == nil && plan != nil {
		ent.PlanName = plan.Name
	}

	addons, err := s.Subs.ListUserAddons(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	// Сначала состояние выпуска (пока подписки скрыты, тариф почти никого не
	// ограничивает — см. EffectiveLimits), и только потом докупки: место,
	// купленное до сокрытия витрины, обязано складываться с бесплатным, а не
	// подменяться им.
	ent.Limits = domain.EffectiveLimits(ent.Limits)
	applyAddons(&ent.Limits, addons, companyID)

	used, err := s.Storage.Total(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	ent.StorageUsed = used

	balance, err := s.AI.GetBalance(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	ent.TokensLimit, _ = domain.AIQuota(ent.Limits, now)
	ent.TokensUsed, ent.TokensLeft = tokenState(balance, ent.TokensLimit, now)
	return ent, nil
}

// applyAddons — докупки поверх лимитов тарифа. Место сотрудника покупается в
// КОНКРЕТНУЮ компанию, поэтому оно считается только для неё.
func applyAddons(limits *domain.Limits, addons []*domain.UserAddon, companyID int64) {
	for _, a := range addons {
		amount := a.Amount * int64(max(a.Qty, 1))
		switch a.Kind {
		case domain.AddonStorage:
			limits.StorageBytes = domain.AddAmount64(limits.StorageBytes, amount)
		case domain.AddonCompany:
			limits.Companies = domain.AddAmount(limits.Companies, int(amount))
		case domain.AddonMember:
			if a.CompanyID == nil || (companyID > 0 && *a.CompanyID == companyID) {
				limits.Members = domain.AddAmount(limits.Members, int(amount))
			}
		}
		// Токены не лимит, а баланс: докупленная пачка лежит в extra_tokens.
	}
}

// tokenState — расход и остаток токенов с УЧЁТОМ ролловера периода, но без
// записи в БД: read-путь не должен трогать хранилище.
func tokenState(b *domain.AIBalance, quota int64, now time.Time) (used, left int64) {
	if quota < 0 {
		quota = 0
	}
	if b == nil {
		return 0, quota
	}
	used = b.UsedTokens
	if !b.PeriodEnd.After(now) {
		used = 0 // период закончился — квота уже обновилась, просто ещё не записана
	}
	left = quota - used + b.ExtraTokens
	if left < 0 {
		left = 0
	}
	return used, left
}

// TrackStorage — учесть появившиеся и удалённые файлы. Зовут сервисы-владельцы
// после заливки и удаления. companyID > 0 — файл компании: место тратится из
// квоты её СОЗДАТЕЛЯ (кто платит, у того и считается).
func (s *Service) TrackStorage(ctx domain.Ctx, userID, companyID int64, service string,
	added []*domain.StoredFile, removedKeys []string) (int64, error) {

	if service == "" {
		return 0, domain.ErrValidation
	}
	ownerID := userID
	if companyID > 0 {
		owner, err := s.Identity.CompanyOwner(ctx, companyID)
		if err != nil {
			return 0, err
		}
		if owner > 0 {
			ownerID = owner
		}
	}
	if ownerID <= 0 {
		return 0, domain.ErrValidation
	}

	// Сначала снимаем удалённое — журнал знает их размеры, поэтому сервису
	// не нужно мерить объекты в хранилище. Затем добавляем новые.
	var delta int64
	if len(removedKeys) > 0 {
		freed, err := s.Storage.RemoveFiles(ctx, ownerID, removedKeys)
		if err != nil {
			return 0, err
		}
		delta -= freed
	}
	if len(added) > 0 {
		for _, f := range added {
			f.Service, f.CompanyID = service, companyID
			delta += f.Size
		}
		if err := s.Storage.AddFiles(ctx, ownerID, added); err != nil {
			return 0, err
		}
	}
	if delta == 0 {
		return s.Storage.Total(ctx, ownerID)
	}
	return s.Storage.Track(ctx, ownerID, service, delta)
}

// CheckStorage — влезает ли файл в квоту владельца (companyID>0 — квота
// создателя компании). Возвращает остаток места.
func (s *Service) CheckStorage(ctx domain.Ctx, userID, companyID, bytes int64) (bool, int64, *domain.Entitlements, error) {
	ent, err := s.Entitlements(ctx, userID, companyID)
	if err != nil {
		return false, 0, nil, err
	}
	if ent.Limits.StorageBytes == domain.Unlimited {
		return true, domain.Unlimited, ent, nil
	}
	free := ent.Limits.StorageBytes - ent.StorageUsed
	if free < 0 {
		free = 0
	}
	return bytes <= free, free, ent, nil
}

// AIState — состояние токенов пользователя: квота тарифа, расход и остаток.
func (s *Service) AIState(ctx domain.Ctx, userID int64) (*domain.Entitlements, map[string]int64, error) {
	ent, err := s.Entitlements(ctx, userID, 0)
	if err != nil {
		return nil, nil, err
	}
	// Расход показываем за ТОТ ЖЕ период, что и квота: иначе «использовано»
	// не сходится с «осталось» — за месяц набегает больше, чем даёт сутки.
	now := s.now()
	_, periodEnd := domain.AIQuota(ent.Limits, now)
	usage, err := s.AI.UsageByFeature(ctx, userID, periodStart(now, periodEnd))
	if err != nil {
		return nil, nil, err
	}
	return ent, usage, nil
}

// periodStart — начало текущего периода квоты: столько же назад, сколько до его
// конца вперёд (сутки для суточной квоты, месяц для месячной).
func periodStart(now, periodEnd time.Time) time.Time {
	if domain.SubscriptionsHidden {
		return periodEnd.AddDate(0, 0, -1)
	}
	return now.AddDate(0, -1, 0)
}

// CheckAI — можно ли обратиться к модели: возвращает плательщика (для
// компанийных функций — создателя компании) и остаток его токенов.
func (s *Service) CheckAI(ctx domain.Ctx, userID, companyID int64) (payerID int64, left int64, err error) {
	ent, err := s.Entitlements(ctx, userID, companyID)
	if err != nil {
		return 0, 0, err
	}
	return ent.OwnerID, ent.TokensLeft, nil
}

// ConsumeAI — списать токены доступа после обращения к модели и записать
// расход. own=true (свой ключ пользователя) — только журнал, без списания.
func (s *Service) ConsumeAI(ctx domain.Ctx, rec domain.AIUsageRecord) (ok bool, left int64, err error) {
	payerID := rec.UserID
	if payerID <= 0 {
		return false, 0, domain.ErrValidation
	}
	if rec.OwnKey || rec.BilledTokens <= 0 {
		return true, 0, s.AI.LogUsage(ctx, rec)
	}

	ent, err := s.Entitlements(ctx, payerID, 0)
	if err != nil {
		return false, 0, err
	}
	now := s.now()
	quota, periodEnd := domain.AIQuota(ent.Limits, now)
	if _, err := s.AI.EnsureBalance(ctx, payerID, quota, now, periodEnd); err != nil {
		return false, 0, err
	}
	ok, balance, err := s.AI.Consume(ctx, payerID, rec.BilledTokens)
	if err != nil {
		return false, 0, err
	}
	if !ok {
		return false, 0, nil
	}
	if err := s.AI.LogUsage(ctx, rec); err != nil {
		s.Log.Warn("billing.ai_usage_log_failed", "user_id", payerID, "error", err)
	}
	left = balance.Left()
	s.publish(ctx, "billing:tokens", payerID, map[string]any{
		"tokens_left": left, "tokens_used": balance.UsedTokens,
	})
	return true, left, nil
}
