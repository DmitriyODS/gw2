// Package bridge — доставка сокет-событий микросервисов клиентам.
//
// Каждый сервис публикует события в свой Redis-канал gw2:<svc>:events в
// общем envelope {"event": ..., "rooms": [...], "payload": {...}} —
// мост подписан на все каналы и транслирует события в WS-комнаты вербатим
// (порт прежних Flask-мостов service_bridge/call_bridge). Имена на "_"
// зарезервированы под служебные хуки и наружу не эмитятся.
//
// Сам gateway (presence, ринг-фаза) публикует свои события в
// gw2:gateway:events тем же envelope — единый путь доставки работает и при
// нескольких инстансах шлюза. Соединение с Redis самовосстанавливается:
// обрыв — пауза и переподключение.
package bridge

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/DmitriyODS/gw2/back-go/gateway/internal/hub"
)

// Channels — каналы событий всех микросервисов платформы.
var Channels = []string{
	"gw2:calls:events",
	"gw2:messenger:events",
	"gw2:groove:events",
	"gw2:tasks:events",
	"gw2:registry:events",
	"gw2:calendar:events",
	"gw2:diary:events",
	"gw2:gateway:events",
}

const reconnectDelay = 3 * time.Second

type envelope struct {
	Event   string          `json:"event"`
	Rooms   []string        `json:"rooms"`
	Payload json.RawMessage `json:"payload"`
}

type Bridge struct {
	rdb *redis.Client
	hub *hub.Hub
	log *slog.Logger
}

func New(rdb *redis.Client, h *hub.Hub, log *slog.Logger) *Bridge {
	return &Bridge{rdb: rdb, hub: h, log: log}
}

// Run — блокирующий цикл подписки; завершается по ctx.
func (b *Bridge) Run(ctx context.Context) {
	for {
		if err := b.listen(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			b.log.Warn("bridge.connection_lost", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func (b *Bridge) listen(ctx context.Context) error {
	pubsub := b.rdb.Subscribe(ctx, Channels...)
	defer pubsub.Close()
	b.log.Info("bridge.subscribed", "channels", Channels)

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return context.Canceled
			}
			b.handle(msg.Payload)
		}
	}
}

func (b *Bridge) handle(raw string) {
	var ev envelope
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		b.log.Warn("bridge.bad_event", "error", err)
		return
	}
	if ev.Event == "" {
		return
	}
	if strings.HasPrefix(ev.Event, "_") {
		b.log.Warn("bridge.unknown_internal", "event", ev.Event)
		return
	}
	frame := hub.MarshalFrame(ev.Event, ev.Payload)
	for _, room := range ev.Rooms {
		b.hub.Broadcast(room, frame)
	}
}
