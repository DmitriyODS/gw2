// Package events — публикация событий звонков для Flask-шлюза.
//
// Flask слушает Redis-канал и транслирует события в Socket.IO-комнаты
// пользователей (call:ended, message:updated для плашки в чате). Канал нужен
// только для изменений, которые инициирует сам сервис (вебхуки LiveKit):
// результат сокет-команд Flask получает синхронно в ответе gRPC.
package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
)

// Channel — канал Redis pub/sub, который слушает Flask (sockets/call_bridge.py).
const Channel = "gw2:calls:events"

type Publisher struct {
	rdb *redis.Client
	log *slog.Logger
}

var _ domain.EventPublisher = (*Publisher)(nil)

func NewPublisher(rdb *redis.Client, log *slog.Logger) *Publisher {
	return &Publisher{rdb: rdb, log: log}
}

func (p *Publisher) publish(ctx context.Context, payload map[string]any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		p.log.Error("events.marshal_failed", "error", err)
		return
	}
	if err := p.rdb.Publish(ctx, Channel, raw).Err(); err != nil {
		// Потеря события не фатальна: фронт самовосстанавливается через
		// GET /api/calls/active (checkRejoin) при следующем reconnect.
		p.log.Warn("events.publish_failed", "error", err)
	}
}

func (p *Publisher) CallEnded(ctx context.Context, callID int64, status string, notifyUserIDs []int64) {
	if notifyUserIDs == nil {
		notifyUserIDs = []int64{}
	}
	p.publish(ctx, map[string]any{
		"type":            "call_ended",
		"call_id":         callID,
		"status":          status,
		"notify_user_ids": notifyUserIDs,
	})
}

func (p *Publisher) CallStatusChanged(ctx context.Context, callID int64) {
	p.publish(ctx, map[string]any{
		"type":    "call_status_changed",
		"call_id": callID,
	})
}
