package domain

import "context"

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// CalendarRepository — персистентность календарей, их полей и записей.
type CalendarRepository interface {
	// ── Календари ──
	ListCalendars(ctx Ctx, companyID int64) ([]*Calendar, error)
	// GetCalendar — календарь без полей (для проверок принадлежности).
	GetCalendar(ctx Ctx, id int64) (*Calendar, error)
	CreateCalendar(ctx Ctx, c *Calendar) error
	UpdateCalendar(ctx Ctx, id int64, name string, position int) error
	DeleteCalendar(ctx Ctx, id int64) error
	NextCalendarPosition(ctx Ctx, companyID int64) (int, error)

	// ── Поля ──
	ListFields(ctx Ctx, calendarID int64) ([]Field, error)
	// FieldsByCalendars — батч-загрузка полей для списка календарей (без N+1).
	FieldsByCalendars(ctx Ctx, calendarIDs []int64) (map[int64][]Field, error)
	// ReplaceFields — полная замена набора полей календаря в одной транзакции:
	// удаляет отсутствующие, обновляет существующие, вставляет новые. Возвращает
	// id удалённых полей (их данные нужно вычистить из записей).
	ReplaceFields(ctx Ctx, calendarID int64, fields []Field) (removed []int64, err error)

	// ── Записи ──
	ListEntries(ctx Ctx, f EntryListFilter) ([]*Entry, error)
	GetEntry(ctx Ctx, id int64) (*Entry, error)
	CreateEntry(ctx Ctx, e *Entry, searchText string) error
	UpdateEntry(ctx Ctx, id int64, eventAt any, data map[string]any, searchText string) error
	DeleteEntry(ctx Ctx, id int64) error
	// DeleteEntries — массовое удаление; возвращает число удалённых.
	DeleteEntries(ctx Ctx, calendarID int64, ids []int64) (int64, error)
	// AllEntries — все записи календаря (для пересчёта search_text после удаления поля).
	AllEntries(ctx Ctx, calendarID int64) ([]*Entry, error)
	// EntriesForExport — записи для выгрузки: при непустом ids — только они,
	// иначе все по фильтру (диапазон дат + поиск). Порядок по event_at.
	EntriesForExport(ctx Ctx, f EntryListFilter, ids []int64) ([]*Entry, error)

	// ── Публичные ссылки ──
	CreateShare(ctx Ctx, s *Share) error
	ListShares(ctx Ctx, calendarID int64) ([]*Share, error)
	GetShareByCode(ctx Ctx, code string) (*Share, error)
	DeleteShare(ctx Ctx, id, calendarID int64) error
}

// UserReader — read-only идентичность пользователей (владелец таблицы — authsvc).
type UserReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
	CompanyActive(ctx Ctx, companyID *int64) (bool, error)
}

// FileStore — хранение загруженных файлов/картинок в общем uploads-томе.
type FileStore interface {
	// Save — записать файл, вернуть относительный путь в uploads.
	Save(fileName string, data []byte) (string, error)
}

// EventBus — сокет-события клиентам через Redis gw2:calendar:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}
