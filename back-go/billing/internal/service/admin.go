package service

import (
	"strconv"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

// Раздел «Аудит платформы»: всё, что супер-админ делает с деньгами и тарифами,
// проходит через эти методы и попадает в журнал действий.

// audit — запись в журнал; сбой журнала не должен ронять саму операцию.
func (s *Service) audit(ctx domain.Ctx, actorID int64, action, targetKind, targetID, summary string, payload map[string]any) {
	var actor *int64
	if actorID > 0 {
		actor = &actorID
	}
	if err := s.Audit.LogAction(ctx, &domain.AuditEntry{
		ActorID: actor, Action: action, TargetKind: targetKind,
		TargetID: targetID, Summary: summary, Payload: payload,
	}); err != nil {
		s.Log.Warn("billing.audit_failed", "action", action, "error", err)
	}
}

// LogExternalAction — журнал для действий других сервисов (authsvc пишет сюда
// блокировки пользователей и компаний). Отдельный метод, чтобы не путать с
// внутренним audit.
func (s *Service) LogExternalAction(ctx domain.Ctx, e *domain.AuditEntry) error {
	return s.Audit.LogAction(ctx, e)
}

// UpdatePlan — правка цен и подводки тарифа. Лимиты неизменны — они в коде.
func (s *Service) UpdatePlan(ctx domain.Ctx, actorID int64, p *domain.Plan) (*domain.Plan, error) {
	current, err := s.Catalog.GetPlan(ctx, p.Code)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, domain.ErrPlanUnknown
	}
	if p.PriceMonth < 0 || p.PriceYear < 0 || strings.TrimSpace(p.Name) == "" {
		return nil, domain.ErrValidation
	}
	if err := s.Catalog.UpdatePlan(ctx, p); err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "plan.update", "plan", p.Code, "Изменён тариф "+p.Name, map[string]any{
		"price_month": p.PriceMonth, "price_year": p.PriceYear, "is_active": p.IsActive,
	})
	return s.Catalog.GetPlan(ctx, p.Code)
}

// UpdateAddon — правка аддона (цена, объём, доступность).
func (s *Service) UpdateAddon(ctx domain.Ctx, actorID int64, a *domain.Addon) (*domain.Addon, error) {
	current, err := s.Catalog.GetAddon(ctx, a.Code)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, domain.ErrAddonUnknown
	}
	if a.Amount <= 0 || a.PriceMonth < 0 || a.PriceYear < 0 {
		return nil, domain.ErrValidation
	}
	a.Kind = current.Kind // вид аддона задан кодом и не меняется
	if err := s.Catalog.UpdateAddon(ctx, a); err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "addon.update", "addon", a.Code, "Изменено дополнение "+a.Name, map[string]any{
		"price_month": a.PriceMonth, "amount": a.Amount, "is_active": a.IsActive,
	})
	return s.Catalog.GetAddon(ctx, a.Code)
}

// UpdateSettings — комиссия платформы, платёжный провайдер, доступность магазина.
func (s *Service) UpdateSettings(ctx domain.Ctx, actorID int64, in *domain.Settings) (*domain.Settings, error) {
	if in.CommissionPct < 0 || in.CommissionPct > 100 {
		return nil, domain.ErrValidation
	}
	if err := s.Settings.UpdateSettings(ctx, in); err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "settings.update", "settings", "1", "Изменены настройки биллинга", map[string]any{
		"commission_pct": in.CommissionPct, "payment_provider": in.PaymentProvider,
		"payment_enabled": in.PaymentEnabled, "store_enabled": in.StoreEnabled,
	})
	return s.Settings.GetSettings(ctx)
}

// AdminSubscription — строка реестра подписок с именем владельца и расходом.
type AdminSubscription struct {
	*domain.Subscription
	UserName    string `json:"user_name"`
	UserLogin   string `json:"user_login"`
	StorageUsed int64  `json:"storage_used"`
	TokensLeft  int64  `json:"tokens_left"`
}

// ListSubscriptions — реестр подписок (одним запросом плюс батч расхода: N+1
// здесь недопустим).
func (s *Service) ListSubscriptions(ctx domain.Ctx, search, plan string, limit, offset int) ([]*AdminSubscription, int, error) {
	subs, total, err := s.Subs.ListSubscriptions(ctx, search, plan, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(subs))
	for _, sub := range subs {
		ids = append(ids, sub.UserID)
	}
	storage, err := s.Storage.TotalsFor(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*AdminSubscription, 0, len(subs))
	for _, sub := range subs {
		row := &AdminSubscription{Subscription: sub, StorageUsed: storage[sub.UserID]}
		if u, err := s.Identity.GetUser(ctx, sub.UserID); err == nil && u != nil {
			row.UserName, row.UserLogin = u.FIO, u.Login
		}
		out = append(out, row)
	}
	return out, total, nil
}

// GrantSubscription — выдать тариф пользователю (дни считаются от текущего
// окончания, если действующий тариф не ниже выдаваемого).
func (s *Service) GrantSubscription(ctx domain.Ctx, actorID, userID int64, plan string, days int, note string) (*domain.Subscription, error) {
	if days <= 0 {
		return nil, domain.ErrValidation
	}
	user, err := s.Identity.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrNotFound
	}
	if _, err := s.grantPlan(ctx, userID, plan, days, domain.SourceGrant, note); err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "subscription.grant", "user", strconv.FormatInt(userID, 10),
		"Выдан тариф "+plan+" пользователю "+user.FIO, map[string]any{"plan": plan, "days": days})
	return s.Subs.GetSubscription(ctx, userID)
}

// RevokeSubscription — снять подписку: пользователь возвращается на «Джуна».
func (s *Service) RevokeSubscription(ctx domain.Ctx, actorID, userID int64) error {
	if err := s.Subs.DeleteSubscription(ctx, userID); err != nil {
		return err
	}
	s.audit(ctx, actorID, "subscription.revoke", "user", strconv.FormatInt(userID, 10),
		"Снята подписка", nil)
	s.publish(ctx, "billing:subscription", userID, map[string]any{"plan": domain.PlanJunior})
	return nil
}

// GrantTokens — начислить (или списать отрицательным числом) токены доступа.
func (s *Service) GrantTokens(ctx domain.Ctx, actorID, userID int64, tokens int64) error {
	if tokens == 0 {
		return domain.ErrValidation
	}
	if err := s.AI.AddExtraTokens(ctx, userID, tokens); err != nil {
		return err
	}
	s.audit(ctx, actorID, "tokens.grant", "user", strconv.FormatInt(userID, 10),
		"Изменён баланс токенов", map[string]any{"tokens": tokens})
	s.publish(ctx, "billing:tokens", userID, map[string]any{"added": tokens})
	return nil
}

// ResetTokens — обнулить расход и докупленные токены пользователя.
func (s *Service) ResetTokens(ctx domain.Ctx, actorID, userID int64) error {
	ent, err := s.Entitlements(ctx, userID, 0)
	if err != nil {
		return err
	}
	if err := s.AI.SetBalance(ctx, userID, max(ent.Limits.AITokens, 0), 0, 0); err != nil {
		return err
	}
	s.audit(ctx, actorID, "tokens.reset", "user", strconv.FormatInt(userID, 10), "Обнулены токены", nil)
	s.publish(ctx, "billing:tokens", userID, map[string]any{"reset": true})
	return nil
}

// ---------- Промокоды ----------

func (s *Service) ListPromos(ctx domain.Ctx) ([]*domain.Promo, error) { return s.Promos.ListPromos(ctx) }

func (s *Service) CreatePromo(ctx domain.Ctx, actorID int64, p *domain.Promo) (*domain.Promo, error) {
	p.Code = strings.ToUpper(strings.TrimSpace(p.Code))
	if p.Code == "" || p.Value < 0 {
		return nil, domain.ErrValidation
	}
	switch p.Kind {
	case domain.PromoPercent, domain.PromoAmount, domain.PromoDays, domain.PromoTokens:
	default:
		return nil, domain.ErrValidation
	}
	if p.PerUserLimit <= 0 {
		p.PerUserLimit = 1
	}
	if p.AppliesTo == "" {
		p.AppliesTo = "any"
	}
	exists, err := s.Promos.GetPromoByCode(ctx, p.Code)
	if err != nil {
		return nil, err
	}
	if exists != nil {
		return nil, domain.ErrPromoExists
	}
	if err := s.Promos.CreatePromo(ctx, p); err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "promo.create", "promo", p.Code, "Создан промокод "+p.Code,
		map[string]any{"kind": p.Kind, "value": p.Value})
	return p, nil
}

func (s *Service) UpdatePromo(ctx domain.Ctx, actorID int64, p *domain.Promo) (*domain.Promo, error) {
	current, err := s.Promos.GetPromo(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, domain.ErrNotFound
	}
	p.Code = current.Code
	if p.PerUserLimit <= 0 {
		p.PerUserLimit = 1
	}
	if err := s.Promos.UpdatePromo(ctx, p); err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "promo.update", "promo", p.Code, "Изменён промокод "+p.Code, nil)
	return s.Promos.GetPromo(ctx, p.ID)
}

func (s *Service) DeletePromo(ctx domain.Ctx, actorID, id int64) error {
	if err := s.Promos.DeletePromo(ctx, id); err != nil {
		return err
	}
	s.audit(ctx, actorID, "promo.delete", "promo", strconv.FormatInt(id, 10), "Удалён промокод", nil)
	return nil
}

// ---------- Товары и модерация ----------

func (s *Service) ListModeration(ctx domain.Ctx) ([]*domain.Product, error) {
	return s.Products.ListModeration(ctx)
}

// ReviewProduct — решение модерации: publish либо reject с причиной.
func (s *Service) ReviewProduct(ctx domain.Ctx, actorID, id int64, approve bool, reason string) (*domain.Product, error) {
	p, err := s.Products.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrNotFound
	}
	status := domain.ProductRejected
	action := "product.reject"
	if approve {
		status, action, reason = domain.ProductPublished, "product.publish", ""
	}
	if err := s.Products.SetProductStatus(ctx, id, status, reason); err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, action, "product", strconv.FormatInt(id, 10), p.Title, map[string]any{"reason": reason})
	if p.AuthorID != nil {
		s.publish(ctx, "billing:product_review", *p.AuthorID, map[string]any{
			"product_id": id, "status": status, "reason": reason,
		})
	}
	return s.Products.GetProduct(ctx, id)
}

// CreatePlatformProduct — позиция магазина от самой платформы (без автора).
func (s *Service) CreatePlatformProduct(ctx domain.Ctx, actorID int64, in ProductInput) (*domain.Product, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	p := &domain.Product{
		Kind: in.Kind, Title: strings.TrimSpace(in.Title), Description: in.Description,
		Price: in.Price, Status: domain.ProductPublished, CoverPath: in.CoverPath, Payload: in.Payload,
	}
	if err := s.Products.CreateProduct(ctx, p); err != nil {
		return nil, err
	}
	if err := s.Products.SetProductStatus(ctx, p.ID, domain.ProductPublished, ""); err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "product.create", "product", strconv.FormatInt(p.ID, 10), p.Title, nil)
	return s.Products.GetProduct(ctx, p.ID)
}

// UpdatePlatformProduct — правка любой позиции магазина супер-админом.
func (s *Service) UpdatePlatformProduct(ctx domain.Ctx, actorID, id int64, in ProductInput) (*domain.Product, error) {
	p, err := s.Products.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrNotFound
	}
	if err := in.validate(); err != nil {
		return nil, err
	}
	p.Kind, p.Title, p.Description, p.Price = in.Kind, strings.TrimSpace(in.Title), in.Description, in.Price
	p.CoverPath, p.Payload = in.CoverPath, in.Payload
	if err := s.Products.UpdateProduct(ctx, p); err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "product.update", "product", strconv.FormatInt(id, 10), p.Title, nil)
	return s.Products.GetProduct(ctx, id)
}

func (s *Service) DeletePlatformProduct(ctx domain.Ctx, actorID, id int64) error {
	p, err := s.Products.GetProduct(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return domain.ErrNotFound
	}
	if p.SalesCount > 0 {
		// Проданное только снимаем с витрины — иначе покупатели потеряют товар.
		if err := s.Products.SetProductStatus(ctx, id, domain.ProductRemoved, ""); err != nil {
			return err
		}
	} else if err := s.Products.DeleteProduct(ctx, id); err != nil {
		return err
	}
	s.audit(ctx, actorID, "product.delete", "product", strconv.FormatInt(id, 10), p.Title, nil)
	return nil
}

// ---------- Заказы, платежи, выплаты ----------

func (s *Service) ListAllOrders(ctx domain.Ctx, status string, limit, offset int) ([]*domain.Order, int, error) {
	return s.Orders.ListAllOrders(ctx, status, limit, offset)
}

// ConfirmOrder — ручное подтверждение оплаты (пока платёжный шлюз заглушка,
// это единственный способ довести заказ до выдачи).
func (s *Service) ConfirmOrder(ctx domain.Ctx, actorID, orderID int64) (*domain.Order, error) {
	order, err := s.Orders.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, domain.ErrNotFound
	}
	if order.Status == domain.OrderPaid {
		return nil, domain.ErrOrderFinished
	}
	payment, err := s.Orders.GetPaymentByOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if err := s.markPaid(ctx, order, payment); err != nil {
		return nil, err
	}
	s.audit(ctx, actorID, "order.confirm", "order", strconv.FormatInt(orderID, 10),
		"Подтверждена оплата заказа", map[string]any{"amount": order.Amount})
	return s.Orders.GetOrder(ctx, orderID)
}

func (s *Service) ListAllPayouts(ctx domain.Ctx) ([]*domain.Payout, error) {
	return s.Products.ListPayouts(ctx, 0, true)
}

// ProcessPayout — выплата автору: paid — деньги отправлены, rejected — отказ
// с возвратом суммы на кошелёк.
func (s *Service) ProcessPayout(ctx domain.Ctx, actorID, id int64, status, note string) error {
	if status != "paid" && status != "rejected" {
		return domain.ErrValidation
	}
	if err := s.Products.ProcessPayout(ctx, id, status, note); err != nil {
		return err
	}
	s.audit(ctx, actorID, "payout."+status, "payout", strconv.FormatInt(id, 10), note, nil)
	return nil
}

func (s *Service) ListAudit(ctx domain.Ctx, action string, limit, offset int) ([]*domain.AuditEntry, int, error) {
	return s.Audit.ListAudit(ctx, action, limit, offset)
}

// SearchUsers — подсказка пользователей для выдачи подписок и токенов.
func (s *Service) SearchUsers(ctx domain.Ctx, query string, limit int) ([]*domain.User, error) {
	return s.Identity.SearchUsers(ctx, strings.TrimSpace(query), limit)
}
