package domain

import (
	"encoding/json"
	"time"
)

// Доступ к заметке/папке: view — только чтение, edit — чтение и редактирование.
// AccessOwner — режим владельца в поле my_access ответов (полные права).
const (
	AccessView  = "view"
	AccessEdit  = "edit"
	AccessOwner = "owner"
)

// ShareTarget — аудитория адресного шаринга: конкретный пользователь платформы
// или целая компания (виден всем её сотрудникам).
const (
	TargetUser    = "user"
	TargetCompany = "company"
)

// Note — личная заметка пользователя. Принадлежит ровно одному пользователю
// (OwnerID) и не зависит от компании (кросс-компанийная). Лежит РОВНО в одной
// папке (FolderID; nil — корень). Doc — rich-документ TipTap (JSON); TextContent
// — плоский текст, пересчитывается сервером из Doc при каждом сохранении (поиск и
// txt-экспорт). В списках Doc не отдаётся — вместо него Excerpt. Color — цвет
// плитки из набора тегов задач (” — без цвета).
type Note struct {
	ID       int64  `json:"id"`
	OwnerID  int64  `json:"owner_id"`
	Title    string `json:"title"`
	Color    string `json:"color"`
	Archived bool   `json:"archived"`
	// FolderID — папка заметки (nil — корень/без папки).
	FolderID *int64 `json:"folder_id"`
	// PinnedAt — закрепление (nil — не закреплена): закреплённые идут первыми
	// в списках владельца. Личное владельческое, в shared-списке не участвует.
	PinnedAt    *time.Time      `json:"pinned_at"`
	Doc         json.RawMessage `json:"doc,omitempty"`
	TextContent string          `json:"-"`
	Excerpt     string          `json:"excerpt"`
	// TagIDs — теги заметки (личные метки владельца, many-to-many).
	TagIDs    []int64   `json:"tag_ids"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// SharedByMe — заметка расшарена мной кому-либо (для значка на плитке
	// владельца). Заполняется в владельческих списках.
	SharedByMe bool `json:"shared_by_me,omitempty"`
	// Owner*/MyAccess — заполняются в ответах для адресатов (список
	// «поделились со мной» и открытая чужая заметка) и в my_access у GetNote.
	OwnerName   string  `json:"owner_name,omitempty"`
	OwnerAvatar *string `json:"owner_avatar,omitempty"`
	// MyAccess — доступ текущего пользователя к заметке: owner | edit | view.
	MyAccess string `json:"my_access,omitempty"`
}

// NoteColors — допустимые цвета плитки/папки (синхронно с front/src/utils/
// taskColors.js и domain.TaskColors tasksvc); пустая строка — без цвета.
var NoteColors = map[string]bool{
	"red": true, "orange": true, "amber": true, "green": true,
	"teal": true, "blue": true, "violet": true, "pink": true,
}

// Folder — иерархическая папка заметок владельца. ParentID nil — корень. Доступ
// по расшаренной папке каскадит на всё поддерево (эффективный доступ считается
// подъёмом по ParentID).
type Folder struct {
	ID         int64     `json:"id"`
	OwnerID    int64     `json:"owner_id"`
	ParentID   *int64    `json:"parent_id"`
	Name       string    `json:"name"`
	Color      string    `json:"color"`
	Position   int       `json:"position"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	NotesCount int       `json:"notes_count"`
	// SharedByMe — папка расшарена мной (значок на владельческой карточке).
	SharedByMe bool `json:"shared_by_me,omitempty"`
	// Owner*/MyAccess — для расшаренных мне папок («Поделились со мной»).
	OwnerName   string  `json:"owner_name,omitempty"`
	OwnerAvatar *string `json:"owner_avatar,omitempty"`
	MyAccess    string  `json:"my_access,omitempty"`
}

// Tag — личная метка заметок владельца (бывшие «группы»): many-to-many, с цветом
// из палитры --tag-*. Заметка может иметь несколько тегов, один или ни одного.
type Tag struct {
	ID         int64     `json:"id"`
	OwnerID    int64     `json:"owner_id"`
	Name       string    `json:"name"`
	Color      string    `json:"color"`
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

// Member — адресат шаринга заметки или папки: пользователь платформы либо целая
// компания. Для пользователя заполнены UserID/FIO/Avatar, для компании —
// CompanyID/CompanyName. CanEdit — чтение+редактирование, иначе только чтение.
type Member struct {
	Target      string    `json:"target"` // user | company
	UserID      int64     `json:"user_id,omitempty"`
	FIO         string    `json:"fio,omitempty"`
	AvatarPath  *string   `json:"avatar_path,omitempty"`
	CompanyID   int64     `json:"company_id,omitempty"`
	CompanyName string    `json:"company_name,omitempty"`
	CanEdit     bool      `json:"can_edit"`
	CreatedAt   time.Time `json:"created_at"`
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

// CollabCursor — позиция выделения отправителя в документе (ProseMirror from/to).
type CollabCursor struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// NoteListFilter — выборка плиток. OwnerID>0 — только заметки владельца (свой
// раздел); OwnerID==0 — без фильтра по владельцу (просмотр чужой расшаренной
// папки, доступ проверяет сервис). FolderSet — фильтровать по папке (FolderID
// nil при FolderSet=true — корень «без папки»). TagIDs — хотя бы один из тегов.
type NoteListFilter struct {
	OwnerID   int64
	FolderID  *int64
	FolderSet bool
	TagIDs    []int64
	Search    string
	Archived  bool
}

// User — идентичность пользователя для авторизации.
type User struct {
	ID           int64
	FIO          string
	AvatarPath   *string
	IsActive     bool
	IsSuperAdmin bool
}

// Company — членство пользователя в компании (для выбора аудитории шаринга и
// скоупа «расшарено моей компании»). Читается напрямую из таблиц authsvc.
type Company struct {
	ID   int64
	Name string
}
