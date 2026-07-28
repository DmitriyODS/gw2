package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

// Лимит тарифа на число папок чатов — ЛИЧНЫЙ (папки принадлежат пользователю,
// а мессенджер кросс-компанийный). Биллинг не подключён или недоступен —
// ограничений нет (fail-open).

// WithBilling — подключить проверку лимитов тарифа.
func (s *Service) WithBilling(billing *billingclient.Client) *Service {
	s.billing = billing
	return s
}

// ensureUploadSpace — влезает ли вложение в квоту хранилища отправителя
// (мессенджер кросс-компанийный, поэтому место всегда личное).
func (s *Service) ensureUploadSpace(ctx context.Context, userID int64, size int64) error {
	if s.billing == nil {
		return nil
	}
	return s.billing.EnsureStorage(ctx, userID, 0, size)
}

// trackUpload — учесть занятое (delta > 0) или освобождённое (delta < 0) место.
func (s *Service) trackUpload(ctx context.Context, userID, delta int64) {
	if s.billing == nil || userID <= 0 || delta == 0 {
		return
	}
	s.billing.TrackStorage(ctx, userID, 0, "messenger", delta)
}

// ensureFolderLimit — влезает ли ещё одна папка чатов в тариф пользователя.
// current — сколько папок уже есть (сервис его и так считает).
func (s *Service) ensureFolderLimit(ctx context.Context, userID int64, current int) error {
	if s.billing == nil {
		return nil
	}
	ent := s.billing.Entitlements(ctx, userID, 0)
	return billingclient.EnsureCount("chat_folders", int(ent.Limits.GetChatFolders()), current, ent.PlanName)
}
