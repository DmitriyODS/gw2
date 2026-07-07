// Package service — адаптеры статистики/поиска задач для gRPC TasksService
// (зовёт aisvc — инструменты ИИ-ассистента). В отличие от stats.go, где
// период задаётся произвольным from/to с REST, здесь период — человеко-
// читаемый код (today/this_week/…) — перенесено из бывшего
// groove/internal/service/tools.go (resolvePeriod), только теперь данные
// идут через существующие StatsRepository-методы, а не напрямую по SQL.
package service

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// assistantMSK — «дневные» периоды инструментов ассистента считаются по
// московскому времени (как раньше у Грувика).
var assistantMSK = time.FixedZone("MSK", 3*60*60)

const (
	assistantEmployeesDefault = 10
	assistantEmployeesMax     = 30
	assistantSearchDefault    = 10
	assistantSearchMax        = 30
)

// AssistantPeriod — окно [Start, End) в UTC + человекочитаемый ярлык.
type AssistantPeriod struct {
	Start time.Time
	End   time.Time
	Label string
}

// ResolveAssistantPeriod — код периода → окно дат. Поддержаны те же
// значения, что были у tool-схем Грувика: today/yesterday/this_week/
// last_week/this_month/last_month/7d/30d. Пустой/незнакомый код → this_week.
func ResolveAssistantPeriod(code string) AssistantPeriod {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		code = "this_week"
	}
	now := time.Now().In(assistantMSK)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, assistantMSK)
	weekStart := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, assistantMSK)

	var p AssistantPeriod
	switch code {
	case "today":
		p = AssistantPeriod{today, today.AddDate(0, 0, 1), "сегодня"}
	case "yesterday":
		p = AssistantPeriod{today.AddDate(0, 0, -1), today, "вчера"}
	case "this_week", "week":
		p = AssistantPeriod{weekStart, weekStart.AddDate(0, 0, 7), "эта неделя"}
	case "last_week":
		p = AssistantPeriod{weekStart.AddDate(0, 0, -7), weekStart, "прошлая неделя"}
	case "this_month", "month":
		p = AssistantPeriod{monthStart, monthStart.AddDate(0, 1, 0), "этот месяц"}
	case "last_month":
		p = AssistantPeriod{monthStart.AddDate(0, -1, 0), monthStart, "прошлый месяц"}
	case "7d", "7days", "last_7_days":
		p = AssistantPeriod{today.AddDate(0, 0, -6), today.AddDate(0, 0, 1), "последние 7 дней"}
	case "30d", "30days", "last_30_days":
		p = AssistantPeriod{today.AddDate(0, 0, -29), today.AddDate(0, 0, 1), "последние 30 дней"}
	default:
		p = AssistantPeriod{weekStart, weekStart.AddDate(0, 0, 7), "эта неделя"}
	}
	p.Start, p.End = p.Start.UTC(), p.End.UTC()
	return p
}

func roundHours(v float64) float64 { return math.Round(v*10) / 10 }

func clampLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

// AssistantStatsSummary — общие метрики компании за период (инструмент
// get_stats_summary ИИ-ассистента): поступило/закрыто/в работе/долг + часы
// команды (сумма часов по сотрудникам, как считал старый toolSummary).
func (s *Service) AssistantStatsSummary(ctx context.Context, companyID int64, periodCode string) (*dto.AssistantSummary, error) {
	p := ResolveAssistantPeriod(periodCode)
	metrics, err := s.stats.CommonMetrics(ctx, p.Start, p.End, &companyID)
	if err != nil {
		return nil, err
	}
	employees, err := s.stats.TasksByEmployees(ctx, p.Start, p.End, &companyID)
	if err != nil {
		return nil, err
	}
	var totalHours float64
	for _, e := range employees {
		totalHours += e.TotalHours
	}
	return &dto.AssistantSummary{
		PeriodLabel:     p.Label,
		NewCount:        metrics.Received,
		ClosedCount:     metrics.Closed,
		InProgressCount: metrics.Remaining,
		DebtCount:       metrics.Debt,
		TotalHours:      roundHours(totalHours),
	}, nil
}

// AssistantDepartments — отделы компании с количеством поступивших задач за
// период (list_departments).
func (s *Service) AssistantDepartments(ctx context.Context, companyID int64, periodCode string) ([]dto.DeptStats, string, error) {
	p := ResolveAssistantPeriod(periodCode)
	rows, err := s.stats.ByDepartments(ctx, p.Start, p.End, &companyID)
	if err != nil {
		return nil, "", err
	}
	return dto.NewDeptStats(rows), p.Label, nil
}

// AssistantTopEmployees — топ сотрудников по отработанным часам за период
// (get_top_employees); TasksByEmployees уже отсортирован по часам DESC.
func (s *Service) AssistantTopEmployees(ctx context.Context, companyID int64, periodCode string, limit int) ([]dto.TaskByEmployee, string, error) {
	limit = clampLimit(limit, assistantEmployeesDefault, assistantEmployeesMax)
	p := ResolveAssistantPeriod(periodCode)
	rows, err := s.stats.TasksByEmployees(ctx, p.Start, p.End, &companyID)
	if err != nil {
		return nil, "", err
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return dto.NewTaskByEmployees(rows), p.Label, nil
}

// AssistantByUnitTypes — разбивка работы по типам юнитов за период
// (get_stats_by_unit_types).
func (s *Service) AssistantByUnitTypes(ctx context.Context, companyID int64, periodCode string) ([]dto.UnitTypeStats, string, error) {
	p := ResolveAssistantPeriod(periodCode)
	rows, err := s.stats.ByUnitTypes(ctx, p.Start, p.End, &companyID)
	if err != nil {
		return nil, "", err
	}
	return dto.NewUnitTypeStats(rows), p.Label, nil
}

// AssistantCalendar — динамика по дням за период (get_stats_calendar).
func (s *Service) AssistantCalendar(ctx context.Context, companyID int64, periodCode string) ([]dto.CalendarDay, string, error) {
	p := ResolveAssistantPeriod(periodCode)
	rows, err := s.stats.Calendar(ctx, p.Start, p.End, &companyID)
	if err != nil {
		return nil, "", err
	}
	return dto.NewCalendar(rows), p.Label, nil
}

// AssistantSearchTasks — поиск задач компании для ИИ-ассистента (SearchTasks
// gRPC). Переиспользует тот же путь, что REST-список задач: включённый AI —
// семантика через aisvc, иначе LIKE по названию (см. ListTasks). Только
// активные (неархивные) задачи — ассистенту нет смысла находить архив.
func (s *Service) AssistantSearchTasks(ctx context.Context, companyID int64, query string, limit int) ([]dto.Task, error) {
	limit = clampLimit(limit, assistantSearchDefault, assistantSearchMax)
	list, err := s.ListTasks(ctx, domain.TaskListFilter{
		CompanyID: &companyID,
		Tab:       "active",
		Search:    query,
		Page:      1,
		PerPage:   limit,
	})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// AssistantTaskLink — карточка задачи для формирования ссылки в ответе
// ассистента (GetTaskLink gRPC); nil без ошибки — задачи нет либо она не в
// этой компании (вызывающий трактует как NOT_FOUND).
func (s *Service) AssistantTaskLink(ctx context.Context, companyID, taskID int64) (*domain.Task, error) {
	task, err := s.taskInCompany(ctx, taskID, &companyID)
	if err != nil {
		if de := domain.AsDomainError(err); de != nil && de.Code == "NOT_FOUND" {
			return nil, nil
		}
		return nil, err
	}
	return task, nil
}
