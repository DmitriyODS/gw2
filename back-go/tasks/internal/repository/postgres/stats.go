package postgres

import (
	"context"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

// Срезы статистики — порт stats_repo.py: те же выражения и округления
// (часы = EXTRACT(EPOCH FROM COALESCE(datetime_end, now()) - datetime_start)/3600,
// открытые юниты считаются до now(); round до 2 знаков на выходе).

const hoursExpr = `EXTRACT(EPOCH FROM COALESCE(u.datetime_end, now()) - u.datetime_start) / 3600`

// bizLocalStart — u.datetime_start в деловой таймзоне платформы (МСК). Все
// «дневные» разрезы (час дня, день недели, активные дни) обязаны бакетить по
// местному времени: сессия БД в UTC, и без конверсии пики смещаются на 3 часа,
// а работа около полуночи попадает не в тот день. Совпадает с МСК ассистента
// (assistant_stats.go) и питомцев.
const bizLocalStart = `(u.datetime_start AT TIME ZONE 'Europe/Moscow')`

func round2(x float64) float64 { return math.Round(x*100) / 100 }

// companyCond — опциональный фильтр "AND <col> = $N".
func companyCond(args *[]any, companyID *int64, col string) string {
	if companyID == nil {
		return ""
	}
	*args = append(*args, *companyID)
	return " AND " + col + " = $" + strconv.Itoa(len(*args))
}

// memberCond — фильтр принадлежности пользователя активной компании через
// user_companies (многокомпанийность: users.company_id — лишь первичная, поэтому
// скоуп людей нельзя вести по ней). userIDCol — колонка id пользователя.
func memberCond(args *[]any, companyID *int64, userIDCol string) string {
	if companyID == nil {
		return ""
	}
	*args = append(*args, *companyID)
	return " AND " + userIDCol + " IN (SELECT user_id FROM user_companies WHERE company_id = $" + strconv.Itoa(len(*args)) + ")"
}

func (r *Repo) CommonMetrics(ctx context.Context, start, end time.Time, companyID *int64) (*domain.CommonMetrics, error) {
	args := []any{start, end}
	cond := companyCond(&args, companyID, "company_id")
	var m domain.CommonMetrics
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(id) FILTER (WHERE is_archived = FALSE AND received_at < $1),
		       COUNT(id) FILTER (WHERE received_at >= $1 AND received_at <= $2),
		       COUNT(id) FILTER (WHERE is_archived = TRUE
		                           AND archived_at >= $1 AND archived_at <= $2),
		       COUNT(id) FILTER (WHERE is_archived = FALSE)
		  FROM tasks
		 WHERE TRUE`+cond, args...).
		Scan(&m.Debt, &m.Received, &m.Closed, &m.Remaining)
	return &m, err
}

func (r *Repo) TasksByHours(ctx context.Context, start, end time.Time, companyID *int64) ([]domain.TaskHours, error) {
	args := []any{start, end}
	cond := companyCond(&args, companyID, "t.company_id")
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.name, COALESCE(SUM(`+hoursExpr+`), 0)
		  FROM tasks t
		  JOIN units u ON u.task_id = t.id
		 WHERE u.datetime_start >= $1 AND u.datetime_start <= $2`+cond+`
		 GROUP BY t.id, t.name
		 ORDER BY SUM(`+hoursExpr+`) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TaskHours{}
	for rows.Next() {
		var t domain.TaskHours
		if err := rows.Scan(&t.TaskID, &t.Name, &t.TotalHours); err != nil {
			return nil, err
		}
		t.TotalHours = round2(t.TotalHours)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repo) TasksByEmployees(ctx context.Context, start, end time.Time, companyID *int64) ([]domain.EmployeeHours, error) {
	args := []any{start, end}
	// Скоуп по компании самого юнита (u.company_id), а не по членству автора:
	// один аккаунт работает в нескольких компаниях, но каждый юнит принадлежит
	// ровно одной — иначе часы из другой компании протекают в эту статистику.
	cond := companyCond(&args, companyID, "u.company_id")
	rows, err := r.pool.Query(ctx, `
		SELECT us.id, us.fio, COUNT(DISTINCT u.task_id), COALESCE(SUM(`+hoursExpr+`), 0)
		  FROM users us
		  JOIN units u ON u.user_id = us.id
		 WHERE u.datetime_start >= $1 AND u.datetime_start <= $2`+cond+`
		 GROUP BY us.id, us.fio
		 ORDER BY SUM(`+hoursExpr+`) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.EmployeeHours{}
	for rows.Next() {
		var e domain.EmployeeHours
		if err := rows.Scan(&e.UserID, &e.FIO, &e.TasksCount, &e.TotalHours); err != nil {
			return nil, err
		}
		e.TotalHours = round2(e.TotalHours)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repo) ByUnitTypes(ctx context.Context, start, end time.Time, companyID *int64) ([]domain.UnitTypeStats, error) {
	args := []any{start, end}
	cond := companyCond(&args, companyID, "ut.company_id")
	rows, err := r.pool.Query(ctx, `
		SELECT ut.id, ut.name, COALESCE(SUM(`+hoursExpr+`), 0), COUNT(DISTINCT u.task_id)
		  FROM unit_types ut
		  JOIN units u ON u.unit_type_id = ut.id
		 WHERE u.datetime_start >= $1 AND u.datetime_start <= $2`+cond+`
		 GROUP BY ut.id, ut.name
		 ORDER BY SUM(`+hoursExpr+`) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.UnitTypeStats{}
	for rows.Next() {
		var s domain.UnitTypeStats
		if err := rows.Scan(&s.TypeID, &s.Name, &s.TotalHours, &s.TasksCount); err != nil {
			return nil, err
		}
		s.TotalHours = round2(s.TotalHours)
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repo) ByDepartments(ctx context.Context, start, end time.Time, companyID *int64) ([]domain.DeptStats, error) {
	args := []any{start, end}
	cond := companyCond(&args, companyID, "d.company_id")
	rows, err := r.pool.Query(ctx, `
		SELECT d.id, d.name, COUNT(DISTINCT t.id)
		  FROM departments d
		  JOIN tasks t ON t.department_id = d.id
		 WHERE t.received_at >= $1 AND t.received_at <= $2`+cond+`
		 GROUP BY d.id, d.name
		 ORDER BY COUNT(DISTINCT t.id) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.DeptStats{}
	for rows.Next() {
		var s domain.DeptStats
		if err := rows.Scan(&s.DeptID, &s.Name, &s.TasksCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repo) ByUnitTypesPerUser(ctx context.Context, start, end time.Time, companyID *int64) ([]domain.UserUnitTypeStats, error) {
	args := []any{start, end}
	cond := companyCond(&args, companyID, "u.company_id")
	rows, err := r.pool.Query(ctx, `
		SELECT us.id, us.fio, ut.id, ut.name,
		       COALESCE(SUM(`+hoursExpr+`), 0), COUNT(DISTINCT u.task_id)
		  FROM users us
		  JOIN units u ON u.user_id = us.id
		  JOIN unit_types ut ON ut.id = u.unit_type_id
		 WHERE u.datetime_start >= $1 AND u.datetime_start <= $2`+cond+`
		 GROUP BY us.id, us.fio, ut.id, ut.name
		 ORDER BY us.fio, ut.name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Группировка по пользователю с сохранением порядка появления.
	byUser := map[int64]*domain.UserUnitTypeStats{}
	orderIDs := []int64{}
	for rows.Next() {
		var (
			userID int64
			fio    string
			ut     domain.UnitTypeHours
		)
		if err := rows.Scan(&userID, &fio, &ut.TypeID, &ut.Name, &ut.Hours, &ut.TasksCount); err != nil {
			return nil, err
		}
		ut.Hours = round2(ut.Hours)
		entry, ok := byUser[userID]
		if !ok {
			entry = &domain.UserUnitTypeStats{UserID: userID, FIO: fio}
			byUser[userID] = entry
			orderIDs = append(orderIDs, userID)
		}
		entry.UnitTypes = append(entry.UnitTypes, ut)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]domain.UserUnitTypeStats, 0, len(orderIDs))
	for _, id := range orderIDs {
		out = append(out, *byUser[id])
	}
	return out, nil
}

func (r *Repo) Calendar(ctx context.Context, start, end time.Time, companyID *int64) ([]domain.CalendarDay, error) {
	type dayAgg struct {
		received, closed int
		hours            float64
	}
	days := map[string]*dayAgg{}
	get := func(d string) *dayAgg {
		if days[d] == nil {
			days[d] = &dayAgg{}
		}
		return days[d]
	}

	args := []any{start, end}
	cond := companyCond(&args, companyID, "company_id")
	rows, err := r.pool.Query(ctx, `
		SELECT (received_at AT TIME ZONE 'Europe/Moscow')::date::text, COUNT(id) FROM tasks
		 WHERE received_at >= $1 AND received_at <= $2`+cond+`
		 GROUP BY 1`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d string
		var n int
		if err := rows.Scan(&d, &n); err != nil {
			rows.Close()
			return nil, err
		}
		get(d).received = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	args = []any{start, end}
	cond = companyCond(&args, companyID, "company_id")
	rows, err = r.pool.Query(ctx, `
		SELECT (archived_at AT TIME ZONE 'Europe/Moscow')::date::text, COUNT(id) FROM tasks
		 WHERE is_archived = TRUE AND archived_at >= $1 AND archived_at <= $2`+cond+`
		 GROUP BY 1`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d string
		var n int
		if err := rows.Scan(&d, &n); err != nil {
			rows.Close()
			return nil, err
		}
		get(d).closed = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	args = []any{start, end}
	hoursJoin := ""
	hoursCond := ""
	if companyID != nil {
		hoursJoin = " JOIN tasks t ON t.id = u.task_id"
		args = append(args, *companyID)
		hoursCond = " AND t.company_id = $3"
	}
	rows, err = r.pool.Query(ctx, `
		SELECT `+bizLocalStart+`::date::text, SUM(`+hoursExpr+`)
		  FROM units u`+hoursJoin+`
		 WHERE u.datetime_start >= $1 AND u.datetime_start <= $2`+hoursCond+`
		 GROUP BY 1`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d string
		var h *float64
		if err := rows.Scan(&d, &h); err != nil {
			rows.Close()
			return nil, err
		}
		if h != nil {
			get(d).hours = round2(*h)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	dates := make([]string, 0, len(days))
	for d := range days {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	out := make([]domain.CalendarDay, 0, len(dates))
	for _, d := range dates {
		agg := days[d]
		out = append(out, domain.CalendarDay{
			Date: d, Received: agg.received, Closed: agg.closed, TotalHours: agg.hours,
		})
	}
	return out, nil
}

func (r *Repo) UserTasksDetail(ctx context.Context, userID int64, companyID *int64, start, end time.Time) ([]domain.UserTaskHours, error) {
	// Скоуп по компании самого юнита (u.company_id), как в общей статистике —
	// иначе часы из других компаний пользователя протекают в этот срез и
	// расходятся с разделом «Статистика».
	args := []any{userID, start, end}
	cond := companyCond(&args, companyID, "u.company_id")
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.name, COALESCE(SUM(`+hoursExpr+`), 0)
		  FROM tasks t
		  JOIN units u ON u.task_id = t.id
		 WHERE u.user_id = $1 AND u.datetime_start >= $2 AND u.datetime_start <= $3`+cond+`
		 GROUP BY t.id, t.name
		 ORDER BY SUM(`+hoursExpr+`) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.UserTaskHours{}
	for rows.Next() {
		var t domain.UserTaskHours
		if err := rows.Scan(&t.TaskID, &t.TaskName, &t.TotalHours); err != nil {
			return nil, err
		}
		t.TotalHours = round2(t.TotalHours)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repo) ProfileStats(ctx context.Context, userID int64, companyID *int64, start, end time.Time) (*domain.ProfileStats, error) {
	// Скоуп по компании юнита (u.company_id) — та же логика, что в разделе
	// «Статистика»: часы личного профиля не должны включать работу в других
	// компаниях пользователя, иначе итоги двух разделов расходятся.
	out := &domain.ProfileStats{ByUnitTypes: []domain.UnitTypeHours{}}
	args := []any{userID, start, end}
	cond := companyCond(&args, companyID, "u.company_id")
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(`+hoursExpr+`), 0), COUNT(DISTINCT u.task_id)
		  FROM units u
		 WHERE u.user_id = $1 AND u.datetime_start >= $2 AND u.datetime_start <= $3`+cond,
		args...).Scan(&out.TotalHours, &out.TasksCount)
	if err != nil {
		return nil, err
	}
	out.TotalHours = round2(out.TotalHours)

	args = []any{userID, start, end}
	cond = companyCond(&args, companyID, "u.company_id")
	rows, err := r.pool.Query(ctx, `
		SELECT ut.id, ut.name, COALESCE(SUM(`+hoursExpr+`), 0), COUNT(DISTINCT u.task_id)
		  FROM unit_types ut
		  JOIN units u ON u.unit_type_id = ut.id
		 WHERE u.user_id = $1 AND u.datetime_start >= $2 AND u.datetime_start <= $3`+cond+`
		 GROUP BY ut.id, ut.name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ut domain.UnitTypeHours
		if err := rows.Scan(&ut.TypeID, &ut.Name, &ut.Hours, &ut.TasksCount); err != nil {
			return nil, err
		}
		ut.Hours = round2(ut.Hours)
		out.ByUnitTypes = append(out.ByUnitTypes, ut)
	}
	return out, rows.Err()
}

func (r *Repo) Responsibles(ctx context.Context, companyID *int64) ([]domain.Responsible, error) {
	args := []any{}
	cond := companyCond(&args, companyID, "t.company_id")
	// Должность переехала в user_companies (привязана к компании); тянем её
	// для компании задачи. is_active — глобальная активность аккаунта.
	rows, err := r.pool.Query(ctx, `
		SELECT us.id, us.fio, us.avatar_path,
		       (SELECT uc.post FROM user_companies uc
		          WHERE uc.user_id = us.id AND uc.company_id = t.company_id LIMIT 1),
		       SUM(CASE WHEN t.is_archived = FALSE THEN 1 ELSE 0 END),
		       SUM(CASE WHEN t.is_archived = TRUE THEN 1 ELSE 0 END)
		  FROM users us
		  JOIN tasks t ON t.responsible_user_id = us.id
		 WHERE us.is_active = TRUE`+cond+`
		 GROUP BY us.id, us.fio, us.avatar_path, t.company_id
		 ORDER BY SUM(CASE WHEN t.is_archived = FALSE THEN 1 ELSE 0 END) DESC,
		          SUM(CASE WHEN t.is_archived = TRUE THEN 1 ELSE 0 END) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Responsible{}
	for rows.Next() {
		var resp domain.Responsible
		if err := rows.Scan(&resp.UserID, &resp.FIO, &resp.AvatarPath, &resp.Post,
			&resp.OpenCount, &resp.ClosedCount); err != nil {
			return nil, err
		}
		out = append(out, resp)
	}
	return out, rows.Err()
}

func (r *Repo) VisibleEmployees(ctx context.Context, companyID *int64) ([]domain.EmployeeRef, error) {
	args := []any{}
	cond := memberCond(&args, companyID, "id")
	rows, err := r.pool.Query(ctx,
		`SELECT id, fio FROM users WHERE is_active = TRUE`+cond+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.EmployeeRef{}
	for rows.Next() {
		var e domain.EmployeeRef
		if err := rows.Scan(&e.ID, &e.FIO); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
