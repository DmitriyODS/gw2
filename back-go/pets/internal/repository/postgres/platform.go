package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// PlatformRepo — read-only доступ к таблицам других доменов: пользователи
// (authsvc), компании, юниты (tasksvc). Только чтение: владельцы таблиц —
// их сервисы.
type PlatformRepo struct {
	pool *pgxpool.Pool
}

var (
	_ domain.UserReader    = (*PlatformRepo)(nil)
	_ domain.CompanyReader = (*PlatformRepo)(nil)
	_ domain.WorkReader    = (*PlatformRepo)(nil)
)

func NewPlatformRepo(pool *pgxpool.Pool) *PlatformRepo {
	return &PlatformRepo{pool: pool}
}

// userRef — поля id/fio/avatar_path из LEFT JOIN users (всё nullable).
func userRef(id *int64, fio, avatar *string) *domain.UserRef {
	if id == nil {
		return nil
	}
	ref := &domain.UserRef{ID: *id, AvatarPath: avatar}
	if fio != nil {
		ref.FIO = *fio
	}
	return ref
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

func (r *PlatformRepo) ActiveCompanyIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM companies WHERE is_active = TRUE`)
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

// ──────────────────────── «Сейчас в эфире» ─────────────────────────

func (r *PlatformRepo) ListActiveUnits(ctx context.Context, companyID int64) ([]*domain.ActiveUnit, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT un.id, un.name, un.task_id, t.name, un.datetime_start,
		       u.id, u.fio, u.avatar_path
		FROM units un
		JOIN users u ON u.id = un.user_id
		LEFT JOIN tasks t ON t.id = un.task_id
		WHERE un.company_id = $1 AND un.datetime_end IS NULL AND u.is_active
		ORDER BY un.datetime_start ASC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.ActiveUnit
	for rows.Next() {
		var a domain.ActiveUnit
		var uid int64
		var fio string
		var avatar *string
		if err := rows.Scan(&a.ID, &a.Name, &a.TaskID, &a.TaskName, &a.StartedAt,
			&uid, &fio, &avatar); err != nil {
			return nil, err
		}
		a.User = &domain.UserRef{ID: uid, FIO: fio, AvatarPath: avatar}
		out = append(out, &a)
	}
	return out, rows.Err()
}
