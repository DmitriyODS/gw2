package domain

import (
	"context"
	"time"
)

// DeviceLink — pending-сессия «спаривания устройств» (живёт в Redis, короткий
// TTL). Одна модель на два сценария (Kind):
//   - login: устройство без входа показывает код/QR, уже авторизованный телефон
//     подтверждает — устройство входит под тем же аккаунтом (обычный login-gate);
//   - tv:    ТВ-киоск показывает код/QR, авторизованный пользователь из настроек
//     подтверждает — киоск входит под ВЫБРАННОЙ им компанией (CompanyID).
//
// Инициатор держит секрет и опрашивает claim; подтверждающий проставляет UserID
// (и CompanyID для tv). Секрет в хранилище — только как sha256.
type DeviceLink struct {
	Kind       string `json:"kind"`        // login | tv
	SecretHash string `json:"secret_hash"` // sha256 от секрета инициатора
	Status     string `json:"status"`      // pending | approved
	UserID     int64  `json:"user_id"`     // кто подтвердил (0 пока pending)
	CompanyID  *int64 `json:"company_id"`  // для tv — компания киоска
}

const (
	LinkKindLogin = "login"
	LinkKindTV    = "tv"

	LinkStatusPending  = "pending"
	LinkStatusApproved = "approved"
)

// DeviceLinkStore — Redis-хранилище pending-спариваний (ключи gw2:devlink:*).
type DeviceLinkStore interface {
	// Save — записать/перезаписать состояние с TTL.
	Save(ctx context.Context, code string, dl DeviceLink, ttl time.Duration) error
	// Get — состояние по коду; nil без ошибки, если код неизвестен/истёк.
	Get(ctx context.Context, code string) (*DeviceLink, error)
	Delete(ctx context.Context, code string) error
}
