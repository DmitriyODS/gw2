package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

// UserReader — read-only доступ к пользователям платформы (владелец таблицы
// в рантайме — authsvc); объём — глобальная активность аккаунта и супер-админ
// для auth-мидлвари. Роль/компания развязаны с users — берутся из токена.
type UserReader struct {
	pool *pgxpool.Pool
}

var _ domain.UserReader = (*UserReader)(nil)

func NewUserReader(pool *pgxpool.Pool) *UserReader {
	return &UserReader{pool: pool}
}

func (r *UserReader) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT u.id, u.is_active, u.is_super_admin
		  FROM users u
		 WHERE u.id = $1`, id).
		Scan(&u.ID, &u.IsActive, &u.IsSuperAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
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
