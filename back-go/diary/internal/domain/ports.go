package domain

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pkg/spaces"
)

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// DiaryRepository — персистентность ежедневников, их записей и шаринга.
type DiaryRepository interface {
	// ── Ежедневники ──
	// ListOwned — ежедневники пространств человека: личные и всех его команд
	// (вкладка «Мои»), без скрытого «Моего дня».
	ListOwned(ctx Ctx, userID int64) ([]*Diary, error)
	// ListShared — чужие ежедневники, открытые пользователю адресно (вкладка
	// «Поделились»), с именем/аватаром владельца.
	ListShared(ctx Ctx, userID int64) ([]*Diary, error)
	GetDiary(ctx Ctx, id int64) (*Diary, error)
	// AccessOf — эффективный уровень человека к ежедневнику ("" — нет доступа).
	AccessOf(ctx Ctx, diaryID, userID int64) (string, error)
	// Audience — кому адресовать события ежедневника.
	Audience(ctx Ctx, diaryID int64) ([]int64, error)
	// MyDay — скрытый «Мой день» человека (заводится при первом обращении).
	MyDay(ctx Ctx, ownerID int64) (*Diary, error)
	// MoveDiary — сменить пространство, хозяина и уровень участников.
	MoveDiary(ctx Ctx, id, ownerID int64, companyID *int64, teamAccess string) error
	// CountDiaries — сколько ежедневников уже есть (лимит тарифа).
	CountDiaries(ctx Ctx, owner_id int64) (int, error)
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
	// SearchEntries — глобальный поиск по записям всех доступных пользователю
	// ежедневников (свои + расшаренные адресно): один запрос, без N+1.
	SearchEntries(ctx Ctx, userID int64, query string, limit int) ([]*SearchHit, error)
	// Agenda — невыполненные записи всех доступных ежедневников за период
	// (живая плитка «Ежедневники» на рабочем столе): один запрос, без N+1.
	// Возвращает записи по возрастанию дня и времени начала и общее их число.
	Agenda(ctx Ctx, userID int64, from, to time.Time, limit int) (items []*SearchHit, total int, err error)
	// TodayEntries — невыполненные дела по день day включительно из всех
	// доступных ежедневников плюс дела «Мого дня» без срока, с названиями
	// ежедневников.
	TodayEntries(ctx Ctx, userID int64, day time.Time, limit int) ([]*Entry, map[int64]string, error)
	// DoneOn — сколько дел закрыто за промежуток [from, to).
	DoneOn(ctx Ctx, userID int64, from, to time.Time) (int, error)

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
	// TeamRole — положение человека в команде (можно ли положить в неё ежедневник).
	TeamRole(ctx Ctx, userID, companyID int64) (spaces.Role, error)
}

// EventBus — сокет-события клиентам через Redis gw2:diary:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}
