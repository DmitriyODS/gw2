// Package events — сокет-события звонков для gatewaysvc.
//
// Публикация в Redis-канал gw2:calls:events в общем envelope микросервисов
// {"event": ..., "rooms": [...], "payload": {...}} (pkg/events) — gateway
// транслирует их в WS-комнаты вербатим, как и события остальных сервисов.
//
// Плашка звонка в чате (kind='call') — тоже здесь: PillCreated/PillUpdated
// ходят в msgsvc по gRPC за снапшотом сообщения и рассылают
// message:new/message:updated адресатам. Fire-and-forget в горутине: плашка
// вторична, её ошибки звонок не роняют и ответ ринг-команды не задерживают.
package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
)

// Channel — канал Redis pub/sub, который слушает gatewaysvc.
const Channel = "gw2:calls:events"

const pillTimeout = 10 * time.Second

// MessengerPill — срез msgsvc-клиента для плашек (internal/clients.Messenger).
type MessengerPill interface {
	CreateCallMessage(ctx context.Context, conversationID, senderID, callID int64) (string, []int64, error)
	GetCallMessage(ctx context.Context, callID int64) (int64, string, []int64, error)
}

type Publisher struct {
	bus  *events.Publisher
	msgr MessengerPill
	log  *slog.Logger
}

var _ domain.EventPublisher = (*Publisher)(nil)

func NewPublisher(bus *events.Publisher, msgr MessengerPill, log *slog.Logger) *Publisher {
	return &Publisher{bus: bus, msgr: msgr, log: log}
}

func userRooms(ids []int64) []string {
	rooms := make([]string, 0, len(ids))
	for _, id := range ids {
		rooms = append(rooms, "user_"+strconv.FormatInt(id, 10))
	}
	return rooms
}

func (p *Publisher) CallEnded(ctx context.Context, callID int64, status string, notifyUserIDs []int64) {
	p.bus.Publish(ctx, "call:ended", userRooms(notifyUserIDs), map[string]any{
		"call_id": callID, "status": status,
	})
}

// PillCreated — создать плашку в парном диалоге и разослать message:new
// (фронт рендерит её по kind='call').
func (p *Publisher) PillCreated(_ context.Context, conversationID, senderID, callID int64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), pillTimeout)
		defer cancel()
		msgJSON, notify, err := p.msgr.CreateCallMessage(ctx, conversationID, senderID, callID)
		if err != nil {
			p.log.Warn("call.pill_create_failed", "call_id", callID, "error", err)
			return
		}
		p.bus.Publish(ctx, "message:new", userRooms(notify), map[string]any{
			"conversation_id": conversationID,
			"message":         json.RawMessage(msgJSON),
			"from_user_id":    senderID,
		})
	}()
}

// PillUpdated — перечитать снапшот плашки (статус ringing → active →
// ended/missed) и разослать message:updated. Плашки нет (group-звонок) или
// msgsvc недоступен — пропускаем.
func (p *Publisher) PillUpdated(_ context.Context, callID int64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), pillTimeout)
		defer cancel()
		convID, msgJSON, notify, err := p.msgr.GetCallMessage(ctx, callID)
		if err != nil {
			p.log.Warn("call.pill_update_skipped", "call_id", callID, "error", err)
			return
		}
		if msgJSON == "" {
			return
		}
		p.bus.Publish(ctx, "message:updated", userRooms(notify), map[string]any{
			"conversation_id": convID,
			"message":         json.RawMessage(msgJSON),
		})
	}()
}
