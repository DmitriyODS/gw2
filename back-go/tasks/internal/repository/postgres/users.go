package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

// UserReader — read-only доступ к пользователям платформы (владелец таблицы
// в рантайме — authsvc); объём — auth-мидлварь (is_active, активность
// выбранной компании), валидация ответственного, ФИО для уведомления force-stop.
type UserReader struct {
	pool *pgxpool.Pool
}

var (
	_ domain.UserReader    = (*UserReader)(nil)
	_ domain.CompanyReader = (*UserReader)(nil)
)

func NewUserReader(pool *pgxpool.Pool) *UserReader {
	return &UserReader{pool: pool}
}

// YougileEnabled — флаг uses_yougile из JSONB-настроек компании.
// Семантика _yougile_enabled во Flask: компании нет / settings пусты →
// true; ключа нет → true.
func (r *UserReader) YougileEnabled(ctx context.Context, companyID int64) (bool, error) {
	var settings map[string]any
	err := r.pool.QueryRow(ctx,
		`SELECT settings FROM companies WHERE id = $1`, companyID).Scan(&settings)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	if len(settings) == 0 {
		return true, nil
	}
	v, ok := settings["uses_yougile"]
	if !ok {
		return true, nil
	}
	b, ok := v.(bool)
	if !ok {
		return true, nil
	}
	return b, nil
}

// GetUser — только идентичность из users (компания/роль развязаны: активная
// компания и роль в ней приходят из токена, заполняются в authSource).
func (r *UserReader) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, fio, avatar_path, is_active, is_super_admin, on_vacation
		  FROM users
		 WHERE id = $1`, id).
		Scan(&u.ID, &u.FIO, &u.AvatarPath, &u.IsActive, &u.IsSuperAdmin, &u.OnVacation)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// IsCompanyMember — состоит ли пользователь в компании по user_companies
// (многокомпанийность: один аккаунт может быть в нескольких компаниях; первичная
// users.company_id — лишь одна из них).
func (r *UserReader) IsCompanyMember(ctx context.Context, userID, companyID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_companies WHERE user_id = $1 AND company_id = $2)`,
		userID, companyID).Scan(&exists)
	return exists, err
}

// CompanyActive — активность ИМЕННО выбранной (активной) компании сессии.
func (r *UserReader) CompanyActive(ctx context.Context, companyID *int64) (bool, error) {
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
