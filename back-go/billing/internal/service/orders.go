package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

// PurchaseRequest — намерение купить: тариф, аддон или товар витрины.
type PurchaseRequest struct {
	Kind      string `json:"kind"`       // subscription | addon | product
	ItemCode  string `json:"item_code"`  // код тарифа или аддона
	ProductID int64  `json:"product_id"` // товар витрины
	Period    string `json:"period"`     // month | year
	Qty       int    `json:"qty"`
	CompanyID *int64 `json:"company_id"` // для аддона «место сотрудника»
	Promo     string `json:"promo"`
}

// Quote — расчёт стоимости покупки до оформления (витрина показывает его в
// карточке товара вместе с промокодом).
type Quote struct {
	Title      string `json:"title"`
	BaseAmount int64  `json:"base_amount"`
	Discount   int64  `json:"discount"`
	Amount     int64  `json:"amount"`
	Period     string `json:"period"`
	PromoCode  string `json:"promo_code,omitempty"`
	promoID    *int64
}

// Quote — посчитать стоимость и проверить, что покупка вообще возможна.
func (s *Service) Quote(ctx domain.Ctx, userID int64, req PurchaseRequest) (*Quote, error) {
	if req.Qty <= 0 {
		req.Qty = 1
	}
	if req.Period != domain.PeriodYear {
		req.Period = domain.PeriodMonth
	}

	q := &Quote{Period: req.Period}
	switch req.Kind {
	case domain.OrderKindSubscription:
		plan, err := s.Catalog.GetPlan(ctx, req.ItemCode)
		if err != nil {
			return nil, err
		}
		if plan == nil || !plan.IsActive {
			return nil, domain.ErrPlanUnknown
		}
		if plan.Code == domain.PlanJunior {
			return nil, domain.ErrFreePlan
		}
		q.Title = "Тариф «" + plan.Name + "»"
		q.BaseAmount = plan.PriceMonth
		if req.Period == domain.PeriodYear {
			q.BaseAmount = plan.PriceYear
		}
	case domain.OrderKindAddon:
		addon, err := s.Catalog.GetAddon(ctx, req.ItemCode)
		if err != nil {
			return nil, err
		}
		if addon == nil || !addon.IsActive {
			return nil, domain.ErrAddonUnknown
		}
		if !addon.Recurring {
			req.Period = domain.PeriodMonth // разовая покупка периода не имеет
			q.Period = domain.PeriodMonth
		}
		price := addon.PriceMonth
		if req.Period == domain.PeriodYear && addon.PriceYear > 0 {
			price = addon.PriceYear
		}
		q.Title = addon.Name
		q.BaseAmount = price * int64(req.Qty)
	case domain.OrderKindProduct:
		product, err := s.Products.GetProduct(ctx, req.ProductID)
		if err != nil {
			return nil, err
		}
		if product == nil || product.Status != domain.ProductPublished {
			return nil, domain.ErrNotFound
		}
		owned, err := s.Products.IsOwned(ctx, product.ID, userID)
		if err != nil {
			return nil, err
		}
		if owned {
			return nil, domain.ErrAlreadyOwned
		}
		q.Title = product.Title
		q.BaseAmount = product.Price
	default:
		return nil, domain.ErrValidation
	}

	q.Amount = q.BaseAmount
	if code := strings.TrimSpace(req.Promo); code != "" {
		promo, err := s.applicablePromo(ctx, userID, code, req.Kind)
		if err != nil {
			return nil, err
		}
		q.Discount = promoDiscount(promo, q.BaseAmount)
		q.Amount = q.BaseAmount - q.Discount
		q.PromoCode = promo.Code
		q.promoID = &promo.ID
	}
	if q.Amount < 0 {
		q.Amount = 0
	}
	return q, nil
}

// applicablePromo — промокод, годный для этой покупки и этого пользователя.
func (s *Service) applicablePromo(ctx domain.Ctx, userID int64, code, kind string) (*domain.Promo, error) {
	promo, err := s.Promos.GetPromoByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if promo == nil {
		return nil, domain.ErrPromoUnknown
	}
	if !promo.Usable(s.now()) {
		return nil, domain.ErrPromoExpired
	}
	if promo.Kind != domain.PromoPercent && promo.Kind != domain.PromoAmount {
		return nil, domain.ErrPromoMismatch // days/tokens активируются отдельно
	}
	if promo.AppliesTo != "any" && promo.AppliesTo != kind {
		return nil, domain.ErrPromoMismatch
	}
	used, err := s.Promos.CountRedemptions(ctx, promo.ID, userID)
	if err != nil {
		return nil, err
	}
	if used >= promo.PerUserLimit {
		return nil, domain.ErrPromoUsed
	}
	return promo, nil
}

func promoDiscount(p *domain.Promo, base int64) int64 {
	switch p.Kind {
	case domain.PromoPercent:
		d := base * p.Value / 100
		if d > base {
			return base
		}
		return d
	case domain.PromoAmount:
		if p.Value > base {
			return base
		}
		return p.Value
	}
	return 0
}

// Purchase — оформить заказ. Бесплатный (промокод на 100%) применяется сразу,
// платный уходит в оплату: пока платёжный шлюз — заглушка, заказ ждёт
// подтверждения супер-админа.
func (s *Service) Purchase(ctx domain.Ctx, userID int64, req PurchaseRequest) (*domain.Order, error) {
	settings, err := s.Settings.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.StoreEnabled {
		return nil, domain.ErrStoreDisabled
	}
	quote, err := s.Quote(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	if req.Qty <= 0 {
		req.Qty = 1
	}

	order := &domain.Order{
		UserID:     userID,
		Kind:       req.Kind,
		ItemCode:   req.ItemCode,
		Period:     quote.Period,
		Qty:        req.Qty,
		CompanyID:  req.CompanyID,
		Amount:     quote.Amount,
		BaseAmount: quote.BaseAmount,
		Discount:   quote.Discount,
		PromoID:    quote.promoID,
		Status:     domain.OrderPending,
		Title:      quote.Title,
		Meta:       map[string]any{},
	}
	if req.Kind == domain.OrderKindProduct {
		id := req.ProductID
		order.ProductID = &id
	}
	if err := s.Orders.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	if quote.promoID != nil {
		ok, err := s.Promos.Redeem(ctx, *quote.promoID, userID, &order.ID)
		if err != nil {
			return nil, err
		}
		if !ok {
			_ = s.Orders.SetOrderStatus(ctx, order.ID, domain.OrderCanceled, nil)
			return nil, domain.ErrPromoUsed
		}
	}

	// Полностью погашенный промокодом заказ платежа не требует.
	if order.Amount == 0 {
		if err := s.markPaid(ctx, order, nil); err != nil {
			return nil, err
		}
		return s.Orders.GetOrder(ctx, order.ID)
	}

	secret, err := randomSecret()
	if err != nil {
		s.Log.Error("billing.secret_generate_failed", "order_id", order.ID, "error", err)
		return nil, domain.ErrPaymentFailed
	}
	pp, err := s.Provider.Create(ctx, order, secret)
	if err != nil {
		s.Log.Error("billing.payment_create_failed", "order_id", order.ID, "error", err)
		return nil, domain.ErrPaymentFailed
	}
	payment := &domain.Payment{
		OrderID:           order.ID,
		Provider:          s.Provider.Name(),
		ProviderPaymentID: pp.ProviderPaymentID,
		Amount:            order.Amount,
		Status:            domain.PaymentPending,
		Method:            "sbp",
		ConfirmationURL:   pp.ConfirmationURL,
		WebhookSecret:     secret,
	}
	if err := s.Orders.CreatePayment(ctx, payment); err != nil {
		return nil, err
	}
	order.Payment = payment
	return order, nil
}

// randomSecret — секрет вебхука платежа. Ошибку генерации отдаём наверх, а не
// глушим в пустую строку: ConfirmPayment трактует пустой сохранённый секрет
// как «нечего сверять» — тихий сбой рандома снял бы проверку с вебхука.
func randomSecret() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// markPaid — заказ оплачен: фиксируем статус и применяем ровно один раз.
func (s *Service) markPaid(ctx domain.Ctx, order *domain.Order, payment *domain.Payment) error {
	now := s.now()
	if err := s.Orders.SetOrderStatus(ctx, order.ID, domain.OrderPaid, &now); err != nil {
		return err
	}
	if payment != nil {
		if err := s.Orders.SetPaymentStatus(ctx, payment.ID, domain.PaymentSucceeded, nil); err != nil {
			return err
		}
	}
	fresh, err := s.Orders.MarkApplied(ctx, order.ID)
	if err != nil {
		return err
	}
	if !fresh {
		return nil // заказ уже применён: повторный вебхук ничего не удваивает
	}
	return s.applyOrder(ctx, order)
}

// applyOrder — выдать оплаченное: продлить подписку, включить аддон или
// передать товар покупателю.
func (s *Service) applyOrder(ctx domain.Ctx, order *domain.Order) error {
	switch order.Kind {
	case domain.OrderKindSubscription:
		return s.applySubscription(ctx, order)
	case domain.OrderKindAddon:
		return s.applyAddon(ctx, order)
	case domain.OrderKindProduct:
		return s.applyProduct(ctx, order)
	}
	return nil
}

func (s *Service) applySubscription(ctx domain.Ctx, order *domain.Order) error {
	now := s.now()
	sub, err := s.Subs.GetSubscription(ctx, order.UserID)
	if err != nil {
		return err
	}
	// Тот же тариф — продлеваем от текущей даты окончания, иначе начинаем заново.
	from := now
	if sub != nil && sub.PlanCode == order.ItemCode && sub.Active(now) && sub.ExpiresAt != nil {
		from = *sub.ExpiresAt
	}
	until := addPeriod(from, order.Period)
	next := &domain.Subscription{
		UserID:    order.UserID,
		PlanCode:  order.ItemCode,
		Period:    order.Period,
		Source:    domain.SourcePurchase,
		StartedAt: now,
		ExpiresAt: &until,
		AutoRenew: true,
	}
	if err := s.Subs.SaveSubscription(ctx, next); err != nil {
		return err
	}
	// Квота токенов подтягивается под новый тариф сразу, а не со следующего
	// обращения к модели.
	quota, periodEnd := domain.AIQuota(domain.LimitsFor(next.PlanCode), now)
	if _, err := s.AI.EnsureBalance(ctx, order.UserID, quota, now, periodEnd); err != nil {
		s.Log.Warn("billing.ai_balance_refresh_failed", "user_id", order.UserID, "error", err)
	}
	s.publish(ctx, "billing:subscription", order.UserID, map[string]any{
		"plan": next.PlanCode, "expires_at": until,
	})
	return nil
}

func (s *Service) applyAddon(ctx domain.Ctx, order *domain.Order) error {
	addon, err := s.Catalog.GetAddon(ctx, order.ItemCode)
	if err != nil {
		return err
	}
	if addon == nil {
		return domain.ErrAddonUnknown
	}
	qty := max(order.Qty, 1)

	// Токены — разовая пачка: она не «действует до», а просто ложится на баланс.
	if addon.Kind == domain.AddonTokens {
		if err := s.AI.AddExtraTokens(ctx, order.UserID, addon.Amount*int64(qty)); err != nil {
			return err
		}
		s.publish(ctx, "billing:tokens", order.UserID, map[string]any{"added": addon.Amount * int64(qty)})
		return nil
	}

	until := addPeriod(s.now(), order.Period)
	ua := &domain.UserAddon{
		UserID:    order.UserID,
		AddonCode: addon.Code,
		Kind:      addon.Kind,
		Amount:    addon.Amount,
		Qty:       qty,
		CompanyID: order.CompanyID,
		Period:    order.Period,
		ExpiresAt: &until,
		AutoRenew: true,
	}
	if err := s.Subs.AddAddon(ctx, ua); err != nil {
		return err
	}
	s.publish(ctx, "billing:addon", order.UserID, map[string]any{"code": addon.Code, "expires_at": until})
	return nil
}

func (s *Service) applyProduct(ctx domain.Ctx, order *domain.Order) error {
	if order.ProductID == nil {
		return domain.ErrValidation
	}
	product, err := s.Products.GetProduct(ctx, *order.ProductID)
	if err != nil {
		return err
	}
	if product == nil {
		return domain.ErrNotFound
	}
	settings, err := s.Settings.GetSettings(ctx)
	if err != nil {
		return err
	}
	authorShare := int64(0)
	if product.AuthorID != nil {
		// Комиссию платформы удерживаем с цены покупки.
		authorShare = order.Amount - order.Amount*int64(settings.CommissionPct)/100
	}
	if err := s.Products.PurchaseProduct(ctx, product.ID, order.UserID, &order.ID,
		order.Amount, authorShare, product.AuthorID); err != nil {
		return err
	}
	s.publish(ctx, "billing:product", order.UserID, map[string]any{"product_id": product.ID})
	if product.AuthorID != nil {
		s.publish(ctx, "billing:sale", *product.AuthorID, map[string]any{
			"product_id": product.ID, "amount": authorShare,
		})
	}
	return nil
}

// addPeriod — конец оплаченного периода.
func addPeriod(from time.Time, period string) time.Time {
	if period == domain.PeriodYear {
		return from.AddDate(1, 0, 0)
	}
	return from.AddDate(0, 1, 0)
}

// ConfirmPayment — вебхук платёжного шлюза. Секрет платежа обязателен: он
// доказывает, что уведомление относится именно к этому счёту.
func (s *Service) ConfirmPayment(ctx domain.Ctx, body []byte, headers map[string]string) error {
	ev, err := s.Provider.Parse(ctx, body, headers)
	if err != nil {
		return domain.ErrValidation
	}
	payment, err := s.paymentByProviderID(ctx, ev.ProviderPaymentID)
	if err != nil {
		return err
	}
	if payment == nil {
		return domain.ErrNotFound
	}
	// payment.WebhookSecret пустым быть не должно (randomSecret отдаёт ошибку
	// наверх, а не пустую строку) — но проверяем явно: ConstantTimeCompare
	// двух пустых слайсов считает их РАВНЫМИ, а пустой сохранённый секрет
	// обязан отклонять вебхук, а не молча его пропускать.
	if payment.WebhookSecret == "" ||
		subtle.ConstantTimeCompare([]byte(payment.WebhookSecret), []byte(ev.Secret)) != 1 {
		return domain.ErrForbidden
	}
	order, err := s.Orders.GetOrder(ctx, payment.OrderID)
	if err != nil || order == nil {
		return domain.ErrNotFound
	}
	switch ev.Status {
	case domain.PaymentSucceeded:
		return s.markPaid(ctx, order, payment)
	case domain.PaymentCanceled, domain.PaymentFailed:
		if err := s.Orders.SetPaymentStatus(ctx, payment.ID, ev.Status, ev.Raw); err != nil {
			return err
		}
		return s.Orders.SetOrderStatus(ctx, order.ID, domain.OrderCanceled, nil)
	}
	return nil
}

// paymentByProviderID — платёж по идентификатору у провайдера. Заглушка
// формирует его как manual-<order_id>, поэтому ищем через заказ.
func (s *Service) paymentByProviderID(ctx domain.Ctx, providerID string) (*domain.Payment, error) {
	id := strings.TrimPrefix(providerID, "manual-")
	orderID := parseInt64(id)
	if orderID == 0 {
		return nil, domain.ErrNotFound
	}
	return s.Orders.GetPaymentByOrder(ctx, orderID)
}

// ListOrders — история заказов пользователя.
func (s *Service) ListOrders(ctx domain.Ctx, userID int64, limit, offset int) ([]*domain.Order, int, error) {
	return s.Orders.ListOrders(ctx, userID, limit, offset)
}

// CancelOrder — отменить неоплаченный заказ.
func (s *Service) CancelOrder(ctx domain.Ctx, userID, orderID int64) error {
	order, err := s.Orders.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil || order.UserID != userID {
		return domain.ErrNotFound
	}
	if order.Status != domain.OrderPending {
		return domain.ErrOrderFinished
	}
	return s.Orders.SetOrderStatus(ctx, orderID, domain.OrderCanceled, nil)
}
