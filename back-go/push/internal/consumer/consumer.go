// Package consumer — подписка на Redis-каналы событий микросервисов и
// передача их в сервис рассылки пушей. Зеркалит мост gatewaysvc, но вместо
// доставки в WS строит пуш-уведомления офлайн-получателям.
package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Channels — каналы, из которых берём события для пушей:
//
//	messenger — message:new; tasks — task:created; gateway — call:incoming
//	(ринг-фазу звонков публикует gatewaysvc в свой канал); pets —
//	kudos:received (входящий перевод кудо-банка); portal — post:new.
var Channels = []string{
	"gw2:messenger:events",
	"gw2:tasks:events",
	"gw2:gateway:events",
	"gw2:pets:events",
	"gw2:portal:events",
}

const reconnectDelay = 3 * time.Second

type Dispatcher interface {
	Dispatch(ctx context.Context, event string, payload json.RawMessage, rooms []string)
}

type envelope struct {
	Event   string          `json:"event"`
	Rooms   []string        `json:"rooms"`
	Payload json.RawMessage `json:"payload"`
}

type Consumer struct {
	rdb        *redis.Client
	dispatcher Dispatcher
	log        *slog.Logger
}

func New(rdb *redis.Client, d Dispatcher, log *slog.Logger) *Consumer {
	return &Consumer{rdb: rdb, dispatcher: d, log: log}
}

// Run — блокирующий цикл с самовосстановлением; завершается по ctx.
func (c *Consumer) Run(ctx context.Context) {
	for {
		if err := c.listen(ctx); err != nil && ctx.Err() == nil {
			c.log.Warn("consumer.connection_lost", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func (c *Consumer) listen(ctx context.Context) error {
	pubsub := c.rdb.Subscribe(ctx, Channels...)
	defer pubsub.Close()
	c.log.Info("consumer.subscribed", "channels", Channels)

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			var env envelope
			if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
				c.log.Warn("consumer.bad_envelope", "channel", msg.Channel, "error", err)
				continue
			}
			c.dispatcher.Dispatch(ctx, env.Event, env.Payload, env.Rooms)
		}
	}
}
