package domain

import (
	"encoding/json"
	"time"
)

// DateLayout — формат даты дня записи (без времени): запись привязана к дню,
// время начала/конца — отдельные опциональные минуты от полуночи.
const DateLayout = "2006-01-02"

// Diary — ежедневник: набор записей-задач, привязанных к дню. Лежит в
// пространстве (см. access.go): личном — тогда OwnerID его хозяин, — либо
// команды (CompanyID), где OwnerID — автор. Другие видят его ещё и через
// шаринг — публичной ссылкой или адресно. Поля Owner*/Shared заполняются для
// чужих ежедневников во вкладке «Поделились».
type Diary struct {
	ID      int64 `json:"id"`
	OwnerID int64 `json:"owner_id"`
	// CompanyID — пространство: nil — личное, иначе ежедневник команды.
	CompanyID *int64 `json:"company_id"`
	// TeamAccess — уровень рядовых участников команды.
	TeamAccess string `json:"team_access"`
	// Kind — regular либо скрытый my_day экрана «Сегодня».
	Kind        string    `json:"kind"`
	Name        string    `json:"name"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	OwnerName   string    `json:"owner_name,omitempty"`
	OwnerAvatar *string   `json:"owner_avatar,omitempty"`
	Shared      bool      `json:"shared"`
	// CanCheck — для чужого ежедневника: адресату разрешено отмечать записи
	// выполненными (сценарий «руководитель раздаёт задачи»). У владельца всегда true.
	CanCheck bool `json:"can_check"`
	// ActiveCount/DoneCount — прогресс списка (активные/выполненные записи),
	// заполняются в списках ежедневников.
	ActiveCount int `json:"active_count"`
	DoneCount   int `json:"done_count"`
	// MyAccess — эффективный уровень спрашивающего; считает сервер.
	MyAccess string `json:"my_access"`
	// CompanyName — название команды-пространства (список группируется по ним).
	CompanyName string `json:"company_name,omitempty"`
}

// Entry — запись (заметка-задача) ежедневника. Date — день, к которому привязана
// запись (без времени); нулевая — записи «Мого дня» без срока («Потом»).
// StartMin/EndMin — опциональное время начала/конца в минутах от полуночи
// (nil — без времени). Done — выполнена (уходит в архив). LinkedTaskID —
// связанная задача в tasksvc, Attachments — вещи других разделов.
type Entry struct {
	ID           int64        `json:"-"`
	DiaryID      int64        `json:"-"`
	Date         time.Time    `json:"-"`
	StartMin     *int         `json:"-"`
	EndMin       *int         `json:"-"`
	Title        string       `json:"-"`
	Description  string       `json:"-"`
	Done         bool         `json:"-"`
	LinkedTaskID *int64       `json:"-"`
	Attachments  []Attachment `json:"-"`
	// Position — ручной порядок внутри дня (0 — не упорядочено, сортируется по
	// времени после упорядоченных; reorder проставляет 1..N).
	Position  int       `json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// MarshalJSON — день записи отдаётся как дата YYYY-MM-DD (без времени/таймзоны),
// чтобы клиент не «сдвигал» запись через границу суток в другом часовом поясе.
func (e Entry) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID           int64        `json:"id"`
		DiaryID      int64        `json:"diary_id"`
		EntryDate    *string      `json:"entry_date"`
		StartMin     *int         `json:"start_min"`
		EndMin       *int         `json:"end_min"`
		Title        string       `json:"title"`
		Description  string       `json:"description"`
		Done         bool         `json:"done"`
		LinkedTaskID *int64       `json:"linked_task_id"`
		Attachments  []Attachment `json:"attachments"`
		Position     int          `json:"position"`
		CreatedAt    time.Time    `json:"created_at"`
		UpdatedAt    time.Time    `json:"updated_at"`
	}{
		ID: e.ID, DiaryID: e.DiaryID, EntryDate: FormatDay(e.Date),
		StartMin: e.StartMin, EndMin: e.EndMin, Title: e.Title, Description: e.Description,
		Done: e.Done, LinkedTaskID: e.LinkedTaskID, Attachments: attachmentsOrEmpty(e.Attachments),
		Position: e.Position, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	})
}

// FormatDay — день записи датой YYYY-MM-DD; nil у записи без срока.
func FormatDay(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.Format(DateLayout)
	return &s
}

func attachmentsOrEmpty(a []Attachment) []Attachment {
	if a == nil {
		return []Attachment{}
	}
	return a
}

// Today — экран «Сегодня» со стороны ежедневников: невыполненные дела дня из
// всех доступных ежедневников, дела «Потом» (без срока) из «Моего дня» и
// сколько дел за день уже закрыто. Diaries — названия ежедневников по id:
// карточке нужно подписать, откуда дело.
type Today struct {
	MyDayID int64            `json:"my_day_id"`
	Items   []*Entry         `json:"items"`
	Later   []*Entry         `json:"later"`
	Done    int              `json:"done"`
	Diaries map[int64]string `json:"diaries"`
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

// Member — пользователь, которому адресно открыт ежедневник. CanCheck —
// разрешено отмечать записи выполненными (иначе только чтение).
type Member struct {
	UserID     int64     `json:"user_id"`
	FIO        string    `json:"fio"`
	AvatarPath *string   `json:"avatar_path"`
	CanCheck   bool      `json:"can_check"`
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

// SearchHit — строка глобального поиска (Spotlight): запись вместе с
// ежедневником, которому она принадлежит.
type SearchHit struct {
	DiaryID   int64     `json:"diary_id"`
	DiaryName string    `json:"diary_name"`
	EntryID   int64     `json:"entry_id"`
	Title     string    `json:"title"`
	Date      time.Time `json:"-"`
	Done      bool      `json:"done"`
	// Начало записи в минутах от полуночи — им живая плитка рабочего стола
	// показывает ближайшее дело («в 14:30 …»). У поиска всегда nil.
	StartMin *int `json:"start_min,omitempty"`
}

// Agenda — сводка невыполненных дел за период для живой плитки рабочего стола:
// ближайшие записи и сколько их всего (плитка показывает «ещё N»).
type Agenda struct {
	Items []*SearchHit `json:"items"`
	Total int          `json:"total"`
}

// MarshalJSON — день записи отдаётся датой YYYY-MM-DD, как и у Entry.
func (h SearchHit) MarshalJSON() ([]byte, error) {
	type alias SearchHit
	return json.Marshal(struct {
		alias
		Date *string `json:"entry_date"`
	}{alias(h), FormatDay(h.Date)})
}
