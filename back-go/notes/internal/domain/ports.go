package domain

import "context"

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// NoteRepository — персистентность заметок, групп и публичных ссылок.
type NoteRepository interface {
	// ── Заметки ──
	// ListNotes — плитки владельца (без doc, с excerpt и group_ids),
	// сортировка updated_at DESC.
	ListNotes(ctx Ctx, f NoteListFilter) ([]*Note, error)
	// GetNote — полная заметка (с doc и group_ids); nil — нет такой.
	GetNote(ctx Ctx, id int64) (*Note, error)
	CreateNote(ctx Ctx, n *Note) error
	// UpdateNote — title/doc/text_content + updated_at = now().
	UpdateNote(ctx Ctx, n *Note) error
	DeleteNote(ctx Ctx, id int64) error
	// SetNoteGroups — полная замена связей заметки с группами.
	SetNoteGroups(ctx Ctx, noteID int64, groupIDs []int64) error

	// ── Группы ──
	ListGroups(ctx Ctx, ownerID int64) ([]*Group, error)
	GetGroup(ctx Ctx, id int64) (*Group, error)
	CreateGroup(ctx Ctx, g *Group) error
	UpdateGroup(ctx Ctx, id int64, name string) error
	// DeleteGroup — удаляет группу и связи; сами заметки не трогает (FK CASCADE
	// только на note_group_items).
	DeleteGroup(ctx Ctx, id int64) error
	NextGroupPosition(ctx Ctx, ownerID int64) (int, error)
	// OwnedGroupIDs — из ids оставить только группы владельца (чужие/несуществующие
	// молча отбрасываются при сохранении связей заметки).
	OwnedGroupIDs(ctx Ctx, ownerID int64, ids []int64) ([]int64, error)

	// ── Публичные ссылки ──
	ListShares(ctx Ctx, noteID int64) ([]*Share, error)
	CreateShare(ctx Ctx, s *Share) error
	GetShareByCode(ctx Ctx, code string) (*Share, error)
	DeleteShare(ctx Ctx, id, noteID int64) error

	// ── Адресный шаринг ──
	// ListMembers — адресаты заметки с ФИО/аватаром (JOIN users).
	ListMembers(ctx Ctx, noteID int64) ([]*NoteMember, error)
	// UpsertMember — идемпотентно открыть доступ / поменять право.
	UpsertMember(ctx Ctx, noteID, userID int64, canEdit bool) error
	DeleteMember(ctx Ctx, noteID, userID int64) error
	// GetMember — (есть ли адресный доступ, can_edit).
	GetMember(ctx Ctx, noteID, userID int64) (found, canEdit bool, err error)
	// MemberIDs — user_id всех адресатов заметки (адресация сокет-событий).
	MemberIDs(ctx Ctx, noteID int64) ([]int64, error)
	// ListSharedWithMe — чужие заметки, открытые пользователю адресно: плитки
	// без doc, с владельцем (owner_name/owner_avatar) и my_access (edit|view).
	ListSharedWithMe(ctx Ctx, userID int64, search string) ([]*Note, error)
}

// UserReader — read-only идентичность пользователей (владелец таблицы — authsvc).
type UserReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
}

// EventBus — сокет-события клиентам через Redis gw2:notes:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}

// FileStore — хранилище картинок редактора (pkg/records.FileStore поверх
// pkg/storage: local-том в dev, S3 в prod).
type FileStore interface {
	Save(fileName string, data []byte) (string, error)
	Remove(paths []string)
}

// WriteLimiter — троттлинг анонимных правок по коду публичной ссылки (защита
// от вандализма). Redis-реализация fail-open: при недоступности — разрешаем.
type WriteLimiter interface {
	Allow(ctx Ctx, code string) bool
}
