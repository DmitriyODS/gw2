package domain

import (
	"context"
	"time"
)

// Ctx — алиас, чтобы сигнатуры портов не разбухали.
type Ctx = context.Context

// TaskRepository — персистентность задач + избранное + личные цвета.
type TaskRepository interface {
	GetTask(ctx Ctx, id int64) (*Task, error)
	// ListTasks — фильтры/сортировки/пагинация как task_repo.get_list.
	ListTasks(ctx Ctx, f TaskListFilter) (items []*Task, total int, err error)
	CreateTask(ctx Ctx, t *Task) error
	// UpdateTaskFields — точечное обновление колонок задачи.
	UpdateTaskFields(ctx Ctx, id int64, fields map[string]any) error
	DeleteTask(ctx Ctx, id int64) error

	HasActiveUnit(ctx Ctx, taskID int64) (bool, error)
	HasAnyUnits(ctx Ctx, taskID int64) (bool, error)
	IsFavorite(ctx Ctx, taskID, userID int64) (bool, error)
	// ToggleFavorite — новое состояние (true = добавлено).
	ToggleFavorite(ctx Ctx, taskID, userID int64) (bool, error)
	ActiveUsers(ctx Ctx, taskID int64) ([]UserRef, error)
	Contributors(ctx Ctx, taskID int64) ([]UserRef, error)
	UserColor(ctx Ctx, taskID, userID int64) (*string, error)
	// SetUserColor — nil-цвет удаляет запись.
	SetUserColor(ctx Ctx, taskID, userID int64, color *string) error
	// Enrichment — батч-обогащение списка (active_users, цвета, избранное,
	// has_units) без N+1.
	Enrichment(ctx Ctx, taskIDs []int64, userID int64) (*TaskEnrichment, error)
}

// UnitRepository — персистентность юнитов.
type UnitRepository interface {
	GetUnit(ctx Ctx, id int64) (*Unit, error)
	// UnitsByTask — юниты задачи с user/unit_type, по datetime_start DESC.
	UnitsByTask(ctx Ctx, taskID int64) ([]*Unit, error)
	ActiveUnitForUser(ctx Ctx, userID int64) (*Unit, error)
	CreateUnit(ctx Ctx, u *Unit) error
	// UpdateUnitFields — точечное обновление; всегда ставит is_edited=TRUE.
	UpdateUnitFields(ctx Ctx, id int64, fields map[string]any) error
	// StopUnit — выставить datetime_end=now(); возвращает время остановки.
	StopUnit(ctx Ctx, id int64) (time.Time, error)
	DeleteUnit(ctx Ctx, id int64) error
}

// UnitTypeRepository — типы юнитов компании.
type UnitTypeRepository interface {
	ListUnitTypes(ctx Ctx, companyID int64) ([]*UnitType, error)
	GetUnitType(ctx Ctx, id int64) (*UnitType, error)
	GetUnitTypeByName(ctx Ctx, name string, companyID int64) (*UnitType, error)
	CreateUnitType(ctx Ctx, ut *UnitType) error
	UpdateUnitTypeName(ctx Ctx, id int64, name string) error
	// DeleteUnitType — каскадно удаляет все юниты с этим типом (FK CASCADE).
	DeleteUnitType(ctx Ctx, id int64) error
}

// DepartmentRepository — отделы компании.
type DepartmentRepository interface {
	ListDepartments(ctx Ctx, companyID int64) ([]*Department, error)
	GetDepartment(ctx Ctx, id int64) (*Department, error)
	GetDepartmentByName(ctx Ctx, name string, companyID int64) (*Department, error)
	CreateDepartment(ctx Ctx, d *Department) error
	UpdateDepartmentName(ctx Ctx, id int64, name string) error
	DeleteDepartment(ctx Ctx, id int64) error
}

// StageRepository — этапы (канбан) компании.
type StageRepository interface {
	ListStages(ctx Ctx, companyID int64) ([]*Stage, error)
	GetStage(ctx Ctx, id int64) (*Stage, error)
	GetStageByName(ctx Ctx, name string, companyID int64) (*Stage, error)
	NextStageOrder(ctx Ctx, companyID int64) (int, error)
	CreateStage(ctx Ctx, s *Stage) error
	UpdateStageFields(ctx Ctx, id int64, fields map[string]any) error
	DeleteStage(ctx Ctx, id int64) error
	// ReorderStages — порядок = позиция в ordered_ids (чужие id игнорируются).
	ReorderStages(ctx Ctx, companyID int64, orderedIDs []int64) error
}

// CommentRepository — комментарии задач (soft-delete).
type CommentRepository interface {
	GetComment(ctx Ctx, id int64) (*Comment, error)
	ListComments(ctx Ctx, taskID int64) ([]*Comment, error)
	CreateComment(ctx Ctx, c *Comment) error
	UpdateCommentText(ctx Ctx, id int64, text string, updatedAt time.Time) error
	SoftDeleteComment(ctx Ctx, id int64, deletedAt time.Time) error
}

// UserReader — read-only доступ к пользователям платформы (auth-мидлварь,
// валидация ответственного, ФИО для force-stop уведомления).
type UserReader interface {
	GetUser(ctx Ctx, id int64) (*User, error)
	// CompanyActive — активна ли компания (для auth-гейта по АКТИВНОЙ компании
	// сессии из токена). nil (Администратор системы) → true.
	CompanyActive(ctx Ctx, companyID *int64) (bool, error)
}

// CompanyReader — read-only настройки компании (флаг uses_yougile для
// _yougile_enabled: показывать ли YouGile-ссылки в дампах задач).
type CompanyReader interface {
	// YougileEnabled — как _yougile_enabled во Flask: компании нет или
	// settings пусты → true; иначе settings.uses_yougile (дефолт true).
	YougileEnabled(ctx Ctx, companyID int64) (bool, error)
}

// GrooveHooks — gRPC-хуки геймификации (fire-and-forget ПОСЛЕ коммита;
// ошибки только в лог — геймификация не роняет основной флоу).
type GrooveHooks interface {
	OnUnitStarted(u *Unit, taskName string)
	OnUnitStopped(u *Unit, taskName string)
	OnTaskClosed(t *Task, actorID int64)
}

// AIClient — gRPC aisvc: семантический поиск задач и переиндексация.
// Fail-open: aisvc недоступен / AI выключен → обычный LIKE-поиск.
type AIClient interface {
	Enabled(ctx Ctx, companyID int64) bool
	// SemanticSearch — id задач по убыванию релевантности; ошибка → nil.
	SemanticSearch(ctx Ctx, companyID int64, query string) []int64
	// ScheduleReindex — fire-and-forget.
	ScheduleReindex(taskID int64)
}

// EventBus — сокет-события клиентам через Redis gw2:tasks:events
// (realtime-шлюз gatewaysvc доставляет их в WS-комнаты вербатим).
type EventBus interface {
	Publish(ctx Ctx, event string, rooms []string, payload any)
}
