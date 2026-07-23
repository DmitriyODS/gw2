package domain

import "time"

// Срезы статистики — формы stats_repo во Flask (числа уже округлены до
// 2 знаков там, где Flask делал round(x, 2)).

type CommonMetrics struct {
	Debt      int
	Received  int
	Closed    int
	Remaining int
}

type TaskHours struct {
	TaskID     int64
	Name       string
	TotalHours float64
}

type EmployeeHours struct {
	UserID     int64
	FIO        string
	TasksCount int
	TotalHours float64
}

type UnitTypeStats struct {
	TypeID     int64
	Name       string
	TotalHours float64
	TasksCount int
}

type DeptStats struct {
	DeptID     int64
	Name       string
	TasksCount int
}

type UserUnitTypeStats struct {
	UserID    int64
	FIO       string
	UnitTypes []UnitTypeHours
}

type UnitTypeHours struct {
	TypeID     int64
	Name       string
	Hours      float64
	TasksCount int
}

type CalendarDay struct {
	Date       string
	Received   int
	Closed     int
	TotalHours float64
}

type UserTaskHours struct {
	TaskID     int64
	TaskName   string
	TotalHours float64
}

type ProfileStats struct {
	TotalHours  float64
	TasksCount  int
	ByUnitTypes []UnitTypeHours
}

type Responsible struct {
	UserID      int64
	FIO         string
	AvatarPath  *string
	Post        *string
	OpenCount   int
	ClosedCount int
}

// EmployeeRef — элемент списка /api/stats/employees ({id, fio}).
type EmployeeRef struct {
	ID  int64
	FIO string
}

// ── Активность сотрудника (раздел «Активность», руководитель компании) ──

// EmployeeActivitySummary — сводные метрики сотрудника за период.
type EmployeeActivitySummary struct {
	WorkedHours       float64 // суммарно отработано (сумма длительностей юнитов)
	TasksCreated      int     // создано задач (author_id)
	TasksClosed       int     // закрыто задач (ответственный, is_archived в период)
	Comments          int     // оставлено комментариев
	ActiveDays        int     // дней с активностью (уникальные даты юнитов)
	UnitsCount        int     // число юнитов
	AvgHoursPerClosed float64 // средние часы на закрытую задачу
	AvgCycleHours     float64 // среднее время жизни закрытой задачи (создана→закрыта)
}

// WeekdayHours — часы по дню недели (Weekday 0..6, 0 — воскресенье, как pg dow).
type WeekdayHours struct {
	Weekday int
	Hours   float64
}

// HourHours — часы по часу суток (0..23) старта юнитов (когда человек работает).
type HourHours struct {
	Hour  int
	Hours float64
}

// WeekPoint — точка недельного тренда (ISO-неделя): часы и закрытые задачи.
type WeekPoint struct {
	Week   string
	Hours  float64
	Closed int
}

// ActivityEvent — событие ленты активности сотрудника (что и когда делал).
type ActivityEvent struct {
	Type     string // unit_started|unit_stopped|task_created|task_closed|comment
	At       time.Time
	TaskID   *int64
	TaskName string
	Detail   string // тип юнита / длительность / фрагмент комментария
}

// StatsRepository — выборки статистики (порт stats_repo). Статистика всегда в
// пределах активной компании; companyID приходит из токена и непустой.
type StatsRepository interface {
	CommonMetrics(ctx Ctx, start, end time.Time, companyID *int64) (*CommonMetrics, error)
	TasksByHours(ctx Ctx, start, end time.Time, companyID *int64) ([]TaskHours, error)
	TasksByEmployees(ctx Ctx, start, end time.Time, companyID *int64) ([]EmployeeHours, error)
	ByUnitTypes(ctx Ctx, start, end time.Time, companyID *int64) ([]UnitTypeStats, error)
	ByDepartments(ctx Ctx, start, end time.Time, companyID *int64) ([]DeptStats, error)
	ByUnitTypesPerUser(ctx Ctx, start, end time.Time, companyID *int64) ([]UserUnitTypeStats, error)
	Calendar(ctx Ctx, start, end time.Time, companyID *int64) ([]CalendarDay, error)
	UserTasksDetail(ctx Ctx, userID int64, start, end time.Time) ([]UserTaskHours, error)
	ProfileStats(ctx Ctx, userID int64, start, end time.Time) (*ProfileStats, error)
	Responsibles(ctx Ctx, companyID *int64) ([]Responsible, error)
	// VisibleEmployees — список для селектора статистики (user_repo.get_all).
	VisibleEmployees(ctx Ctx, companyID *int64) ([]EmployeeRef, error)

	// ── Активность сотрудника (companyID непустой — активная компания) ──
	EmployeeSummary(ctx Ctx, companyID, userID int64, start, end time.Time) (*EmployeeActivitySummary, error)
	EmployeeByUnitTypes(ctx Ctx, companyID, userID int64, start, end time.Time) ([]UnitTypeHours, error)
	EmployeeByWeekday(ctx Ctx, companyID, userID int64, start, end time.Time) ([]WeekdayHours, error)
	EmployeeByHour(ctx Ctx, companyID, userID int64, start, end time.Time) ([]HourHours, error)
	EmployeeWeeklyTrend(ctx Ctx, companyID, userID int64, start, end time.Time) ([]WeekPoint, error)
	EmployeeFeed(ctx Ctx, companyID, userID int64, start, end time.Time, limit, offset int) ([]ActivityEvent, int, error)
}
