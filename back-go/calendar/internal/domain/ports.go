package domain

import (
	"context"
	"io"
	"time"
)

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// CalendarRepository — персистентность календарей, их полей и записей.
type CalendarRepository interface {
	// ── Календари ──
	ListCalendars(ctx Ctx, companyID int64) ([]*Calendar, error)
	// GetCalendar — календарь без полей (для проверок принадлежности).
	GetCalendar(ctx Ctx, id int64) (*Calendar, error)
	// CountCalendars — сколько календарей уже есть (лимит тарифа).
	CountCalendars(ctx Ctx, company_id int64) (int, error)
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
	// CompanyEntries — ближайшие записи ВСЕХ календарей компании за период
	// (живая плитка рабочего стола): один запрос вместо обхода календарей.
	// Возвращает срез (не длиннее limit) и общее число записей за период.
	CompanyEntries(ctx Ctx, companyID int64, from, to time.Time, limit int) (rows []AgendaRow, total int, err error)
	// EntriesForExport — записи для выгрузки: при непустом ids — только они,
	// иначе все по фильтру (диапазон дат + поиск). Порядок по event_at.
	EntriesForExport(ctx Ctx, f EntryListFilter, ids []int64) ([]*Entry, error)
	// EntriesOfCompanies — записи вместе с их календарём: раздел «Настройки →
	// Хранилище» показывает, в каком календаре лежит файл. Скоуп — компании,
	// чью квоту оплачивает спрашивающий (их присылает биллинг).
	EntriesOfCompanies(ctx Ctx, companyIDs []int64) ([]*EntryScope, error)

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

// FileStore — хранение загруженных файлов/картинок (общий uploads-том или S3).
type FileStore interface {
	// SaveFor — записать файл в квоту компании (её оплачивает создатель):
	// сверх лимита тарифа файл не сохраняется. Возвращает ключ хранилища.
	SaveFor(ctx context.Context, userID, companyID int64, fileName string, data []byte) (string, error)
	// SaveStreamFor — то же потоком: файл, пришедший частями, нельзя подержать
	// в памяти целиком.
	SaveStreamFor(ctx context.Context, userID, companyID int64, fileName string, r io.Reader, size int64) (string, error)
	// RemoveFor — best-effort удаление файлов по ключам с возвратом места в
	// квоту (чистка при удалении записей/полей); ошибки не возвращаются.
	RemoveFor(ctx context.Context, userID, companyID int64, paths []string)
	// Remove — удаление БЕЗ учёта: так чистит раздел «Хранилище», где место
	// пересчитывает сам биллинг (он же инициатор и знает размеры).
	Remove(paths []string)
}

// EventBus — сокет-события клиентам через Redis gw2:calendar:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}
