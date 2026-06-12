package postgres

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

func noRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// AICompanyIDs — компании с включённым AI (как Company.query.filter_by(
// ai_enabled=True) во Flask: без фильтра по is_active).
func (r *Repo) AICompanyIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id FROM companies WHERE ai_enabled = TRUE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }

// TVWeekContext — срез stats_repo за окно для контекстного ТВ-факта:
// получено/закрыто задач, часы команды (сумма по сотрудникам, округлённым
// до 2 знаков, итог до 1 — как в _context_for_company), лидер недели по
// часам юнитов, самый активный отдел по числу поступивших задач.
func (r *Repo) TVWeekContext(ctx context.Context, companyID int64, start, end time.Time) (*domain.TVWeekContext, error) {
	out := &domain.TVWeekContext{}

	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE received_at >= $2 AND received_at <= $3),
		       COUNT(*) FILTER (WHERE is_archived = TRUE
		                          AND archived_at >= $2 AND archived_at <= $3)
		  FROM tasks
		 WHERE company_id = $1`, companyID, start, end).
		Scan(&out.ReceivedWeek, &out.ClosedWeek)
	if err != nil {
		return nil, err
	}

	// Часы по сотрудникам — как get_tasks_by_employees: фильтр по старту
	// юнита в окне, открытые юниты считаются до now().
	rows, err := r.pool.Query(ctx, `
		SELECT u.fio,
		       COALESCE(SUM(EXTRACT(EPOCH FROM COALESCE(un.datetime_end, now()) - un.datetime_start) / 3600), 0)
		  FROM users u
		  JOIN units un ON un.user_id = u.id
		 WHERE un.datetime_start >= $2 AND un.datetime_start <= $3
		   AND u.company_id = $1
		 GROUP BY u.id, u.fio
		 ORDER BY SUM(EXTRACT(EPOCH FROM COALESCE(un.datetime_end, now()) - un.datetime_start) / 3600) DESC`,
		companyID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	total := 0.0
	first := true
	for rows.Next() {
		var fio string
		var hours float64
		if err := rows.Scan(&fio, &hours); err != nil {
			return nil, err
		}
		hours = round2(hours)
		total += hours
		if first {
			out.LeaderFIO, out.LeaderHours = &fio, &hours
			first = false
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out.TeamHoursWeek = math.Round(total*10) / 10

	// Самый активный отдел — как get_by_departments (топ-1).
	var dept *string
	err = r.pool.QueryRow(ctx, `
		SELECT d.name
		  FROM departments d
		  JOIN tasks t ON t.department_id = d.id
		 WHERE t.received_at >= $2 AND t.received_at <= $3
		   AND d.company_id = $1
		 GROUP BY d.id, d.name
		 ORDER BY COUNT(DISTINCT t.id) DESC
		 LIMIT 1`, companyID, start, end).Scan(&dept)
	if err == nil {
		out.TopDept = dept
	} else if !noRows(err) {
		return nil, err
	}
	return out, nil
}
