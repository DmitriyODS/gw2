package domain

import "context"

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// ScheduleRepository — персистентность расписаний, их структуры, занятий и шаринга.
type ScheduleRepository interface {
	// ── Расписания ──
	// ListOwned — свои расписания (вкладка «Мои»).
	ListOwned(ctx Ctx, ownerID int64) ([]*Schedule, error)
	// ListShared — чужие расписания, открытые пользователю лично или его
	// компании (вкладка «Поделились»), с именем владельца. Read-only.
	ListShared(ctx Ctx, userID int64, companyIDs []int64) ([]*Schedule, error)
	GetSchedule(ctx Ctx, id int64) (*Schedule, error)
	// HasAccess — открыто ли расписание пользователю адресно (лично или его
	// компании). Владение проверяется отдельно — по owner_id.
	HasAccess(ctx Ctx, scheduleID, userID int64, companyIDs []int64) (bool, error)
	NextPosition(ctx Ctx, ownerID int64) (int, error)
	CreateSchedule(ctx Ctx, s *Schedule) error
	UpdateSchedule(ctx Ctx, s *Schedule) error
	DeleteSchedule(ctx Ctx, id int64) error
	// Audience — кому адресовать сокет-события: владелец, адресаты личных шар и
	// участники компаний, которым расписание роздано.
	Audience(ctx Ctx, scheduleID int64) ([]int64, error)

	// ── Категории ──
	ListCategories(ctx Ctx, scheduleID int64) ([]*Category, error)
	// CategoriesBySchedules — категории пачкой для списка расписаний (без N+1).
	CategoriesBySchedules(ctx Ctx, ids []int64) (map[int64][]*Category, error)
	CreateCategory(ctx Ctx, c *Category) error
	// ReplaceCategories — заменить справочник целиком (импорт расписания).
	ReplaceCategories(ctx Ctx, scheduleID int64, cats []*Category) error
	UpdateCategory(ctx Ctx, c *Category) error
	DeleteCategory(ctx Ctx, scheduleID, id int64) error
	NextCategoryPosition(ctx Ctx, scheduleID int64) (int, error)

	// ── Поля карточки занятия ──
	ListFields(ctx Ctx, scheduleID int64) ([]*Field, error)
	FieldsBySchedules(ctx Ctx, ids []int64) (map[int64][]*Field, error)
	// ReplaceFields — полная замена набора полей; возвращает id удалённых, их
	// значения вычищаются из занятий вызывающим.
	ReplaceFields(ctx Ctx, scheduleID int64, fields []*Field) (removed []int64, err error)

	// ── Занятия ──
	// ListItems — все занятия расписания: их десятки, и клиент фильтрует по
	// неделе сам (правило повтора у него зеркальное).
	ListItems(ctx Ctx, scheduleID int64) ([]*Item, error)
	GetItem(ctx Ctx, scheduleID, id int64) (*Item, error)
	CreateItem(ctx Ctx, i *Item, searchText string) error
	UpdateItem(ctx Ctx, i *Item, searchText string) error
	DeleteItem(ctx Ctx, scheduleID, id int64) error
	// ReplaceItems — заменить все занятия расписания (импорт JSON).
	ReplaceItems(ctx Ctx, scheduleID int64, items []*Item, searchText func(*Item) string) error
	// NormalizeItemWeeks — подрезать номера недель под новую длину цикла;
	// возвращает, сколько занятий тронуто. Опустевший набор означает
	// «каждую неделю» — занятие не должно исчезнуть со шкалы.
	NormalizeItemWeeks(ctx Ctx, scheduleID int64, cycleWeeks int) (int, error)

	// ── Плитка и поиск ──
	// ItemsForDay — занятия всех доступных расписаний на конкретный день
	// недели: один запрос, отбор по неделе цикла делает сервис.
	ItemsForDay(ctx Ctx, userID int64, companyIDs []int64, weekday int) ([]*DayItem, error)
	// SearchItems — глобальный поиск по занятиям всех доступных расписаний.
	SearchItems(ctx Ctx, userID int64, companyIDs []int64, query string, limit int) ([]*ItemScope, error)

	// ── Публичные ссылки ──
	CreateShare(ctx Ctx, s *Share) error
	ListShares(ctx Ctx, scheduleID int64) ([]*Share, error)
	GetShareByCode(ctx Ctx, code string) (*Share, error)
	DeleteShare(ctx Ctx, id, scheduleID int64) error

	// ── Адресный доступ ──
	ListUserShares(ctx Ctx, scheduleID int64) ([]*UserShare, error)
	PutUserShare(ctx Ctx, s *UserShare) error
	DeleteUserShare(ctx Ctx, scheduleID int64, userID, companyID *int64) error
}

// DayItem — занятие вместе с его расписанием и категорией: живая плитка и
// поиск работают сразу по всем доступным расписаниям, и джойн делает БД.
type DayItem struct {
	Item     *Item
	Schedule *Schedule
	Color    string
}

// UserReader — read-only идентичность (владелец таблиц — authsvc).
type UserReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
	// CompaniesOf — компании пользователя: через них приходит доступ к
	// расписаниям, розданным компании.
	CompaniesOf(ctx Ctx, userID int64) ([]int64, error)
	// Colleagues — коллеги по компаниям (список «с кем поделиться»).
	Colleagues(ctx Ctx, userID int64, query string, limit int) ([]*UserRef, error)
	// Companies — компании пользователя с названиями.
	Companies(ctx Ctx, userID int64) ([]*CompanyRef, error)
}

// EventBus — сокет-события клиентам через Redis gw2:schedule:events
// (доставляет gatewaysvc).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}
