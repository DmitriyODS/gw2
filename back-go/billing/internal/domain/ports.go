package domain

import (
	"context"
	"time"
)

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// CatalogRepository — витрина: тарифы и аддоны (цены правит супер-админ).
type CatalogRepository interface {
	ListPlans(ctx Ctx, onlyActive bool) ([]*Plan, error)
	GetPlan(ctx Ctx, code string) (*Plan, error)
	UpdatePlan(ctx Ctx, p *Plan) error
	ListAddons(ctx Ctx, onlyActive bool) ([]*Addon, error)
	GetAddon(ctx Ctx, code string) (*Addon, error)
	UpdateAddon(ctx Ctx, a *Addon) error
}

// SubscriptionRepository — подписки пользователей и их аддоны.
type SubscriptionRepository interface {
	GetSubscription(ctx Ctx, userID int64) (*Subscription, error)
	// GetSubscriptions — подписки пачкой (лимиты компании читаются вместе с
	// личными, N+1 здесь недопустим).
	GetSubscriptions(ctx Ctx, userIDs []int64) (map[int64]*Subscription, error)
	SaveSubscription(ctx Ctx, s *Subscription) error
	DeleteSubscription(ctx Ctx, userID int64) error
	// ListSubscriptions — реестр подписок для «Аудита платформы».
	ListSubscriptions(ctx Ctx, search string, plan string, limit, offset int) ([]*Subscription, int, error)
	// DueRenewals — подписки, которым пора продлеваться (шедулер).
	DueRenewals(ctx Ctx, now time.Time, limit int) ([]*Subscription, error)

	ListUserAddons(ctx Ctx, userID int64) ([]*UserAddon, error)
	ListUserAddonsFor(ctx Ctx, userIDs []int64) (map[int64][]*UserAddon, error)
	AddAddon(ctx Ctx, a *UserAddon) error
	CancelAddon(ctx Ctx, id, userID int64) error
	DueAddonRenewals(ctx Ctx, now time.Time, limit int) ([]*UserAddon, error)
	RenewAddon(ctx Ctx, id int64, until time.Time) error
}

// OrderRepository — заказы и платежи.
type OrderRepository interface {
	CreateOrder(ctx Ctx, o *Order) error
	GetOrder(ctx Ctx, id int64) (*Order, error)
	ListOrders(ctx Ctx, userID int64, limit, offset int) ([]*Order, int, error)
	ListAllOrders(ctx Ctx, status string, limit, offset int) ([]*Order, int, error)
	SetOrderStatus(ctx Ctx, id int64, status string, paidAt *time.Time) error
	// MarkApplied — пометить заказ применённым; false, если его уже применили
	// (повторный вебхук платежа не должен продлевать подписку дважды).
	MarkApplied(ctx Ctx, id int64) (bool, error)

	CreatePayment(ctx Ctx, p *Payment) error
	GetPayment(ctx Ctx, id int64) (*Payment, error)
	GetPaymentByOrder(ctx Ctx, orderID int64) (*Payment, error)
	SetPaymentStatus(ctx Ctx, id int64, status string, raw map[string]any) error
}

// PromoRepository — промокоды и их активации.
type PromoRepository interface {
	ListPromos(ctx Ctx) ([]*Promo, error)
	GetPromo(ctx Ctx, id int64) (*Promo, error)
	GetPromoByCode(ctx Ctx, code string) (*Promo, error)
	CreatePromo(ctx Ctx, p *Promo) error
	UpdatePromo(ctx Ctx, p *Promo) error
	DeletePromo(ctx Ctx, id int64) error
	CountRedemptions(ctx Ctx, promoID, userID int64) (int, error)
	// Redeem — атомарно списать активацию (проверка лимитов в WHERE).
	Redeem(ctx Ctx, promoID, userID int64, orderID *int64) (bool, error)
}

// ProductRepository — товары магазина и покупки.
type ProductRepository interface {
	ListShowcase(ctx Ctx, kind, search string, viewerID int64, limit, offset int) ([]*Product, int, error)
	GetProduct(ctx Ctx, id int64) (*Product, error)
	CreateProduct(ctx Ctx, p *Product) error
	UpdateProduct(ctx Ctx, p *Product) error
	SetProductStatus(ctx Ctx, id int64, status, reason string) error
	DeleteProduct(ctx Ctx, id int64) error
	ListByAuthor(ctx Ctx, authorID int64) ([]*Product, error)
	ListModeration(ctx Ctx) ([]*Product, error)
	// PurchaseProduct — покупка одной транзакцией: запись покупки, счётчик
	// продаж и выручка автора за вычетом комиссии.
	PurchaseProduct(ctx Ctx, productID, userID int64, orderID *int64, amount, authorShare int64, authorID *int64) error
	ListPurchases(ctx Ctx, userID int64) ([]*ProductPurchase, error)
	IsOwned(ctx Ctx, productID, userID int64) (bool, error)

	GetSellerBalance(ctx Ctx, userID int64) (*SellerBalance, error)
	CreatePayout(ctx Ctx, p *Payout) error
	ListPayouts(ctx Ctx, userID int64, all bool) ([]*Payout, error)
	ProcessPayout(ctx Ctx, id int64, status, note string) error
}

// AIRepository — токены доступа к ИИ.
type AIRepository interface {
	// GetBalance — баланс как есть, без записи (read-путь: карточка настроек,
	// entitlements). Ролловер периода здесь только вычисляется, не пишется —
	// частый поллинг клиента не должен гонять read-modify-write.
	GetBalance(ctx Ctx, userID int64) (*AIBalance, error)
	// EnsureBalance — баланс пользователя с ленивым ролловером периода: конец
	// периода обнуляет расход и ставит новую квоту до periodEnd. Длину периода
	// (сутки или месяц) решает домен — см. AIQuota.
	EnsureBalance(ctx Ctx, userID int64, planTokens int64, now, periodEnd time.Time) (*AIBalance, error)
	// Consume — атомарно списать токены; false, если не хватило.
	Consume(ctx Ctx, userID int64, tokens int64) (bool, *AIBalance, error)
	AddExtraTokens(ctx Ctx, userID int64, tokens int64) error
	SetBalance(ctx Ctx, userID int64, planTokens, usedTokens, extraTokens int64) error
	LogUsage(ctx Ctx, rec AIUsageRecord) error
	// UsageByFeature — расход за период (карточка настроек и «Аудит платформы»).
	UsageByFeature(ctx Ctx, userID int64, from time.Time) (map[string]int64, error)
	TotalUsage(ctx Ctx, from time.Time) (map[string]int64, error)
}

// StorageRepository — учёт занятого места (дельтами от сервисов-владельцев).
type StorageRepository interface {
	Usage(ctx Ctx, userID int64) ([]*StorageEntry, error)
	Total(ctx Ctx, userID int64) (int64, error)
	TotalsFor(ctx Ctx, userIDs []int64) (map[int64]int64, error)
	Track(ctx Ctx, userID int64, service string, delta int64) (int64, error)
	Set(ctx Ctx, userID int64, service string, bytes int64) error
}

// SettingsRepository — платформенные настройки биллинга.
type SettingsRepository interface {
	GetSettings(ctx Ctx) (*Settings, error)
	UpdateSettings(ctx Ctx, s *Settings) error
}

// AuditRepository — журнал административных действий.
type AuditRepository interface {
	LogAction(ctx Ctx, e *AuditEntry) error
	ListAudit(ctx Ctx, action string, limit, offset int) ([]*AuditEntry, int, error)
}

// IdentityReader — read-only идентичность: пользователи, создатели компаний и
// членство. Таблицы ведёт authsvc, мы их только читаем (как remindersvc читает
// users, а notesvc — user_companies).
type IdentityReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
	SearchUsers(ctx Ctx, query string, limit int) ([]*User, error)
	// CompanyOwner — создатель компании: его тариф действует на всю компанию.
	CompanyOwner(ctx Ctx, companyID int64) (int64, error)
	// OwnedCompanies — компании, созданные пользователем.
	OwnedCompanies(ctx Ctx, userID int64) ([]int64, error)
	// IsCompanyMember — состоит ли пользователь в компании.
	IsCompanyMember(ctx Ctx, userID, companyID int64) (bool, error)
}

// EventBus — сокет-события клиентам через Redis gw2:billing:events.
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}
