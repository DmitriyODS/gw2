package domain

import (
	"context"
	"time"
)

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// ReminderRepository — персистентность напоминаний.
type ReminderRepository interface {
	List(ctx Ctx, ownerID int64, scope ListScope) ([]*Reminder, error)
	// Upcoming — ближайшие активные напоминания владельца (живая плитка).
	Upcoming(ctx Ctx, ownerID int64, until time.Time, limit int) ([]*Reminder, error)
	// ByLink — напоминания владельца, привязанные к записи раздела (раздел
	// правит запись → обновляет снимок в своих напоминаниях).
	ByLink(ctx Ctx, ownerID int64, kind string, recordID int64) ([]*Reminder, error)
	Get(ctx Ctx, id int64) (*Reminder, error)
	Create(ctx Ctx, r *Reminder) error
	Update(ctx Ctx, r *Reminder) error
	Delete(ctx Ctx, id int64) error
	// ClaimDue — атомарно забрать сработавшие напоминания (remind_at <= now,
	// active): помечает их отработанными, чтобы ни один инстанс сервиса не
	// доставил одно и то же напоминание дважды. Пересчёт следующего срока —
	// в сервисе (правило повтора — доменное знание), запись — Update.
	ClaimDue(ctx Ctx, now time.Time, limit int) ([]*Reminder, error)
}

// UserReader — read-only идентичность пользователей (владелец таблицы в
// рантайме — authsvc; читаем напрямую из общей БД, как остальные сервисы).
type UserReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
}

// EventBus — сокет-события клиентам через Redis gw2:reminder:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим; pushsvc
// слушает тот же канал и шлёт FCM-пуш офлайн-получателю).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}
