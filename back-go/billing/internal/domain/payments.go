package domain

// PaymentProvider — платёжный шлюз. Сейчас в проде живёт заглушка manual
// (счёт подтверждает супер-админ), а подключение СБП конкретного банка — это
// новая реализация этого интерфейса и строка в фабрике: ни сервис, ни фронт
// при этом не меняются.
type PaymentProvider interface {
	// Name — код провайдера, он же пишется в billing_payments.provider.
	Name() string
	// Enabled — готов ли провайдер принимать деньги (у заглушки — нет:
	// витрина показывает «оплата скоро подключится»).
	Enabled() bool
	// Create — создать платёж по заказу. secret — одноразовый код, которым
	// вебхук докажет, что говорит именно об этом платеже (для настоящего
	// провайдера его место займёт подпись запроса).
	Create(ctx Ctx, order *Order, secret string) (*ProviderPayment, error)
	// Parse — разобрать тело вебхука провайдера.
	Parse(ctx Ctx, body []byte, headers map[string]string) (*WebhookEvent, error)
}

// ProviderPayment — то, что вернул шлюз при создании платежа.
type ProviderPayment struct {
	ProviderPaymentID string
	ConfirmationURL   string // ссылка/QR СБП для оплаты
	Status            string
}

// WebhookEvent — разобранное уведомление шлюза об изменении статуса платежа.
type WebhookEvent struct {
	ProviderPaymentID string
	Status            string // PaymentSucceeded | PaymentCanceled | PaymentFailed
	Secret            string
	Raw               map[string]any
}
