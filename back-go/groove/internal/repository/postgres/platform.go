package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

// PlatformRepo — read-only доступ к таблицам других доменов: пользователи
// (authsvc), компании, задачи/юниты/отделы (Flask), pet-чаты (msgsvc).
// Только чтение: владельцы таблиц — их сервисы.
type PlatformRepo struct {
	pool *pgxpool.Pool
}

var (
	_ domain.UserReader         = (*PlatformRepo)(nil)
	_ domain.CompanyReader      = (*PlatformRepo)(nil)
	_ domain.WorkReader         = (*PlatformRepo)(nil)
	_ domain.ConversationReader = (*PlatformRepo)(nil)
)

func NewPlatformRepo(pool *pgxpool.Pool) *PlatformRepo {
	return &PlatformRepo{pool: pool}
}

// ───────────────────────────── пользователи ────────────────────────

// GetUser — только идентичность пользователя. Роль и компания приходят из
// access-токена (их проставляет транспорт), не читаются из users.
func (r *PlatformRepo) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT u.id, u.fio, u.avatar_path, u.is_active, u.is_super_admin
		FROM users u
		WHERE u.id = $1`, id,
	).Scan(&u.ID, &u.FIO, &u.AvatarPath, &u.IsActive, &u.IsSuperAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// IsCompanyMember — членство пользователя в компании (таблица user_companies).
func (r *PlatformRepo) IsCompanyMember(ctx context.Context, userID, companyID int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_companies WHERE user_id = $1 AND company_id = $2)`,
		userID, companyID).Scan(&ok)
	return ok, err
}

// CompanyActive — активность ИМЕННО выбранной (активной) компании сессии.
func (r *PlatformRepo) CompanyActive(ctx context.Context, companyID *int64) (bool, error) {
	if companyID == nil {
		return true, nil
	}
	var active bool
	err := r.pool.QueryRow(ctx,
		`SELECT is_active FROM companies WHERE id = $1`, *companyID).Scan(&active)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return active, nil
}

// ───────────────────────────── компании ────────────────────────────

func (r *PlatformRepo) companyIDs(ctx context.Context, where string) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM companies WHERE `+where)
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

func (r *PlatformRepo) ActiveCompanyIDs(ctx context.Context) ([]int64, error) {
	return r.companyIDs(ctx, `is_active = TRUE`)
}

func (r *PlatformRepo) AICompanyIDs(ctx context.Context) ([]int64, error) {
	return r.companyIDs(ctx, `ai_enabled = TRUE`)
}

func (r *PlatformRepo) WeekendDays(ctx context.Context, companyID int64) ([]int, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx,
		`SELECT settings FROM companies WHERE id = $1`, companyID).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return append([]int{}, domain.DefaultWeekend...), nil
		}
		return nil, err
	}
	var settings struct {
		WeekendDays []any `json:"weekend_days"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &settings) != nil || settings.WeekendDays == nil {
		return append([]int{}, domain.DefaultWeekend...), nil
	}
	// На любой мусор в настройках отвечаем дефолтом Сб+Вс (как Flask).
	var days []int
	for _, v := range settings.WeekendDays {
		f, ok := v.(float64)
		if !ok || f != float64(int(f)) || int(f) < 0 || int(f) > 6 {
			return append([]int{}, domain.DefaultWeekend...), nil
		}
		days = append(days, int(f))
	}
	return days, nil
}

// GrooveEnabled — включён ли режим «Мой Groove» у компании
// (settings.uses_groove). Отсутствие ключа, мусор или несуществующая
// компания → включён (как и на фронте: uses_groove !== false).
func (r *PlatformRepo) GrooveEnabled(ctx context.Context, companyID int64) (bool, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx,
		`SELECT settings FROM companies WHERE id = $1`, companyID).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	var settings struct {
		UsesGroove *bool `json:"uses_groove"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &settings) != nil || settings.UsesGroove == nil {
		return true, nil
	}
	return *settings.UsesGroove, nil
}

// ───────────────────────────── pet-чаты ────────────────────────────

func (r *PlatformRepo) GetConversation(ctx context.Context, id int64) (*domain.PetConversation, error) {
	var conv domain.PetConversation
	var companyID *int64
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_a_id, company_id, is_pet_chat
		FROM conversations WHERE id = $1`, id,
	).Scan(&conv.ID, &conv.OwnerID, &companyID, &conv.IsPetChat)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if companyID != nil {
		conv.CompanyID = *companyID
	}
	return &conv, nil
}

func (r *PlatformRepo) GetPetConversationByOwner(ctx context.Context, ownerID int64) (*domain.PetConversation, error) {
	var conv domain.PetConversation
	var companyID *int64
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_a_id, company_id, is_pet_chat
		FROM conversations WHERE user_a_id = $1 AND is_pet_chat = TRUE`, ownerID,
	).Scan(&conv.ID, &conv.OwnerID, &companyID, &conv.IsPetChat)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if companyID != nil {
		conv.CompanyID = *companyID
	}
	return &conv, nil
}

// ──────────────────────── задачи и юниты ───────────────────────────

// «Мои» задачи: сотрудник ответственный ИЛИ хоть раз работал по задаче —
// единая трактовка с has_units="mine" в списке задач Flask.
const minePredicate = `(t.responsible_user_id = $1
	OR EXISTS (SELECT 1 FROM units mu WHERE mu.task_id = t.id AND mu.user_id = $1))`

func (r *PlatformRepo) CountUserActive(ctx context.Context, userID, companyID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(t.id) FROM tasks t
		WHERE t.is_archived = FALSE AND t.company_id = $2 AND `+minePredicate,
		userID, companyID).Scan(&count)
	return count, err
}

func (r *PlatformRepo) UserStale(ctx context.Context, userID, companyID int64,
	threshold time.Time, limit int) ([]*domain.StaleTask, error) {

	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.name, d.name, t.received_at
		FROM tasks t LEFT JOIN departments d ON d.id = t.department_id
		WHERE t.is_archived = FALSE AND t.received_at < $3 AND t.company_id = $2
		  AND `+minePredicate+`
		ORDER BY t.received_at ASC
		LIMIT $4`, userID, companyID, threshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.StaleTask
	for rows.Next() {
		var t domain.StaleTask
		if err := rows.Scan(&t.ID, &t.Name, &t.DepartmentName, &t.ReceivedAt); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

func (r *PlatformRepo) ActiveUnitForUser(ctx context.Context, userID int64) (int64, int64, error) {
	var unitID, companyID int64
	err := r.pool.QueryRow(ctx, `
		SELECT id, company_id FROM units
		WHERE user_id = $1 AND datetime_end IS NULL`, userID,
	).Scan(&unitID, &companyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	return unitID, companyID, nil
}

// DaySummary — активность компании за интервал [start, end) для события
// «Итоги дня»: юниты и часы (незавершённые считаются по текущему моменту),
// закрытые задачи и лидер дня по часам юнитов.
func (r *PlatformRepo) DaySummary(ctx context.Context, companyID int64,
	start, end time.Time) (*domain.DaySummaryStats, error) {

	var s domain.DaySummaryStats
	err := r.pool.QueryRow(ctx, `
		SELECT count(id), COALESCE(sum(`+hoursExpr+`), 0)
		FROM units un
		WHERE un.company_id = $1 AND un.datetime_start >= $2 AND un.datetime_start < $3`,
		companyID, start, end,
	).Scan(&s.UnitsCount, &s.TotalHours)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx, `
		SELECT count(id) FROM tasks
		WHERE company_id = $1 AND is_archived = TRUE
		  AND archived_at >= $2 AND archived_at < $3`,
		companyID, start, end,
	).Scan(&s.TasksClosed)
	if err != nil {
		return nil, err
	}
	var leaderID int64
	var fio string
	var avatar *string
	var hours float64
	err = r.pool.QueryRow(ctx, `
		SELECT u.id, u.fio, u.avatar_path, COALESCE(sum(`+hoursExpr+`), 0) AS h
		FROM units un JOIN users u ON u.id = un.user_id
		WHERE un.company_id = $1 AND un.datetime_start >= $2 AND un.datetime_start < $3
		GROUP BY u.id, u.fio, u.avatar_path
		ORDER BY h DESC
		LIMIT 1`, companyID, start, end,
	).Scan(&leaderID, &fio, &avatar, &hours)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		s.LeaderID = &leaderID
		s.LeaderFIO = fio
		s.LeaderAvatar = avatar
		s.LeaderHours = hours
	}
	return &s, nil
}

// ─────────────── статистика (инструменты Грувика, дайджест) ────────

const hoursExpr = `EXTRACT(EPOCH FROM (COALESCE(un.datetime_end, now()) - un.datetime_start)) / 3600`

func (r *PlatformRepo) CommonMetrics(ctx context.Context, companyID int64,
	start, end time.Time) (*domain.CommonMetrics, error) {

	var m domain.CommonMetrics
	err := r.pool.QueryRow(ctx, `
		SELECT
			count(id) FILTER (WHERE is_archived = FALSE AND received_at < $2),
			count(id) FILTER (WHERE received_at >= $2 AND received_at <= $3),
			count(id) FILTER (WHERE is_archived = TRUE AND archived_at >= $2 AND archived_at <= $3),
			count(id) FILTER (WHERE is_archived = FALSE)
		FROM tasks WHERE company_id = $1`,
		companyID, start, end,
	).Scan(&m.Debt, &m.Received, &m.Closed, &m.Remaining)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *PlatformRepo) TopEmployees(ctx context.Context, companyID int64,
	start, end time.Time) ([]domain.EmployeeStat, error) {

	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.fio, count(DISTINCT un.task_id),
		       COALESCE(sum(`+hoursExpr+`), 0)
		FROM users u JOIN units un ON un.user_id = u.id
		WHERE un.datetime_start >= $2 AND un.datetime_start <= $3 AND un.company_id = $1
		GROUP BY u.id, u.fio
		ORDER BY sum(`+hoursExpr+`) DESC`, companyID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.EmployeeStat
	for rows.Next() {
		var s domain.EmployeeStat
		if err := rows.Scan(&s.UserID, &s.FIO, &s.TasksCount, &s.TotalHours); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PlatformRepo) ByDepartments(ctx context.Context, companyID int64,
	start, end time.Time) ([]domain.DeptStat, error) {

	rows, err := r.pool.Query(ctx, `
		SELECT d.id, d.name, count(DISTINCT t.id)
		FROM departments d JOIN tasks t ON t.department_id = d.id
		WHERE t.received_at >= $2 AND t.received_at <= $3 AND d.company_id = $1
		GROUP BY d.id, d.name
		ORDER BY count(DISTINCT t.id) DESC`, companyID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.DeptStat
	for rows.Next() {
		var s domain.DeptStat
		if err := rows.Scan(&s.ID, &s.Name, &s.TasksCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PlatformRepo) ByUnitTypes(ctx context.Context, companyID int64,
	start, end time.Time) ([]domain.UnitTypeStat, error) {

	rows, err := r.pool.Query(ctx, `
		SELECT ut.name, COALESCE(sum(`+hoursExpr+`), 0), count(DISTINCT un.task_id)
		FROM unit_types ut JOIN units un ON un.unit_type_id = ut.id
		WHERE un.datetime_start >= $2 AND un.datetime_start <= $3 AND ut.company_id = $1
		GROUP BY ut.id, ut.name
		ORDER BY sum(`+hoursExpr+`) DESC`, companyID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.UnitTypeStat
	for rows.Next() {
		var s domain.UnitTypeStat
		if err := rows.Scan(&s.Name, &s.TotalHours, &s.TasksCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PlatformRepo) Calendar(ctx context.Context, companyID int64,
	start, end time.Time) ([]domain.CalendarDay, error) {

	calendar := map[string]*domain.CalendarDay{}
	day := func(d string) *domain.CalendarDay {
		if calendar[d] == nil {
			calendar[d] = &domain.CalendarDay{Date: d}
		}
		return calendar[d]
	}

	collect := func(query string, apply func(*domain.CalendarDay, int, float64)) error {
		rows, err := r.pool.Query(ctx, query, companyID, start, end)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var d time.Time
			var count int
			var hours float64
			if err := rows.Scan(&d, &count, &hours); err != nil {
				return err
			}
			apply(day(d.Format("2006-01-02")), count, hours)
		}
		return rows.Err()
	}

	if err := collect(`
		SELECT date(received_at), count(id), 0::float8 FROM tasks
		WHERE received_at >= $2 AND received_at <= $3 AND company_id = $1
		GROUP BY date(received_at)`,
		func(d *domain.CalendarDay, c int, _ float64) { d.Received = c }); err != nil {
		return nil, err
	}
	if err := collect(`
		SELECT date(archived_at), count(id), 0::float8 FROM tasks
		WHERE is_archived = TRUE AND archived_at >= $2 AND archived_at <= $3 AND company_id = $1
		GROUP BY date(archived_at)`,
		func(d *domain.CalendarDay, c int, _ float64) { d.Closed = c }); err != nil {
		return nil, err
	}
	if err := collect(`
		SELECT date(un.datetime_start), 0, sum(`+hoursExpr+`)
		FROM units un JOIN tasks t ON t.id = un.task_id
		WHERE un.datetime_start >= $2 AND un.datetime_start <= $3 AND t.company_id = $1
		GROUP BY date(un.datetime_start)`,
		func(d *domain.CalendarDay, _ int, h float64) { d.TotalHours = h }); err != nil {
		return nil, err
	}

	out := make([]domain.CalendarDay, 0, len(calendar))
	for _, d := range calendar {
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out, nil
}
