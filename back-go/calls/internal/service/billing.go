package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

// Число участников группового звонка задаёт тариф ИНИЦИАТОРА: он собирает
// людей, его подписка и определяет размер комнаты. Жёсткий потолок
// domain.MaxParticipants остаётся страховкой SFU — тариф не может его
// превысить. Биллинг не подключён или недоступен — действует прежний потолок.

// WithBilling — подключить лимиты тарифа.
func (s *Service) WithBilling(billing *billingclient.Client) *Service {
	s.billing = billing
	return s
}

// maxParticipants — сколько человек помещается в звонок инициатора.
func (s *Service) maxParticipants(ctx context.Context, initiatorID int64) int {
	if s.billing == nil || initiatorID <= 0 {
		return domain.MaxParticipants
	}
	limit := int(s.billing.Entitlements(ctx, initiatorID, 0).Limits.GetCallParticipants())
	if limit == billingclient.Unlimited || limit > domain.MaxParticipants {
		return domain.MaxParticipants
	}
	if limit < 1 {
		return 1
	}
	return limit
}
