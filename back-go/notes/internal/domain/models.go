package domain

import (
	"encoding/json"
	"time"
)

// Доступ публичной ссылки: view — только чтение, edit — чтение и редактирование.
// AccessOwner — режим владельца в поле my_access ответов (полные права).
const (
	AccessView  = "view"
	AccessEdit  = "edit"
	AccessOwner = "owner"
)

// Note — личная заметка пользователя. Принадлежит ровно одному пользователю
// (OwnerID) и не зависит от компании (кросс-компанийная, как ежедневник);
// другим доступна только по публичной ссылке (Share). Doc — rich-документ
// TipTap (JSON); TextContent — плоский текст, пересчитывается сервером из Doc
// при каждом сохранении (поиск и txt-экспорт). В списках Doc не отдаётся —
// вместо него Excerpt (начало TextContent для плитки-стикера). Color — цвет
// плитки из набора тегов задач (” — без цвета).
type Note struct {
	ID       int64  `json:"id"`
	OwnerID  int64  `json:"owner_id"`
	Title    string `json:"title"`
	Color    string `json:"color"`
	Archived bool   `json:"archived"`
	// PinnedAt — закрепление (nil — не закреплена): закреплённые идут первыми
	// в списках владельца. Личное владельческое, в shared-списке не участвует.
	PinnedAt    *time.Time      `json:"pinned_at"`
	Doc         json.RawMessage `json:"doc,omitempty"`
	TextContent string          `json:"-"`
	Excerpt     string          `json:"excerpt"`
	GroupIDs    []int64         `json:"group_ids"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	// Owner*/MyAccess — заполняются в ответах для адресатов (список
	// «поделились со мной» и открытая чужая заметка) и в my_access у GetNote.
	OwnerName   string  `json:"owner_name,omitempty"`
	OwnerAvatar *string `json:"owner_avatar,omitempty"`
	// MyAccess — доступ текущего пользователя к заметке: owner | edit | view.
	MyAccess string `json:"my_access,omitempty"`
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

// NoteUpdate — частичная правка заметки: nil-поля не меняются. Color/Archived/
// Pinned — только владелец (личный стиль плитки); Title/Doc — владелец, адресат
// с can_edit или edit-ссылка.
type NoteUpdate struct {
	Title    *string
	Color    *string
	Archived *bool
	Pinned   *bool
	Doc      json.RawMessage
}

// NoteMember — пользователь, которому заметка открыта адресно. CanEdit —
// разрешено править title/doc (иначе только чтение).
type NoteMember struct {
	UserID     int64     `json:"user_id"`
	FIO        string    `json:"fio"`
	AvatarPath *string   `json:"avatar_path"`
	CanEdit    bool      `json:"can_edit"`
	CreatedAt  time.Time `json:"created_at"`
}

// CollabCursor — позиция выделения отправителя в документе (ProseMirror from/to).
type CollabCursor struct {
	From int `json:"from"`
	To   int `json:"to"`
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
