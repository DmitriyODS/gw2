package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

// UserReader — read-only доступ к пользователям платформы (владелец таблицы
// в рантайме — authsvc); объём — auth-мидлварь (is_hidden, активность
// компании), валидация ответственного, ФИО для уведомления force-stop.
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

func (r *UserReader) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	var companyActive *bool
	err := r.pool.QueryRow(ctx, `
		SELECT u.id, u.fio, u.post, u.avatar_path, r.level, u.company_id,
		       u.is_hidden, u.is_root_admin, c.is_active
		  FROM users u
		  JOIN roles r ON r.id = u.role_id
		  LEFT JOIN companies c ON c.id = u.company_id
		 WHERE u.id = $1`, id).
		Scan(&u.ID, &u.FIO, &u.Post, &u.AvatarPath, &u.RoleLevel, &u.CompanyID,
			&u.IsHidden, &u.IsRootAdmin, &companyActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	// Пользователь без компании (Администратор системы) считается активным.
	u.CompanyActive = companyActive == nil || *companyActive
	return &u, nil
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
