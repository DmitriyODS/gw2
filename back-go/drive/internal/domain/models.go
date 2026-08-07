package domain

import "time"

/* Диск — личное файловое хранилище: папки, файлы, корзина и шаринг.
   По устройству близнец заметок: владелец один, компания ни при чём, доступ
   коллегам выдаётся шарингом. Отличие в предмете — вместо документа лежит
   объект в хранилище, поэтому у файла есть ключ, размер и MIME. */

// Folder — папка диска. ParentID nil — корень.
type Folder struct {
	ID        int64      `json:"id"`
	OwnerID   int64      `json:"owner_id"`
	ParentID  *int64     `json:"parent_id"`
	Name      string     `json:"name"`
	Color     string     `json:"color,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	// Служебные поля выдачи (в БД их нет).
	Shared   bool   `json:"shared,omitempty"`    // владелец кому-то её открыл
	MyAccess string `json:"my_access,omitempty"` // owner | edit | view
	// OwnerName — чья папка; заполняется во вкладке «Поделились со мной».
	OwnerName string `json:"owner_name,omitempty"`
}

// File — файл диска. StorageKey — ключ объекта в хранилище, он же хвост
// адреса /uploads/<key>; наружу отдаётся именно адрес, а не ключ.
type File struct {
	ID        int64      `json:"id"`
	OwnerID   int64      `json:"owner_id"`
	FolderID  *int64     `json:"folder_id"`
	Name      string     `json:"name"`
	Key       string     `json:"-"`
	URL       string     `json:"url"`
	Mime      string     `json:"mime"`
	Size      int64      `json:"size"`
	Starred   bool       `json:"starred"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	Shared    bool   `json:"shared,omitempty"`
	MyAccess  string `json:"my_access,omitempty"`
	OwnerName string `json:"owner_name,omitempty"`
}

// Share — публичная ссылка на файл или папку (код в адресе — capability).
type Share struct {
	ID        int64     `json:"id"`
	FileID    *int64    `json:"file_id,omitempty"`
	FolderID  *int64    `json:"folder_id,omitempty"`
	Code      string    `json:"code"`
	CreatedBy int64     `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// UserShare — адресный доступ: человеку или всей компании.
type UserShare struct {
	ID        int64     `json:"id"`
	FileID    *int64    `json:"file_id,omitempty"`
	FolderID  *int64    `json:"folder_id,omitempty"`
	UserID    *int64    `json:"user_id,omitempty"`
	CompanyID *int64    `json:"company_id,omitempty"`
	CanEdit   bool      `json:"can_edit"`
	CreatedAt time.Time `json:"created_at"`

	// Подписи для списка доступа (JOIN на стороне репозитория).
	UserName    string `json:"user_name,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
}

// Уровни доступа. Пустая строка — доступа нет вовсе.
const (
	AccessOwner = "owner"
	AccessEdit  = "edit"
	AccessView  = "view"
)

// AccessAtLeast — покрывает ли have требуемый уровень want.
func AccessAtLeast(have, want string) bool {
	rank := map[string]int{AccessView: 1, AccessEdit: 2, AccessOwner: 3}
	return rank[have] >= rank[want] && rank[have] > 0
}

// Listing — содержимое папки: подпапки, файлы и путь до корня (хлебные крошки).
type Listing struct {
	Folder  *Folder   `json:"folder"`
	Path    []*Folder `json:"path"`
	Folders []*Folder `json:"folders"`
	Files   []*File   `json:"files"`
}

// ListFilter — что показывать в разделе.
type ListFilter struct {
	OwnerID  int64
	FolderID *int64
	// Search — поиск по имени; ГЛОБАЛЬНЫЙ (папка не ограничивает выдачу), как
	// в заметках: искомое чаще лежит не там, где сейчас открыто.
	Search string
	// Trash — показать корзину вместо обычного содержимого.
	Trash bool
	// Starred/Recent — вкладки «Избранное» и «Недавние».
	Starred bool
	Recent  bool
}

// MaxFileSize — потолок одного файла (как у вложений мессенджера).
const MaxFileSize = 500 << 20

// TrashKeepDays — сколько корзина хранит удалённое, прежде чем сервис
// вычистит его сам.
const TrashKeepDays = 30
