package postgres

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

// stats_activity.go — выборки раздела «Активность» сотрудника (руководитель
// компании смотрит работу подчинённого). Всё скоупится активной компанией
// (companyID непустой). Часы — та же формула hoursExpr, что в общей статистике.

func (r *Repo) EmployeeSummary(ctx context.Context, companyID, userID int64, start, end time.Time) (*domain.EmployeeActivitySummary, error) {
	s := &domain.EmployeeActivitySummary{}
	// Отработанные часы, число юнитов, активные дни.
	if err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(`+hoursExpr+`), 0), COUNT(*),
		       COUNT(DISTINCT (u.datetime_start AT TIME ZONE 'UTC')::date)
		  FROM units u
		 WHERE u.user_id = $1 AND u.company_id = $2
		   AND u.datetime_start >= $3 AND u.datetime_start <= $4`,
		userID, companyID, start, end).Scan(&s.WorkedHours, &s.UnitsCount, &s.ActiveDays); err != nil {
		return nil, err
	}
	// Создано задач.
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM tasks t
		 WHERE t.author_id = $1 AND t.company_id = $2
		   AND t.created_at >= $3 AND t.created_at <= $4`,
		userID, companyID, start, end).Scan(&s.TasksCreated); err != nil {
		return nil, err
	}
	// Закрыто задач (ответственный) + среднее время жизни закрытой задачи.
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*),
		       COALESCE(AVG(EXTRACT(EPOCH FROM t.archived_at - t.created_at) / 3600), 0)
		  FROM tasks t
		 WHERE t.responsible_user_id = $1 AND t.company_id = $2 AND t.is_archived
		   AND t.archived_at >= $3 AND t.archived_at <= $4`,
		userID, companyID, start, end).Scan(&s.TasksClosed, &s.AvgCycleHours); err != nil {
		return nil, err
	}
	// Комментарии.
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM comments c JOIN tasks t ON t.id = c.task_id
		 WHERE c.author_id = $1 AND t.company_id = $2
		   AND c.created_at >= $3 AND c.created_at <= $4`,
		userID, companyID, start, end).Scan(&s.Comments); err != nil {
		return nil, err
	}
	s.WorkedHours = round2(s.WorkedHours)
	s.AvgCycleHours = round2(s.AvgCycleHours)
	if s.TasksClosed > 0 {
		s.AvgHoursPerClosed = round2(s.WorkedHours / float64(s.TasksClosed))
	}
	return s, nil
}

func (r *Repo) EmployeeByUnitTypes(ctx context.Context, companyID, userID int64, start, end time.Time) ([]domain.UnitTypeHours, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ut.id, ut.name, COALESCE(SUM(`+hoursExpr+`), 0), COUNT(DISTINCT u.task_id)
		  FROM unit_types ut
		  JOIN units u ON u.unit_type_id = ut.id
		 WHERE u.user_id = $1 AND u.company_id = $2
		   AND u.datetime_start >= $3 AND u.datetime_start <= $4
		 GROUP BY ut.id, ut.name
		 ORDER BY 3 DESC`, userID, companyID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.UnitTypeHours{}
	for rows.Next() {
		var ut domain.UnitTypeHours
		if err := rows.Scan(&ut.TypeID, &ut.Name, &ut.Hours, &ut.TasksCount); err != nil {
			return nil, err
		}
		ut.Hours = round2(ut.Hours)
		out = append(out, ut)
	}
	return out, rows.Err()
}

func (r *Repo) EmployeeByWeekday(ctx context.Context, companyID, userID int64, start, end time.Time) ([]domain.WeekdayHours, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT EXTRACT(DOW FROM u.datetime_start)::int, COALESCE(SUM(`+hoursExpr+`), 0)
		  FROM units u
		 WHERE u.user_id = $1 AND u.company_id = $2
		   AND u.datetime_start >= $3 AND u.datetime_start <= $4
		 GROUP BY 1 ORDER BY 1`, userID, companyID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.WeekdayHours{}
	for rows.Next() {
		var w domain.WeekdayHours
		if err := rows.Scan(&w.Weekday, &w.Hours); err != nil {
			return nil, err
		}
		w.Hours = round2(w.Hours)
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *Repo) EmployeeByHour(ctx context.Context, companyID, userID int64, start, end time.Time) ([]domain.HourHours, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT EXTRACT(HOUR FROM u.datetime_start)::int, COALESCE(SUM(`+hoursExpr+`), 0)
		  FROM units u
		 WHERE u.user_id = $1 AND u.company_id = $2
		   AND u.datetime_start >= $3 AND u.datetime_start <= $4
		 GROUP BY 1 ORDER BY 1`, userID, companyID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.HourHours{}
	for rows.Next() {
		var h domain.HourHours
		if err := rows.Scan(&h.Hour, &h.Hours); err != nil {
			return nil, err
		}
		h.Hours = round2(h.Hours)
		out = append(out, h)
	}
	return out, rows.Err()
}

// EmployeeWeeklyTrend — часы (по юнитам) и закрытые задачи по ISO-неделям;
// объединяем два разреза по ключу недели.
func (r *Repo) EmployeeWeeklyTrend(ctx context.Context, companyID, userID int64, start, end time.Time) ([]domain.WeekPoint, error) {
	byWeek := map[string]*domain.WeekPoint{}
	order := []string{}
	touch := func(week string) *domain.WeekPoint {
		if p := byWeek[week]; p != nil {
			return p
		}
		p := &domain.WeekPoint{Week: week}
		byWeek[week] = p
		order = append(order, week)
		return p
	}

	hoursRows, err := r.pool.Query(ctx, `
		SELECT to_char(u.datetime_start, 'IYYY-"W"IW'), COALESCE(SUM(`+hoursExpr+`), 0)
		  FROM units u
		 WHERE u.user_id = $1 AND u.company_id = $2
		   AND u.datetime_start >= $3 AND u.datetime_start <= $4
		 GROUP BY 1 ORDER BY 1`, userID, companyID, start, end)
	if err != nil {
		return nil, err
	}
	for hoursRows.Next() {
		var week string
		var hours float64
		if err := hoursRows.Scan(&week, &hours); err != nil {
			hoursRows.Close()
			return nil, err
		}
		touch(week).Hours = round2(hours)
	}
	hoursRows.Close()
	if err := hoursRows.Err(); err != nil {
		return nil, err
	}

	closedRows, err := r.pool.Query(ctx, `
		SELECT to_char(t.archived_at, 'IYYY-"W"IW'), COUNT(*)
		  FROM tasks t
		 WHERE t.responsible_user_id = $1 AND t.company_id = $2 AND t.is_archived
		   AND t.archived_at >= $3 AND t.archived_at <= $4
		 GROUP BY 1 ORDER BY 1`, userID, companyID, start, end)
	if err != nil {
		return nil, err
	}
	defer closedRows.Close()
	for closedRows.Next() {
		var week string
		var closed int
		if err := closedRows.Scan(&week, &closed); err != nil {
			return nil, err
		}
		touch(week).Closed = closed
	}
	if err := closedRows.Err(); err != nil {
		return nil, err
	}

	sortWeeks(order)
	out := make([]domain.WeekPoint, 0, len(order))
	for _, w := range order {
		out = append(out, *byWeek[w])
	}
	return out, nil
}

// sortWeeks — лексикографическая сортировка ключей вида "2026-W07" (совпадает с
// хронологической: год и двузначная неделя дополнены нулями).
func sortWeeks(weeks []string) {
	for i := 1; i < len(weeks); i++ {
		for j := i; j > 0 && weeks[j-1] > weeks[j]; j-- {
			weeks[j-1], weeks[j] = weeks[j], weeks[j-1]
		}
	}
}

// EmployeeFeed — хронологическая лента событий сотрудника (что и когда делал) с
// пагинацией. total — общее число событий за период (для постраничного вывода).
func (r *Repo) EmployeeFeed(ctx context.Context, companyID, userID int64, start, end time.Time, limit, offset int) ([]domain.ActivityEvent, int, error) {
	rows, err := r.pool.Query(ctx, `
		WITH ev AS (
			SELECT 'unit_started'::text AS type, u.datetime_start AS at, u.task_id, t.name AS task_name, ut.name AS detail
			  FROM units u JOIN tasks t ON t.id = u.task_id JOIN unit_types ut ON ut.id = u.unit_type_id
			 WHERE u.user_id = $1 AND u.company_id = $2 AND u.datetime_start >= $3 AND u.datetime_start <= $4
			UNION ALL
			SELECT 'unit_stopped', u.datetime_end, u.task_id, t.name, ut.name
			  FROM units u JOIN tasks t ON t.id = u.task_id JOIN unit_types ut ON ut.id = u.unit_type_id
			 WHERE u.user_id = $1 AND u.company_id = $2 AND u.datetime_end IS NOT NULL
			   AND u.datetime_end >= $3 AND u.datetime_end <= $4
			UNION ALL
			SELECT 'task_created', t.created_at, t.id, t.name, ''
			  FROM tasks t WHERE t.author_id = $1 AND t.company_id = $2
			   AND t.created_at >= $3 AND t.created_at <= $4
			UNION ALL
			SELECT 'task_closed', t.archived_at, t.id, t.name, ''
			  FROM tasks t WHERE t.responsible_user_id = $1 AND t.company_id = $2 AND t.is_archived
			   AND t.archived_at >= $3 AND t.archived_at <= $4
			UNION ALL
			SELECT 'comment', c.created_at, c.task_id, t.name, left(c.text, 160)
			  FROM comments c JOIN tasks t ON t.id = c.task_id
			 WHERE c.author_id = $1 AND t.company_id = $2 AND c.created_at >= $3 AND c.created_at <= $4
		)
		SELECT type, at, task_id, task_name, detail, count(*) OVER() AS total
		  FROM ev ORDER BY at DESC, task_id DESC LIMIT $5 OFFSET $6`,
		userID, companyID, start, end, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []domain.ActivityEvent{}
	total := 0
	for rows.Next() {
		var (
			e      domain.ActivityEvent
			taskID int64
		)
		if err := rows.Scan(&e.Type, &e.At, &taskID, &e.TaskName, &e.Detail, &total); err != nil {
			return nil, 0, err
		}
		e.TaskID = &taskID
		out = append(out, e)
	}
	return out, total, rows.Err()
}
