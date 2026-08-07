// Package payments — реализации платёжного шлюза. Пока их одна: заглушка
// manual, которая заводит счёт и ждёт подтверждения супер-админа. Реальный СБП
// (ЮKassa, Т-Банк, СБП напрямую) добавляется сюда же отдельным файлом и
// строкой в фабрике — остальной код биллинга про провайдера не знает.
package payments

import (
	"encoding/json"
	"fmt"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

// Manual — заглушка: платёж создаётся в статусе pending и подтверждается
// вручную из «Аудита платформы» (или вебхуком с тем же секретом — так удобно
// проверять сквозной сценарий до подключения банка).
type Manual struct{}

var _ domain.PaymentProvider = (*Manual)(nil)

func NewManual() *Manual { return &Manual{} }

func (m *Manual) Name() string  { return "manual" }
func (m *Manual) Enabled() bool { return false }

func (m *Manual) Create(_ domain.Ctx, order *domain.Order, _ string) (*domain.ProviderPayment, error) {
	return &domain.ProviderPayment{
		ProviderPaymentID: fmt.Sprintf("manual-%d", order.ID),
		Status:            domain.PaymentPending,
	}, nil
}

// Parse — тело вида {"payment_id": "...", "status": "succeeded", "secret": "..."}.
func (m *Manual) Parse(_ domain.Ctx, body []byte, _ map[string]string) (*domain.WebhookEvent, error) {
	var in struct {
		PaymentID string `json:"payment_id"`
		Status    string `json:"status"`
		Secret    string `json:"secret"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, err
	}
	if in.Status == "" {
		in.Status = domain.PaymentSucceeded
	}
	return &domain.WebhookEvent{
		ProviderPaymentID: in.PaymentID,
		Status:            in.Status,
		Secret:            in.Secret,
		Raw:               map[string]any{"payment_id": in.PaymentID, "status": in.Status},
	}, nil
}

// New — провайдер по коду из настроек платформы. Неизвестный код — заглушка:
// без оплаты магазин остаётся витриной, а не падает.
func New(code string) domain.PaymentProvider {
	switch code {
	default:
		return NewManual()
	}
}
