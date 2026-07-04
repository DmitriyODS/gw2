package domain

import (
	"context"
	"time"
)

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// DiaryRepository — персистентность ежедневников, их записей и шаринга.
type DiaryRepository interface {
	// ── Ежедневники ──
	// ListOwned — личные ежедневники владельца (вкладка «Мои»).
	ListOwned(ctx Ctx, ownerID int64) ([]*Diary, error)
	// ListShared — чужие ежедневники, открытые пользователю адресно (вкладка
	// «Поделились»), с именем/аватаром владельца. Read-only.
	ListShared(ctx Ctx, userID int64) ([]*Diary, error)
	GetDiary(ctx Ctx, id int64) (*Diary, error)
	CreateDiary(ctx Ctx, d *Diary) error
	UpdateDiary(ctx Ctx, id int64, name string) error
	DeleteDiary(ctx Ctx, id int64) error
	NextPosition(ctx Ctx, ownerID int64) (int, error)

	// ── Записи ──
	ListEntries(ctx Ctx, f EntryListFilter) ([]*Entry, error)
	GetEntry(ctx Ctx, id int64) (*Entry, error)
	CreateEntry(ctx Ctx, e *Entry, searchText string) error
	UpdateEntry(ctx Ctx, e *Entry, searchText string) error
	SetEntryDone(ctx Ctx, id int64, done bool) error
	SetEntryTask(ctx Ctx, id int64, taskID *int64) error
	// MoveEntry — перенос записи на другой день и/или в другой ежедневник
	// (drag-and-drop). Дата — начало дня.
	MoveEntry(ctx Ctx, id, diaryID int64, date time.Time) error
	// ReorderEntries — ручной порядок записей дня: ids в желаемом порядке
	// получают position 1..N (в рамках ежедневника и дня).
	ReorderEntries(ctx Ctx, diaryID int64, date time.Time, ids []int64) error
	DeleteEntry(ctx Ctx, id int64) error
	DeleteEntries(ctx Ctx, diaryID int64, ids []int64) (int64, error)
	EntriesForExport(ctx Ctx, f EntryListFilter, ids []int64) ([]*Entry, error)

	// ── Публичные ссылки ──
	CreateShare(ctx Ctx, s *Share) error
	ListShares(ctx Ctx, diaryID int64) ([]*Share, error)
	GetShareByCode(ctx Ctx, code string) (*Share, error)
	DeleteShare(ctx Ctx, id, diaryID int64) error

	// ── Адресный доступ (поделиться с пользователем) ──
	ListMembers(ctx Ctx, diaryID int64) ([]*Member, error)
	// MemberIDs — id пользователей с адресным доступом (для адресации сокет-событий).
	MemberIDs(ctx Ctx, diaryID int64) ([]int64, error)
	// MemberAccess — адресный доступ пользователя: есть ли он и разрешено ли
	// ему отмечать записи выполненными.
	MemberAccess(ctx Ctx, diaryID, userID int64) (found bool, canCheck bool, err error)
	// AddMember — идемпотентный upsert: повторный вызов обновляет can_check.
	AddMember(ctx Ctx, diaryID, userID int64, canCheck bool) error
	RemoveMember(ctx Ctx, diaryID, userID int64) error
}

// UserReader — read-only идентичность пользователей (владелец таблицы — authsvc).
type UserReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
}

// EventBus — сокет-события клиентам через Redis gw2:diary:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}
