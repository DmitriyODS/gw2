package domain

import (
	"encoding/json"
	"time"
)

// DateLayout — формат даты дня записи (без времени): запись привязана к дню,
// время начала/конца — отдельные опциональные минуты от полуночи.
const DateLayout = "2006-01-02"

// Diary — личный ежедневник пользователя: набор записей-задач, привязанных к
// дню. Принадлежит ровно одному пользователю (OwnerID); другие видят его только
// через шаринг (read-only) — публичной ссылкой или адресно. Поля Owner*/Shared
// заполняются лишь для чужих ежедневников во вкладке «Поделились».
type Diary struct {
	ID          int64     `json:"id"`
	OwnerID     int64     `json:"owner_id"`
	Name        string    `json:"name"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	OwnerName   string    `json:"owner_name,omitempty"`
	OwnerAvatar *string   `json:"owner_avatar,omitempty"`
	Shared      bool      `json:"shared"`
}

// Entry — запись (заметка-задача) ежедневника. Date — день, к которому привязана
// запись (без времени). StartMin/EndMin — опциональное время начала/конца в
// минутах от полуночи (nil — без времени). Done — выполнена (уходит в архив).
// LinkedTaskID — связанная задача в tasksvc (создаётся кнопкой в карточке).
type Entry struct {
	ID           int64     `json:"-"`
	DiaryID      int64     `json:"-"`
	Date         time.Time `json:"-"`
	StartMin     *int      `json:"-"`
	EndMin       *int      `json:"-"`
	Title        string    `json:"-"`
	Description  string    `json:"-"`
	Done         bool      `json:"-"`
	LinkedTaskID *int64    `json:"-"`
	CreatedAt    time.Time `json:"-"`
	UpdatedAt    time.Time `json:"-"`
}

// MarshalJSON — день записи отдаётся как дата YYYY-MM-DD (без времени/таймзоны),
// чтобы клиент не «сдвигал» запись через границу суток в другом часовом поясе.
func (e Entry) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID           int64     `json:"id"`
		DiaryID      int64     `json:"diary_id"`
		EntryDate    string    `json:"entry_date"`
		StartMin     *int      `json:"start_min"`
		EndMin       *int      `json:"end_min"`
		Title        string    `json:"title"`
		Description  string    `json:"description"`
		Done         bool      `json:"done"`
		LinkedTaskID *int64    `json:"linked_task_id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
	}{
		ID: e.ID, DiaryID: e.DiaryID, EntryDate: e.Date.Format(DateLayout),
		StartMin: e.StartMin, EndMin: e.EndMin, Title: e.Title, Description: e.Description,
		Done: e.Done, LinkedTaskID: e.LinkedTaskID, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	})
}

// EntryListFilter — выборка записей одного ежедневника. Archived делит на
// вкладки: false — активные (за диапазон дат для просмотра по дню/неделе/месяцу),
// true — архив выполненных (весь, без диапазона).
type EntryListFilter struct {
	DiaryID  int64
	Archived bool
	Search   string
	From     *time.Time // включительно (только для активных)
	To       *time.Time // НЕ включительно (только для активных)
	Limit    int
}

// Share — публичная ссылка на ежедневник (read-only, без авторизации). Code в
// URL — capability.
type Share struct {
	ID        int64     `json:"id"`
	DiaryID   int64     `json:"diary_id"`
	Code      string    `json:"code"`
	CreatedBy *int64    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// Member — пользователь, которому адресно открыт ежедневник (read-only).
type Member struct {
	UserID     int64     `json:"user_id"`
	FIO        string    `json:"fio"`
	AvatarPath *string   `json:"avatar_path"`
	CreatedAt  time.Time `json:"created_at"`
}

// User — идентичность пользователя для авторизации.
type User struct {
	ID           int64
	FIO          string
	AvatarPath   *string
	IsActive     bool
	IsSuperAdmin bool
}
