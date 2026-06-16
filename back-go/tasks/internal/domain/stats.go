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
}
