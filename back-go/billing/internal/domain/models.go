package domain

import "time"

// Периоды оплаты.
const (
	PeriodMonth = "month"
	PeriodYear  = "year"
)

// Источники подписки.
const (
	SourcePurchase = "purchase" // оплачена пользователем
	SourceGrant    = "grant"    // выдал супер-админ
	SourceGrace    = "grace"    // переходный период при запуске подписок
)

// Виды аддонов.
const (
	AddonStorage = "storage"
	AddonTokens  = "tokens"
	AddonCompany = "company"
	AddonMember  = "member"
)

// Статусы заказа.
const (
	OrderPending  = "pending"
	OrderPaid     = "paid"
	OrderCanceled = "canceled"
	OrderFailed   = "failed"
	OrderRefunded = "refunded"
)

// Виды заказа.
const (
	OrderKindSubscription = "subscription"
	OrderKindAddon        = "addon"
	OrderKindProduct      = "product"
)

// Статусы товара.
const (
	ProductDraft     = "draft"
	ProductReview    = "review"
	ProductPublished = "published"
	ProductRejected  = "rejected"
	ProductRemoved   = "removed"
)

// Статусы платежа.
const (
	PaymentPending   = "pending"
	PaymentSucceeded = "succeeded"
	PaymentCanceled  = "canceled"
	PaymentFailed    = "failed"
)

// Виды промокодов.
const (
	PromoPercent = "percent" // скидка в процентах
	PromoAmount  = "amount"  // скидка в копейках
	PromoDays    = "days"    // бесплатные дни тарифа PlanCode
	PromoTokens  = "tokens"  // пачка токенов доступа
)

// Plan — тариф витрины (лимиты берутся из PlanLimits по code).
type Plan struct {
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Tagline    string    `json:"tagline"`
	PriceMonth int64     `json:"price_month"`
	PriceYear  int64     `json:"price_year"`
	Sort       int       `json:"sort"`
	IsActive   bool      `json:"is_active"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Addon — докупка сверх тарифа.
type Addon struct {
	Code        string `json:"code"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Amount      int64  `json:"amount"`
	PriceMonth  int64  `json:"price_month"`
	PriceYear   int64  `json:"price_year"`
	Recurring   bool   `json:"recurring"`
	Sort        int    `json:"sort"`
	IsActive    bool   `json:"is_active"`
}

// Subscription — подписка пользователя. Отсутствие записи означает бесплатный
// «Джун», поэтому строку заводим только при покупке или выдаче.
type Subscription struct {
	UserID      int64      `json:"user_id"`
	PlanCode    string     `json:"plan_code"`
	Period      string     `json:"period"`
	Source      string     `json:"source"`
	StartedAt   time.Time  `json:"started_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	AutoRenew   bool       `json:"auto_renew"`
	CancelledAt *time.Time `json:"cancelled_at"`
	Note        string     `json:"note"`
}

// Active — действует ли подписка на момент now.
func (s *Subscription) Active(now time.Time) bool {
	if s == nil {
		return false
	}
	return s.ExpiresAt == nil || s.ExpiresAt.After(now)
}

// UserAddon — купленный аддон.
type UserAddon struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	AddonCode string     `json:"addon_code"`
	Kind      string     `json:"kind"`
	Amount    int64      `json:"amount"`
	Qty       int        `json:"qty"`
	CompanyID *int64     `json:"company_id"`
	Period    string     `json:"period"`
	StartedAt time.Time  `json:"started_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	AutoRenew bool       `json:"auto_renew"`
	Name      string     `json:"name"`
}

// Product — товар магазина: платформенный (AuthorID == nil) или авторский.
type Product struct {
	ID           int64          `json:"id"`
	Kind         string         `json:"kind"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Price        int64          `json:"price"`
	AuthorID     *int64         `json:"author_id"`
	AuthorName   string         `json:"author_name"`
	Status       string         `json:"status"`
	RejectReason string         `json:"reject_reason"`
	CoverPath    *string        `json:"cover_path"`
	Payload      map[string]any `json:"payload"`
	SalesCount   int            `json:"sales_count"`
	Sort         int            `json:"sort"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	PublishedAt  *time.Time     `json:"published_at"`
	// Owned — товар уже куплен зрителем витрины (заполняет сервис).
	Owned bool `json:"owned"`
}

// Promo — промокод.
type Promo struct {
	ID           int64      `json:"id"`
	Code         string     `json:"code"`
	Kind         string     `json:"kind"`
	Value        int64      `json:"value"`
	PlanCode     *string    `json:"plan_code"`
	AppliesTo    string     `json:"applies_to"`
	MaxUses      int        `json:"max_uses"`
	PerUserLimit int        `json:"per_user_limit"`
	UsedCount    int        `json:"used_count"`
	StartsAt     *time.Time `json:"starts_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
	IsActive     bool       `json:"is_active"`
	Comment      string     `json:"comment"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Usable — можно ли применить промокод сейчас (без учёта личных активаций).
func (p *Promo) Usable(now time.Time) bool {
	if p == nil || !p.IsActive {
		return false
	}
	if p.StartsAt != nil && p.StartsAt.After(now) {
		return false
	}
	if p.ExpiresAt != nil && !p.ExpiresAt.After(now) {
		return false
	}
	if p.MaxUses > 0 && p.UsedCount >= p.MaxUses {
		return false
	}
	return true
}

// Order — заказ (намерение купить).
type Order struct {
	ID         int64          `json:"id"`
	UserID     int64          `json:"user_id"`
	Kind       string         `json:"kind"`
	ItemCode   string         `json:"item_code"`
	ProductID  *int64         `json:"product_id"`
	Period     string         `json:"period"`
	Qty        int            `json:"qty"`
	CompanyID  *int64         `json:"company_id"`
	Amount     int64          `json:"amount"`
	BaseAmount int64          `json:"base_amount"`
	Discount   int64          `json:"discount"`
	PromoID    *int64         `json:"promo_id"`
	Status     string         `json:"status"`
	Title      string         `json:"title"`
	CreatedAt  time.Time      `json:"created_at"`
	PaidAt     *time.Time     `json:"paid_at"`
	AppliedAt  *time.Time     `json:"applied_at"`
	Meta       map[string]any `json:"meta"`
	Payment    *Payment       `json:"payment,omitempty"`
}

// Payment — платёж по заказу (реальный провайдер придёт на место заглушки).
type Payment struct {
	ID                int64     `json:"id"`
	OrderID           int64     `json:"order_id"`
	Provider          string    `json:"provider"`
	ProviderPaymentID string    `json:"provider_payment_id"`
	Amount            int64     `json:"amount"`
	Status            string    `json:"status"`
	Method            string    `json:"method"`
	ConfirmationURL   string    `json:"confirmation_url"`
	WebhookSecret     string    `json:"-"`
	CreatedAt         time.Time `json:"created_at"`
}

// ProductPurchase — купленный товар в разделе «Мои товары».
type ProductPurchase struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	UserID    int64     `json:"user_id"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
	Product   *Product  `json:"product,omitempty"`
}

// SellerBalance — кошелёк автора товаров.
type SellerBalance struct {
	UserID      int64 `json:"user_id"`
	Balance     int64 `json:"balance"`
	TotalEarned int64 `json:"total_earned"`
}

// Payout — заявка автора на вывод выручки (подтверждает супер-админ).
type Payout struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	UserName    string     `json:"user_name"`
	Amount      int64      `json:"amount"`
	Status      string     `json:"status"`
	Requisites  string     `json:"requisites"`
	Note        string     `json:"note"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at"`
}

// AIBalance — токены доступа к ИИ. PlanTokens — квота текущего периода,
// ExtraTokens — докупленные (не сгорают).
type AIBalance struct {
	UserID      int64     `json:"user_id"`
	PlanTokens  int64     `json:"plan_tokens"`
	UsedTokens  int64     `json:"used_tokens"`
	ExtraTokens int64     `json:"extra_tokens"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
}

// Left — сколько токенов ещё можно потратить.
func (b *AIBalance) Left() int64 {
	if b == nil {
		return 0
	}
	left := b.PlanTokens - b.UsedTokens + b.ExtraTokens
	if left < 0 {
		return 0
	}
	return left
}

// AIUsageRecord — одно обращение к модели.
type AIUsageRecord struct {
	UserID           int64
	CompanyID        *int64
	ActorID          *int64
	Feature          string
	Model            string
	PromptTokens     int
	CompletionTokens int
	BilledTokens     int64
	OwnKey           bool
}

// StorageEntry — занятое место в разрезе сервиса.
type StorageEntry struct {
	Service string `json:"service"`
	Bytes   int64  `json:"bytes"`
}

// OwnedFile — файл глазами сервиса-владельца (ответ ListFiles). Размера здесь
// нет: его знает журнал, а для незнакомых ключей биллинг меряет объект сам.
type OwnedFile struct {
	Key       string
	Name      string
	RefKind   string
	RefID     string
	RefTitle  string
	CompanyID int64
	CreatedAt time.Time
}

// StoredFile — запись журнала файлов: чем именно занято место. Ведётся в
// момент заливки (там размер и имя известны даром) и выверяется обходом
// сервисов-владельцев.
type StoredFile struct {
	Key     string `json:"key"`
	Service string `json:"service"`
	// CompanyID > 0 — файл компании: место тратится из квоты её создателя,
	// он же UserID записи.
	CompanyID int64  `json:"company_id,omitempty"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	// RefKind/RefID/RefTitle — где файл лежит: раздел «Хранилище» строит по ним
	// переход к источнику. Пустые — файл ещё не привязан к сущности.
	RefKind   string    `json:"ref_kind,omitempty"`
	RefID     string    `json:"ref_id,omitempty"`
	RefTitle  string    `json:"ref_title,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Settings — платформенные настройки биллинга.
type Settings struct {
	CommissionPct   int    `json:"commission_pct"`
	PaymentProvider string `json:"payment_provider"`
	PaymentEnabled  bool   `json:"payment_enabled"`
	StoreEnabled    bool   `json:"store_enabled"`
}

// AuditEntry — строка журнала действий супер-админа.
type AuditEntry struct {
	ID         int64          `json:"id"`
	ActorID    *int64         `json:"actor_id"`
	ActorName  string         `json:"actor_name"`
	Action     string         `json:"action"`
	TargetKind string         `json:"target_kind"`
	TargetID   string         `json:"target_id"`
	Summary    string         `json:"summary"`
	Payload    map[string]any `json:"payload"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Entitlements — что доступно пользователю здесь и сейчас: тариф, лимиты с
// учётом докупок и текущий расход. Это ответ, на который опираются остальные
// сервисы при проверке своих лимитов.
type Entitlements struct {
	UserID    int64      `json:"user_id"`
	Plan      string     `json:"plan"`
	PlanName  string     `json:"plan_name"`
	Source    string     `json:"source"`
	ExpiresAt *time.Time `json:"expires_at"`
	AutoRenew bool       `json:"auto_renew"`
	Limits    Limits     `json:"limits"`

	StorageUsed int64 `json:"storage_used"`
	// TokensLimit — квота ТЕКУЩЕГО периода. Не равна Limits.AITokens, пока
	// подписки скрыты: тогда всем выдаётся одинаковая суточная норма (AIQuota).
	TokensLimit int64 `json:"tokens_limit"`
	TokensUsed  int64 `json:"tokens_used"`
	TokensLeft  int64 `json:"tokens_left"`

	// OwnerID — чей тариф применён. Для компанийного запроса это создатель
	// компании, для личного — сам пользователь.
	OwnerID int64 `json:"owner_id"`
}

// User — минимальная идентичность (читаем таблицы authsvc только на чтение).
type User struct {
	ID           int64
	FIO          string
	Login        string
	IsActive     bool
	IsSuperAdmin bool
	Email        *string
}
