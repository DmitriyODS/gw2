// Package redisx — Redis-адаптеры callsvc.
package redisx

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
)

// webhookSeenTTL — LiveKit повторяет доставку минутами, сутки — с запасом.
const webhookSeenTTL = 24 * time.Hour

// WebhookDedup — отметки обработанных событий LiveKit (SETNX по id).
type WebhookDedup struct {
	rdb *redis.Client
	log *slog.Logger
}

var _ domain.WebhookDedup = (*WebhookDedup)(nil)

func NewWebhookDedup(rdb *redis.Client, log *slog.Logger) *WebhookDedup {
	return &WebhookDedup{rdb: rdb, log: log}
}

// FirstSeen — true, если событие пришло впервые. Redis недоступен — true:
// обработчики идемпотентны, повтор безопаснее потери события.
func (d *WebhookDedup) FirstSeen(ctx context.Context, eventID string) bool {
	ok, err := d.rdb.SetNX(ctx, "gw2:calls:webhook:"+eventID, 1, webhookSeenTTL).Result()
	if err != nil {
		d.log.Warn("livekit.webhook_dedup_unavailable", "error", err)
		return true
	}
	return ok
}
