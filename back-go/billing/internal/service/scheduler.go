package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

// Планировщик биллинга: продлевает подписки и аддоны, у которых кончился
// оплаченный срок. Автопродление заложено в модель целиком — при подключённом
// платёжном шлюзе счёт списывается сам; пока шлюз заглушка, продление
// выставляется счётом в «Заказах», а подписка возвращается на бесплатный тариф.
const (
	schedulerTick  = 5 * time.Minute
	schedulerBatch = 200
)

// RunScheduler — фоновый цикл; выходит по отмене контекста.
func (s *Service) RunScheduler(ctx context.Context) {
	ticker := time.NewTicker(schedulerTick)
	defer ticker.Stop()
	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Service) tick(ctx context.Context) {
	now := s.now()
	subs, err := s.Subs.DueRenewals(ctx, now, schedulerBatch)
	if err != nil {
		s.Log.Error("billing.due_renewals_failed", "error", err)
	}
	for _, sub := range subs {
		if err := s.renewSubscription(ctx, sub); err != nil {
			s.Log.Error("billing.renew_failed", "user_id", sub.UserID, "error", err)
		}
	}

	addons, err := s.Subs.DueAddonRenewals(ctx, now, schedulerBatch)
	if err != nil {
		s.Log.Error("billing.due_addons_failed", "error", err)
	}
	for _, a := range addons {
		if err := s.renewAddon(ctx, a); err != nil {
			s.Log.Error("billing.addon_renew_failed", "addon_id", a.ID, "error", err)
		}
	}
}

// renewSubscription — срок вышел: продлеваем счётом либо возвращаем на «Джуна».
// Строка подписки при этом удаляется — отсутствие записи и есть бесплатный
// тариф, и истёкшая подписка перестаёт попадать в выборку планировщика.
func (s *Service) renewSubscription(ctx context.Context, sub *domain.Subscription) error {
	if sub.AutoRenew && sub.Source == domain.SourcePurchase {
		order, err := s.renewalOrder(ctx, sub)
		if err != nil {
			return err
		}
		s.publish(ctx, "billing:renewal", sub.UserID, map[string]any{
			"order_id": order.ID, "plan": sub.PlanCode, "amount": order.Amount,
		})
	} else {
		s.publish(ctx, "billing:expired", sub.UserID, map[string]any{"plan": sub.PlanCode})
	}
	return s.Subs.DeleteSubscription(ctx, sub.UserID)
}

// renewalOrder — счёт на продление текущего тарифа тем же периодом.
func (s *Service) renewalOrder(ctx context.Context, sub *domain.Subscription) (*domain.Order, error) {
	plan, err := s.Catalog.GetPlan(ctx, sub.PlanCode)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, domain.ErrPlanUnknown
	}
	amount := plan.PriceMonth
	if sub.Period == domain.PeriodYear {
		amount = plan.PriceYear
	}
	order := &domain.Order{
		UserID: sub.UserID, Kind: domain.OrderKindSubscription, ItemCode: sub.PlanCode,
		Period: sub.Period, Qty: 1, Amount: amount, BaseAmount: amount,
		Status: domain.OrderPending, Title: "Продление тарифа «" + plan.Name + "»",
		Meta: map[string]any{"renewal": true},
	}
	if err := s.Orders.CreateOrder(ctx, order); err != nil {
		return nil, err
	}
	secret := randomSecret()
	pp, err := s.Provider.Create(ctx, order, secret)
	if err != nil {
		return order, nil // счёт есть, оплатить его можно из «Заказов»
	}
	payment := &domain.Payment{
		OrderID: order.ID, Provider: s.Provider.Name(), ProviderPaymentID: pp.ProviderPaymentID,
		Amount: order.Amount, Status: domain.PaymentPending, Method: "sbp",
		ConfirmationURL: pp.ConfirmationURL, WebhookSecret: secret,
	}
	if err := s.Orders.CreatePayment(ctx, payment); err != nil {
		return nil, err
	}
	return order, nil
}

// renewAddon — докупка с автопродлением получает новый срок и счёт; без
// автопродления просто отключается.
func (s *Service) renewAddon(ctx context.Context, a *domain.UserAddon) error {
	if !a.AutoRenew {
		return s.Subs.CancelAddon(ctx, a.ID, a.UserID)
	}
	addon, err := s.Catalog.GetAddon(ctx, a.AddonCode)
	if err != nil {
		return err
	}
	if addon == nil || !addon.IsActive {
		return s.Subs.CancelAddon(ctx, a.ID, a.UserID)
	}
	price := addon.PriceMonth
	if a.Period == domain.PeriodYear && addon.PriceYear > 0 {
		price = addon.PriceYear
	}
	order := &domain.Order{
		UserID: a.UserID, Kind: domain.OrderKindAddon, ItemCode: addon.Code,
		Period: a.Period, Qty: a.Qty, CompanyID: a.CompanyID,
		Amount: price * int64(max(a.Qty, 1)), BaseAmount: price * int64(max(a.Qty, 1)),
		Status: domain.OrderPending, Title: "Продление: " + addon.Name,
		Meta: map[string]any{"renewal": true, "addon_id": a.ID},
	}
	if err := s.Orders.CreateOrder(ctx, order); err != nil {
		return err
	}
	// Пока счёт не оплачен, докупка не действует: отключаем её и вернём при
	// оплате новым аддоном (applyAddon).
	return s.Subs.CancelAddon(ctx, a.ID, a.UserID)
}
