package domain

import (
	"encoding/json"
	"time"
)

// Доступ к доске/папке: view — только чтение, edit — чтение и редактирование.
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

// Board — личная доска пользователя. Принадлежит ровно одному пользователю
// (OwnerID) и не зависит от компании (кросс-компанийная). Лежит РОВНО в одной
// папке (FolderID; nil — корень). Scene — сцена рисования (объекты холста, JSON);
// TextContent — плоский текст надписей и стикеров, пересчитывается сервером из
// Scene при каждом сохранении (поиск и превью). В списках Scene не отдаётся —
// вместо неё Excerpt и картинка-превью. Color — цвет плитки из набора тегов
// задач (” — без цвета).
type Board struct {
	ID       int64  `json:"id"`
	OwnerID  int64  `json:"owner_id"`
	Title    string `json:"title"`
	Color    string `json:"color"`
	Archived bool   `json:"archived"`
	// FolderID — папка доски (nil — корень/без папки).
	FolderID *int64 `json:"folder_id"`
	// PinnedAt — закрепление (nil — не закреплена): закреплённые идут первыми
	// в списках владельца. Личное владельческое, в shared-списке не участвует.
	PinnedAt    *time.Time      `json:"pinned_at"`
	Scene       json.RawMessage `json:"scene,omitempty"`
	TextContent string          `json:"-"`
	Excerpt     string          `json:"excerpt"`
	// PreviewPath — ключ миниатюры холста в хранилище (растр снимает клиент при
	// сохранении); PreviewURL — она же для клиента (/uploads/<key>).
	PreviewPath string    `json:"-"`
	PreviewURL  string    `json:"preview_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// SharedByMe — доска расшарена мной кому-либо (для значка на плитке
	// владельца). Заполняется в владельческих списках.
	SharedByMe bool `json:"shared_by_me,omitempty"`
	// Owner*/MyAccess — заполняются в ответах для адресатов (список
	// «поделились со мной» и открытая чужая доска) и в my_access у GetBoard.
	OwnerName   string  `json:"owner_name,omitempty"`
	OwnerAvatar *string `json:"owner_avatar,omitempty"`
	// MyAccess — доступ текущего пользователя к доске: owner | edit | view.
	MyAccess string `json:"my_access,omitempty"`
}

// FillPreviewURL — развернуть ключ миниатюры в адрес для клиента (/uploads/…).
func (b *Board) FillPreviewURL() {
	if b.PreviewPath != "" {
		b.PreviewURL = "/uploads/" + b.PreviewPath
	}
}

// BoardColors — допустимые цвета плитки/папки (синхронно с front/src/utils/
// taskColors.js и domain.TaskColors tasksvc); пустая строка — без цвета.
var BoardColors = map[string]bool{
	"red": true, "orange": true, "amber": true, "green": true,
	"teal": true, "blue": true, "violet": true, "pink": true,
}

// Folder — иерархическая папка досок владельца. ParentID nil — корень. Доступ
// по расшаренной папке каскадит на всё поддерево (эффективный доступ считается
// подъёмом по ParentID).
type Folder struct {
	ID          int64     `json:"id"`
	OwnerID     int64     `json:"owner_id"`
	ParentID    *int64    `json:"parent_id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	BoardsCount int       `json:"boards_count"`
	// SharedByMe — папка расшарена мной (значок на владельческой карточке).
	SharedByMe bool `json:"shared_by_me,omitempty"`
	// Owner*/MyAccess — для расшаренных мне папок («Поделились со мной»).
	OwnerName   string  `json:"owner_name,omitempty"`
	OwnerAvatar *string `json:"owner_avatar,omitempty"`
	MyAccess    string  `json:"my_access,omitempty"`
}

// Share — публичная ссылка на доску (без авторизации). Code в URL —
// capability; Access — режим доступа (view|edit).
type Share struct {
	ID        int64     `json:"id"`
	BoardID   int64     `json:"board_id"`
	Code      string    `json:"code"`
	Access    string    `json:"access"`
	CreatedAt time.Time `json:"created_at"`
}

// Member — адресат шаринга доски или папки: пользователь платформы либо целая
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

// BoardUpdate — частичная правка доски: nil-поля не меняются. Color/Archived/
// Pinned — только владелец (личный стиль плитки); Title/Scene — владелец, адресат
// с can_edit или edit-ссылка.
type BoardUpdate struct {
	Title    *string
	Color    *string
	Archived *bool
	Pinned   *bool
	Scene    json.RawMessage
}

// CollabCursor — точка внимания соавтора на холсте (координаты сцены).
type CollabCursor struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// BoardListFilter — выборка плиток. OwnerID>0 — только доски владельца (свой
// раздел); OwnerID==0 — без фильтра по владельцу (просмотр чужой расшаренной
// папки, доступ проверяет сервис). FolderSet — фильтровать по папке (FolderID
// nil при FolderSet=true — корень «без папки»).
type BoardListFilter struct {
	OwnerID   int64
	FolderID  *int64
	FolderSet bool
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
