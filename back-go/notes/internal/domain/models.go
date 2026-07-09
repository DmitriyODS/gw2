package domain

import (
	"encoding/json"
	"time"
)

// Доступ публичной ссылки: view — только чтение, edit — чтение и редактирование.
const (
	AccessView = "view"
	AccessEdit = "edit"
)

// Note — личная заметка пользователя. Принадлежит ровно одному пользователю
// (OwnerID) и не зависит от компании (кросс-компанийная, как ежедневник);
// другим доступна только по публичной ссылке (Share). Doc — rich-документ
// TipTap (JSON); TextContent — плоский текст, пересчитывается сервером из Doc
// при каждом сохранении (поиск и txt-экспорт). В списках Doc не отдаётся —
// вместо него Excerpt (начало TextContent для плитки-стикера). Color — цвет
// плитки из набора тегов задач ('' — без цвета).
type Note struct {
	ID          int64           `json:"id"`
	OwnerID     int64           `json:"owner_id"`
	Title       string          `json:"title"`
	Color       string          `json:"color"`
	Archived    bool            `json:"archived"`
	Doc         json.RawMessage `json:"doc,omitempty"`
	TextContent string          `json:"-"`
	Excerpt     string          `json:"excerpt"`
	GroupIDs    []int64         `json:"group_ids"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// NoteColors — допустимые цвета плитки (синхронно с front/src/utils/
// taskColors.js и domain.TaskColors tasksvc); пустая строка — без цвета.
var NoteColors = map[string]bool{
	"red": true, "orange": true, "amber": true, "green": true,
	"teal": true, "blue": true, "violet": true, "pink": true,
}

// Group — личная группа (папка-фильтр) заметок владельца. Заметка может
// входить в несколько групп, одну или ни одной; удаление группы не трогает
// заметки — только связи.
type Group struct {
	ID         int64     `json:"id"`
	OwnerID    int64     `json:"owner_id"`
	Name       string    `json:"name"`
	Position   int       `json:"position"`
	NotesCount int       `json:"notes_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// Share — публичная ссылка на заметку (без авторизации). Code в URL —
// capability; Access — режим доступа (view|edit).
type Share struct {
	ID        int64     `json:"id"`
	NoteID    int64     `json:"note_id"`
	Code      string    `json:"code"`
	Access    string    `json:"access"`
	CreatedAt time.Time `json:"created_at"`
}

// NoteListFilter — выборка плиток владельца: по группе (0 — все) и сквозному
// поиску по заголовку+тексту.
type NoteListFilter struct {
	OwnerID  int64
	GroupID  int64
	Search   string
	Archived bool
}

// User — идентичность пользователя для авторизации.
type User struct {
	ID           int64
	FIO          string
	AvatarPath   *string
	IsActive     bool
	IsSuperAdmin bool
}
