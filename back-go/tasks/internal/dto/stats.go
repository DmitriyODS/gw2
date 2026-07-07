package dto

import "github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"

// Формы /api/stats/* — байт-в-байт со схемами schemas/stats.py
// (порядок ключей алфавитный, как jsonify).

type Period struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type TaskMetrics struct {
	Closed    int `json:"closed"`
	Debt      int `json:"debt"`
	Received  int `json:"received"`
	Remaining int `json:"remaining"`
}

type TaskByHours struct {
	Name       string  `json:"name"`
	TaskID     int64   `json:"task_id"`
	TotalHours float64 `json:"total_hours"`
}

type TaskByEmployee struct {
	FIO        string  `json:"fio"`
	TasksCount int     `json:"tasks_count"`
	TotalHours float64 `json:"total_hours"`
	UserID     int64   `json:"user_id"`
}

type StatsCommon struct {
	Period           Period           `json:"period"`
	Tasks            TaskMetrics      `json:"tasks"`
	TasksByEmployees []TaskByEmployee `json:"tasks_by_employees"`
	TasksByHours     []TaskByHours    `json:"tasks_by_hours"`
}

type UnitTypeStats struct {
	Name       string  `json:"name"`
	TasksCount int     `json:"tasks_count"`
	TotalHours float64 `json:"total_hours"`
	TypeID     int64   `json:"type_id"`
}

type DeptStats struct {
	DeptID     int64  `json:"dept_id"`
	Name       string `json:"name"`
	TasksCount int    `json:"tasks_count"`
}

type UnitTypePerUser struct {
	Hours      float64 `json:"hours"`
	Name       string  `json:"name"`
	TasksCount int     `json:"tasks_count"`
	TypeID     int64   `json:"type_id"`
}

type UserUnitTypeStats struct {
	FIO       string            `json:"fio"`
	UnitTypes []UnitTypePerUser `json:"unit_types"`
	UserID    int64             `json:"user_id"`
}

type CalendarDay struct {
	Closed     int     `json:"closed"`
	Date       string  `json:"date"`
	Received   int     `json:"received"`
	TotalHours float64 `json:"total_hours"`
}

type StatsExtended struct {
	ByDepartments      []DeptStats         `json:"by_departments"`
	ByUnitTypes        []UnitTypeStats     `json:"by_unit_types"`
	ByUnitTypesPerUser []UserUnitTypeStats `json:"by_unit_types_per_user"`
	Calendar           []CalendarDay       `json:"calendar"`
}

type UserTaskHours struct {
	TaskID     int64   `json:"task_id"`
	TaskName   string  `json:"task_name"`
	TotalHours float64 `json:"total_hours"`
}

type StatsUserTasks struct {
	Tasks      []UserTaskHours `json:"tasks"`
	TasksCount int             `json:"tasks_count"`
}

type StatsProfile struct {
	ByUnitTypes []UnitTypePerUser `json:"by_unit_types"`
	Period      Period            `json:"period"`
	TasksCount  int               `json:"tasks_count"`
	TotalHours  float64           `json:"total_hours"`
}

type ResponsibleDTO struct {
	AvatarPath  *string `json:"avatar_path"`
	ClosedCount int     `json:"closed_count"`
	FIO         string  `json:"fio"`
	OpenCount   int     `json:"open_count"`
	Post        *string `json:"post"`
	UserID      int64   `json:"user_id"`
}

type EmployeeRef struct {
	FIO string `json:"fio"`
	ID  int64  `json:"id"`
}

// AssistantSummary — ответ инструмента get_stats_summary ИИ-ассистента
// (gRPC GetStatsSummary): период задаётся человекочитаемым кодом, не парой
// from/to, поэтому это отдельная от StatsCommon форма.
type AssistantSummary struct {
	PeriodLabel     string
	NewCount        int
	ClosedCount     int
	InProgressCount int
	DebtCount       int
	TotalHours      float64
}

func NewTaskByHours(items []domain.TaskHours) []TaskByHours {
	out := make([]TaskByHours, 0, len(items))
	for _, r := range items {
		out = append(out, TaskByHours{Name: r.Name, TaskID: r.TaskID, TotalHours: r.TotalHours})
	}
	return out
}

func NewTaskByEmployees(items []domain.EmployeeHours) []TaskByEmployee {
	out := make([]TaskByEmployee, 0, len(items))
	for _, r := range items {
		out = append(out, TaskByEmployee{FIO: r.FIO, TasksCount: r.TasksCount,
			TotalHours: r.TotalHours, UserID: r.UserID})
	}
	return out
}

func NewUnitTypeStats(items []domain.UnitTypeStats) []UnitTypeStats {
	out := make([]UnitTypeStats, 0, len(items))
	for _, r := range items {
		out = append(out, UnitTypeStats{Name: r.Name, TasksCount: r.TasksCount,
			TotalHours: r.TotalHours, TypeID: r.TypeID})
	}
	return out
}

func NewDeptStats(items []domain.DeptStats) []DeptStats {
	out := make([]DeptStats, 0, len(items))
	for _, r := range items {
		out = append(out, DeptStats{DeptID: r.DeptID, Name: r.Name, TasksCount: r.TasksCount})
	}
	return out
}

func NewUnitTypesPerUser(items []domain.UserUnitTypeStats) []UserUnitTypeStats {
	out := make([]UserUnitTypeStats, 0, len(items))
	for _, r := range items {
		types := make([]UnitTypePerUser, 0, len(r.UnitTypes))
		for _, ut := range r.UnitTypes {
			types = append(types, UnitTypePerUser{Hours: ut.Hours, Name: ut.Name,
				TasksCount: ut.TasksCount, TypeID: ut.TypeID})
		}
		out = append(out, UserUnitTypeStats{FIO: r.FIO, UnitTypes: types, UserID: r.UserID})
	}
	return out
}

func NewCalendar(items []domain.CalendarDay) []CalendarDay {
	out := make([]CalendarDay, 0, len(items))
	for _, r := range items {
		out = append(out, CalendarDay{Closed: r.Closed, Date: r.Date,
			Received: r.Received, TotalHours: r.TotalHours})
	}
	return out
}

func NewResponsibles(items []domain.Responsible) []ResponsibleDTO {
	out := make([]ResponsibleDTO, 0, len(items))
	for _, r := range items {
		out = append(out, ResponsibleDTO{AvatarPath: r.AvatarPath, ClosedCount: r.ClosedCount,
			FIO: r.FIO, OpenCount: r.OpenCount, Post: r.Post, UserID: r.UserID})
	}
	return out
}

func NewEmployeeRefs(items []domain.EmployeeRef) []EmployeeRef {
	out := make([]EmployeeRef, 0, len(items))
	for _, r := range items {
		out = append(out, EmployeeRef{FIO: r.FIO, ID: r.ID})
	}
	return out
}
