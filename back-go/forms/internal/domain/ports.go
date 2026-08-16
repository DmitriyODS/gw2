package domain

import (
	"context"
	"io"
	"time"
)

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// FormRepository — персистентность форм, их структуры, ответов, доступов и
// внешних ссылок.
type FormRepository interface {
	// ── Формы ──
	// ListForms — формы области: свои, назначенные, совместные. Уровень
	// доступа, число ответов и собственная обязанность считаются тем же
	// запросом (список раздела показывает всё это сразу).
	ListForms(ctx Ctx, userID int64, companyIDs []int64, scope string) ([]*Form, error)
	// GetForm — форма без структуры и без проверки доступа (её делает сервис).
	GetForm(ctx Ctx, id int64) (*Form, error)
	CountOwned(ctx Ctx, ownerID int64) (int, error)
	CreateForm(ctx Ctx, f *Form) error
	UpdateForm(ctx Ctx, f *Form) error
	DeleteForm(ctx Ctx, id int64) error
	NextPosition(ctx Ctx, ownerID int64) (int, error)
	// SearchForms — глобальный поиск (строка Hola) по названиям и описаниям
	// доступных форм.
	SearchForms(ctx Ctx, userID int64, companyIDs []int64, query string, limit int) ([]*SearchHit, error)

	// ── Структура ──
	// ListSections — разделы формы вместе с вопросами, в порядке показа.
	ListSections(ctx Ctx, formID int64) ([]Section, error)
	// GetQuestion — один вопрос формы (потолок размера файла задаёт он сам).
	GetQuestion(ctx Ctx, formID, questionID int64) (*Question, error)
	/* ReplaceStructure — полная замена разделов и вопросов в одной транзакции:
	   удаляет отсутствующие, обновляет существующие, вставляет новые. Возвращает
	   id удалённых вопросов — их значения нужно вычистить из ответов. */
	ReplaceStructure(ctx Ctx, formID int64, sections []Section) (removed []int64, err error)

	// ── Ответы ──
	ListResponses(ctx Ctx, f ResponseListFilter) (items []*Response, total int, err error)
	GetResponse(ctx Ctx, id int64) (*Response, error)
	// ResponseOfUser — ответ конкретного человека (свой ответ и «уже отвечал»).
	ResponseOfUser(ctx Ctx, formID, userID int64) (*Response, error)
	CountResponses(ctx Ctx, formID int64) (int, error)
	/* CreateResponse / UpdateResponse — сохранить ответ вместе с производной
	   строкой поиска (её считает сервис по актуальной структуре формы).

	   bookings — занимаемые места вопросов «Запись». Их остаток проверяется В
	   ТОЙ ЖЕ транзакции под локом формы: между «посмотрели остаток» и «записали
	   ответ» последнее место мог занять другой человек. */
	CreateResponse(ctx Ctx, r *Response, searchText string, bookings []Booking) error
	UpdateResponse(ctx Ctx, r *Response, searchText string, bookings []Booking) error
	// BookingCounts — сколько мест уже занято: {ключ вопроса: {вариант: занято}}.
	// exceptResponse > 0 — не считать свой прежний ответ (правка брони).
	BookingCounts(ctx Ctx, formID int64, questionKeys []string, exceptResponse int64) (map[string]map[string]int, error)
	// PublishGrades — открыть оценки теста отвечающим (режим ручной проверки).
	PublishGrades(ctx Ctx, formID int64, responseID int64) error
	DeleteResponse(ctx Ctx, formID, id int64) error
	// DeleteResponses — массовое удаление: перечисленные либо все ответы формы.
	// Возвращает удалённые (id — событию, значения — чистке файлов).
	DeleteResponses(ctx Ctx, formID int64, ids []int64, all bool) ([]*Response, error)
	// EachResponse — обход ВСЕХ ответов формы потоком: сводка и выгрузка не
	// должны держать в памяти всю переписку разом.
	EachResponse(ctx Ctx, formID int64, fn func(*Response) error) error
	// SetResponsesSingle — синхронизировать копию настройки «один ответ» в уже
	// собранных ответах (её сторожит частичный уникальный индекс).
	SetResponsesSingle(ctx Ctx, formID int64, single bool) error
	// ResponsesOfOwner — ответы вместе с их формой: раздел «Настройки →
	// Хранилище» показывает, к какой форме приложен файл.
	ResponsesOfOwner(ctx Ctx, userID int64, companyIDs []int64) ([]*ResponseScope, error)

	// ── Доступ ──
	// AccessOf — эффективный уровень человека к форме (лучший из личной шары и
	// шар его компаний; владельцу — AccessOwner).
	AccessOf(ctx Ctx, formID, userID int64, companyIDs []int64) (string, error)
	// Audience — кому адресовать сокет-события формы.
	Audience(ctx Ctx, formID int64) ([]int64, error)
	ListUserShares(ctx Ctx, formID int64) ([]*UserShare, error)
	// PutUserShare — выдать или обновить адресный доступ (повторная выдача
	// меняет уровень и срок, а не плодит строки).
	PutUserShare(ctx Ctx, s *UserShare) error
	DeleteUserShare(ctx Ctx, formID int64, userID, companyID *int64) error
	// Assignees — назначенные заполнить форму, с отметкой, кто уже ответил
	// (контроль исполнения). Развёрнуто поимённо, включая участников компаний.
	Assignees(ctx Ctx, formID int64) ([]*Assignee, error)
	/* ClaimDueReminders — атомарно забрать наступившие сроки ответа: строка
	   помечается напомненной тем же запросом, поэтому при нескольких инстансах
	   сервиса напоминание уходит ровно один раз. */
	ClaimDueReminders(ctx Ctx, now time.Time, limit int) ([]DueReminder, error)

	// ── Внешние ссылки ──
	CreateShare(ctx Ctx, s *Share) error
	ListShares(ctx Ctx, formID int64) ([]*Share, error)
	GetShareByCode(ctx Ctx, code string) (*Share, error)
	UpdateShare(ctx Ctx, id, formID int64, name string, requireAuth bool) error
	DeleteShare(ctx Ctx, id, formID int64) error
	LogVisit(ctx Ctx, v *ShareVisit) error
	ListVisits(ctx Ctx, shareID int64, limit int) ([]*ShareVisit, error)
}

// UserReader — read-only идентичность пользователей (владелец таблицы — authsvc).
type UserReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
	CompanyActive(ctx Ctx, companyID *int64) (bool, error)
	// CompaniesOf — компании, где человек состоит: через них приходит доступ к
	// формам, назначенным компании целиком.
	CompaniesOf(ctx Ctx, userID int64) ([]int64, error)
	// CompanyMembers — участники компании: их извещают о назначенной ей форме.
	CompanyMembers(ctx Ctx, companyID int64) ([]int64, error)
	// SearchDirectory — кандидаты в адресаты из компаний спрашивающего.
	SearchDirectory(ctx Ctx, companyIDs []int64, query string, limit int) ([]*User, error)
	// CompanyName — название компании для карточки выданного доступа.
	CompanyName(ctx Ctx, companyID int64) (string, error)
}

// FileStore — хранение приложенных к ответам файлов (uploads-том или S3).
type FileStore interface {
	// SaveFor — записать файл в квоту владельца: сверх лимита тарифа файл не
	// сохраняется. Возвращает ключ хранилища.
	SaveFor(ctx context.Context, userID, companyID int64, fileName string, data []byte) (string, error)
	// SaveStreamFor — то же потоком: большой файл нельзя подержать в памяти.
	SaveStreamFor(ctx context.Context, userID, companyID int64, fileName string, r io.Reader, size int64) (string, error)
	// RemoveFor — best-effort удаление по ключам с возвратом места в квоту.
	RemoveFor(ctx context.Context, userID, companyID int64, paths []string)
	// Remove — удаление БЕЗ учёта: так чистит раздел «Хранилище», где место
	// пересчитывает сам биллинг.
	Remove(paths []string)
}

// EventBus — сокет-события клиентам через Redis gw2:forms:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}
