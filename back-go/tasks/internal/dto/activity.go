package dto

import (
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

// Формы раздела «Активность» сотрудника (руководитель компании).

type ActivitySummary struct {
	WorkedHours       float64 `json:"worked_hours"`
	TasksCreated      int     `json:"tasks_created"`
	TasksClosed       int     `json:"tasks_closed"`
	Comments          int     `json:"comments"`
	ActiveDays        int     `json:"active_days"`
	UnitsCount        int     `json:"units_count"`
	AvgHoursPerClosed float64 `json:"avg_hours_per_closed"`
	AvgCycleHours     float64 `json:"avg_cycle_hours"`
}

type WeekdayHours struct {
	Weekday int     `json:"weekday"` // 0..6, 0 — воскресенье
	Hours   float64 `json:"hours"`
}

type HourHours struct {
	Hour  int     `json:"hour"`
	Hours float64 `json:"hours"`
}

type WeekPoint struct {
	Week   string  `json:"week"`
	Hours  float64 `json:"hours"`
	Closed int     `json:"closed"`
}

type EmployeeActivity struct {
	Period      Period            `json:"period"`
	Summary     ActivitySummary   `json:"summary"`
	ByUnitTypes []UnitTypePerUser `json:"by_unit_types"`
	ByWeekday   []WeekdayHours    `json:"by_weekday"`
	ByHour      []HourHours       `json:"by_hour"`
	WeeklyTrend []WeekPoint       `json:"weekly_trend"`
}

type ActivityEvent struct {
	Type     string    `json:"type"`
	At       time.Time `json:"at"`
	TaskID   *int64    `json:"task_id"`
	TaskName string    `json:"task_name"`
	Detail   string    `json:"detail"`
}

type ActivityFeed struct {
	Items   []ActivityEvent `json:"items"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	PerPage int             `json:"per_page"`
}

func NewActivitySummary(s *domain.EmployeeActivitySummary) ActivitySummary {
	return ActivitySummary{
		WorkedHours: s.WorkedHours, TasksCreated: s.TasksCreated, TasksClosed: s.TasksClosed,
		Comments: s.Comments, ActiveDays: s.ActiveDays, UnitsCount: s.UnitsCount,
		AvgHoursPerClosed: s.AvgHoursPerClosed, AvgCycleHours: s.AvgCycleHours,
	}
}

func NewActivityUnitTypes(items []domain.UnitTypeHours) []UnitTypePerUser {
	out := make([]UnitTypePerUser, 0, len(items))
	for _, u := range items {
		out = append(out, UnitTypePerUser{Hours: u.Hours, Name: u.Name, TasksCount: u.TasksCount, TypeID: u.TypeID})
	}
	return out
}

func NewWeekdayHours(items []domain.WeekdayHours) []WeekdayHours {
	out := make([]WeekdayHours, 0, len(items))
	for _, w := range items {
		out = append(out, WeekdayHours{Weekday: w.Weekday, Hours: w.Hours})
	}
	return out
}

func NewHourHours(items []domain.HourHours) []HourHours {
	out := make([]HourHours, 0, len(items))
	for _, h := range items {
		out = append(out, HourHours{Hour: h.Hour, Hours: h.Hours})
	}
	return out
}

func NewWeekPoints(items []domain.WeekPoint) []WeekPoint {
	out := make([]WeekPoint, 0, len(items))
	for _, p := range items {
		out = append(out, WeekPoint{Week: p.Week, Hours: p.Hours, Closed: p.Closed})
	}
	return out
}

func NewActivityEvents(items []domain.ActivityEvent) []ActivityEvent {
	out := make([]ActivityEvent, 0, len(items))
	for _, e := range items {
		out = append(out, ActivityEvent{Type: e.Type, At: e.At, TaskID: e.TaskID, TaskName: e.TaskName, Detail: e.Detail})
	}
	return out
}
