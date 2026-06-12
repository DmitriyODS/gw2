package service

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

// Инструменты Грувика для запроса статистики через function-calling.
// Все запросы жёстко скоупятся по company_id владельца — Грувик видит
// только данные своей компании. Любая внутренняя ошибка возвращается как
// {"error": "..."} — LLM сам адекватно реагирует на сбой.

const (
	listLimitDefault = 10
	listLimitMax     = 30
)

// toolSchemasJSON — OpenAI-совместимые tool-definition'ы (1-в-1 с прежним
// TOOL_SCHEMAS из groove_ai_tools.py).
const toolSchemasJSON = `[
  {"type": "function", "function": {
    "name": "get_stats_summary",
    "description": "Общие метрики компании за период: поступило задач, закрыто, ещё в работе (на конец периода), долг до периода, суммарные часы команды. Использовать для ответа на вопросы типа «сколько задач поступило/закрыто», «сколько часов отработали», «как у нас дела за неделю».",
    "parameters": {"type": "object", "properties": {
      "period": {"type": "string", "enum": ["today", "yesterday", "this_week", "last_week", "this_month", "last_month", "7d", "30d"], "description": "Период статистики. По умолчанию this_week."}
    }, "additionalProperties": false}}},
  {"type": "function", "function": {
    "name": "list_departments",
    "description": "Список всех отделов компании с количеством поступивших задач за указанный период. Полезно когда пользователь упоминает отдел по названию или спрашивает «какие отделы у нас есть».",
    "parameters": {"type": "object", "properties": {
      "period": {"type": "string", "enum": ["today", "yesterday", "this_week", "last_week", "this_month", "last_month", "7d", "30d"], "description": "Период для счётчиков. По умолчанию this_week."}
    }, "additionalProperties": false}}},
  {"type": "function", "function": {
    "name": "get_top_employees",
    "description": "Топ сотрудников компании по отработанному времени за период. Возвращает ФИО, кол-во разных задач и суммарные часы. Использовать для вопросов «кто больше всего работал», «лидеры недели».",
    "parameters": {"type": "object", "properties": {
      "period": {"type": "string", "enum": ["today", "yesterday", "this_week", "last_week", "this_month", "last_month", "7d", "30d"]},
      "limit": {"type": "integer", "minimum": 1, "maximum": 30, "description": "Сколько строк вернуть (1..30). По умолчанию 10."}
    }, "additionalProperties": false}}},
  {"type": "function", "function": {
    "name": "get_stats_by_unit_types",
    "description": "Распределение работы по типам юнитов (звонок, разработка, встреча и т. п.) за период: часы и количество задач.",
    "parameters": {"type": "object", "properties": {
      "period": {"type": "string", "enum": ["today", "yesterday", "this_week", "last_week", "this_month", "last_month", "7d", "30d"]}
    }, "additionalProperties": false}}},
  {"type": "function", "function": {
    "name": "get_stats_calendar",
    "description": "Динамика по дням за период: на каждый день — поступило, закрыто и часы. Используй для вопросов «как менялось», «в какой день было больше всего работы».",
    "parameters": {"type": "object", "properties": {
      "period": {"type": "string", "enum": ["today", "yesterday", "this_week", "last_week", "this_month", "last_month", "7d", "30d"]}
    }, "additionalProperties": false}}}
]`

type period struct {
	Start time.Time
	End   time.Time
	Label string
}

// resolvePeriod — дружелюбный код периода → окно start/end (UTC) + ярлык.
func resolvePeriod(code string) period {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		code = "this_week"
	}
	today := mskMidnight(todayMSK())
	weekStart := today.AddDate(0, 0, -pyWeekday(today))
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, domain.MSK)

	var p period
	switch code {
	case "today":
		p = period{today, today.AddDate(0, 0, 1), "сегодня"}
	case "yesterday":
		p = period{today.AddDate(0, 0, -1), today, "вчера"}
	case "this_week", "week":
		p = period{weekStart, weekStart.AddDate(0, 0, 7), "эта неделя"}
	case "last_week":
		p = period{weekStart.AddDate(0, 0, -7), weekStart, "прошлая неделя"}
	case "this_month", "month":
		p = period{monthStart, monthStart.AddDate(0, 1, 0), "этот месяц"}
	case "last_month":
		p = period{monthStart.AddDate(0, -1, 0), monthStart, "прошлый месяц"}
	case "7d", "7days", "last_7_days":
		p = period{today.AddDate(0, 0, -6), today.AddDate(0, 0, 1), "последние 7 дней"}
	case "30d", "30days", "last_30_days":
		p = period{today.AddDate(0, 0, -29), today.AddDate(0, 0, 1), "последние 30 дней"}
	default:
		p = period{weekStart, weekStart.AddDate(0, 0, 7), "эта неделя"}
	}
	p.Start, p.End = p.Start.UTC(), p.End.UTC()
	return p
}

func roundHours(v float64) float64 {
	return math.Round(v*10) / 10
}

func argPeriod(args map[string]any) period {
	code, _ := args["period"].(string)
	return resolvePeriod(code)
}

// dispatchTool — запустить инструмент по имени; сбой → {"error": "..."}.
func (s *Service) dispatchTool(ctx context.Context, name string,
	args map[string]any, companyID int64) any {

	if args == nil {
		args = map[string]any{}
	}
	var result any
	var err error
	switch name {
	case "get_stats_summary":
		result, err = s.toolSummary(ctx, args, companyID)
	case "list_departments":
		result, err = s.toolDepartments(ctx, args, companyID)
	case "get_top_employees":
		result, err = s.toolTopEmployees(ctx, args, companyID)
	case "get_stats_by_unit_types":
		result, err = s.toolByUnitTypes(ctx, args, companyID)
	case "get_stats_calendar":
		result, err = s.toolCalendar(ctx, args, companyID)
	default:
		return map[string]any{"error": "unknown_tool:" + name}
	}
	if err != nil {
		s.log.Warn("groove.ai.tool_failed", "tool", name,
			"company_id", companyID, "error", err)
		return map[string]any{"error": "tool_execution_failed"}
	}
	return result
}

func (s *Service) toolSummary(ctx context.Context, args map[string]any, companyID int64) (any, error) {
	p := argPeriod(args)
	common, err := s.work.CommonMetrics(ctx, companyID, p.Start, p.End)
	if err != nil {
		return nil, err
	}
	employees, err := s.work.TopEmployees(ctx, companyID, p.Start, p.End)
	if err != nil {
		return nil, err
	}
	totalHours := 0.0
	for _, e := range employees {
		totalHours += roundHours(e.TotalHours)
	}
	return map[string]any{
		"period":             p.Label,
		"received":           common.Received,
		"closed":             common.Closed,
		"in_progress_now":    common.Remaining,
		"debt_before_period": common.Debt,
		"team_hours":         roundHours(totalHours),
	}, nil
}

func (s *Service) toolDepartments(ctx context.Context, args map[string]any, companyID int64) (any, error) {
	p := argPeriod(args)
	rows, err := s.work.ByDepartments(ctx, companyID, p.Start, p.End)
	if err != nil {
		return nil, err
	}
	departments := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		departments = append(departments, map[string]any{
			"id": r.ID, "name": r.Name, "received_count": r.TasksCount,
		})
	}
	return map[string]any{"period": p.Label, "departments": departments}, nil
}

func (s *Service) toolTopEmployees(ctx context.Context, args map[string]any, companyID int64) (any, error) {
	p := argPeriod(args)
	limit := listLimitDefault
	if raw, ok := args["limit"].(float64); ok && raw > 0 {
		limit = max(1, min(listLimitMax, int(raw)))
	}
	rows, err := s.work.TopEmployees(ctx, companyID, p.Start, p.End)
	if err != nil {
		return nil, err
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	employees := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		employees = append(employees, map[string]any{
			"fio": r.FIO, "tasks_count": r.TasksCount, "hours": roundHours(r.TotalHours),
		})
	}
	return map[string]any{"period": p.Label, "employees": employees}, nil
}

func (s *Service) toolByUnitTypes(ctx context.Context, args map[string]any, companyID int64) (any, error) {
	p := argPeriod(args)
	rows, err := s.work.ByUnitTypes(ctx, companyID, p.Start, p.End)
	if err != nil {
		return nil, err
	}
	if len(rows) > listLimitMax {
		rows = rows[:listLimitMax]
	}
	unitTypes := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		unitTypes = append(unitTypes, map[string]any{
			"name": r.Name, "hours": roundHours(r.TotalHours), "tasks_count": r.TasksCount,
		})
	}
	return map[string]any{"period": p.Label, "unit_types": unitTypes}, nil
}

func (s *Service) toolCalendar(ctx context.Context, args map[string]any, companyID int64) (any, error) {
	p := argPeriod(args)
	rows, err := s.work.Calendar(ctx, companyID, p.Start, p.End)
	if err != nil {
		return nil, err
	}
	days := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		days = append(days, map[string]any{
			"date": r.Date, "received": r.Received, "closed": r.Closed,
			"hours": roundHours(r.TotalHours),
		})
	}
	return map[string]any{"period": p.Label, "days": days}, nil
}
