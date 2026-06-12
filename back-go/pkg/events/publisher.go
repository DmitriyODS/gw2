// Package events — публикация Socket.IO-событий микросервисов для Flask-моста.
//
// Socket.IO остаётся во Flask: generic-мост (sockets/service_bridge.py)
// слушает Redis-каналы gw2:<svc>:events и эмитит обычные события вербатим
// в каждую комнату из rooms; события с префиксом "_" (служебные хуки)
// наружу не эмитятся, а диспатчатся в python-обработчики.
//
// Канал звонков (gw2:calls:events) живёт в историческом формате —
// см. back-go/calls/internal/events.
package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

type Publisher struct {
	rdb     *redis.Client
	log     *slog.Logger
	channel string
}

// NewPublisher — публикатор в Redis-канал channel (вида "gw2:<svc>:events").
func NewPublisher(rdb *redis.Client, log *slog.Logger, channel string) *Publisher {
	return &Publisher{rdb: rdb, log: log, channel: channel}
}

// envelope — обобщённый формат событий микросервисов:
// {"event": "message:new", "rooms": ["user_12"], "payload": {...}}.
type envelope struct {
	Event   string   `json:"event"`
	Rooms   []string `json:"rooms"`
	Payload any      `json:"payload"`
}

func (p *Publisher) Publish(ctx context.Context, event string, rooms []string, payload any) {
	if rooms == nil {
		rooms = []string{}
	}
	raw, err := json.Marshal(envelope{Event: event, Rooms: rooms, Payload: payload})
	if err != nil {
		p.log.Error("events.marshal_failed", "event", event, "error", err)
		return
	}
	if err := p.rdb.Publish(ctx, p.channel, raw).Err(); err != nil {
		// Потеря события не фатальна: клиент дотянет состояние при
		// следующем REST-запросе/переподключении.
		p.log.Warn("events.publish_failed", "event", event, "error", err)
	}
}
