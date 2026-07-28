// Package service — бизнес-логика billingsvc: подписки, магазин, промокоды,
// заказы и платежи, токены доступа к ИИ и учёт хранилища.
//
// Ключевое правило скоупа: подписка принадлежит ПОЛЬЗОВАТЕЛЮ, а компания
// наследует лимиты своего СОЗДАТЕЛЯ. Поэтому любой запрос лимитов сводится к
// «чей тариф применяем» — см. Entitlements.
package service

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

type Deps struct {
	Catalog  domain.CatalogRepository
	Subs     domain.SubscriptionRepository
	Orders   domain.OrderRepository
	Promos   domain.PromoRepository
	Products domain.ProductRepository
	AI       domain.AIRepository
	Storage  domain.StorageRepository
	Settings domain.SettingsRepository
	Audit    domain.AuditRepository
	Identity domain.IdentityReader
	Provider domain.PaymentProvider
	Bus      domain.EventBus
	Log      *slog.Logger
	// Now — источник времени (тесты подменяют).
	Now func() time.Time
}

type Service struct {
	Deps
}

func New(d Deps) *Service {
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Service{Deps: d}
}

func (s *Service) now() time.Time { return s.Now().UTC() }

// publish — сокет-событие владельцу (баланс, подписка, статус заказа).
func (s *Service) publish(ctx domain.Ctx, event string, userID int64, payload any) {
	if s.Bus == nil {
		return
	}
	s.Bus.Publish(ctx, event, []string{roomUser(userID)}, payload)
}

func roomUser(id int64) string {
	return "user_" + strconv.FormatInt(id, 10)
}

// parseInt64 — число из строки; мусор даёт 0 (вызывающий трактует как «нет»).
func parseInt64(s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// GetSettings — платформенные настройки биллинга (комиссия, провайдер оплаты).
func (s *Service) GetSettings(ctx domain.Ctx) (*domain.Settings, error) {
	return s.Settings.GetSettings(ctx)
}
