// Package postgres — read-only доступ gateway к пользователям платформы
// (auth-мидлварь REST-роутов) и запись users.last_seen_at для presence.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/pkg/pasetoauth"
)

type UserReader struct {
	pool *pgxpool.Pool
}

func NewUserReader(pool *pgxpool.Pool) *UserReader {
	return &UserReader{pool: pool}
}

// AuthInfo — сверка пользователя для pkg-мидлвари (is_hidden, активность
// компании, уровень роли).
func (r *UserReader) AuthInfo(ctx context.Context, userID int64) (*pasetoauth.AuthInfo, error) {
	var (
		info          pasetoauth.AuthInfo
		companyActive *bool
	)
	err := r.pool.QueryRow(ctx, `
		SELECT r.level, u.is_hidden, c.is_active
		  FROM users u
		  JOIN roles r ON r.id = u.role_id
		  LEFT JOIN companies c ON c.id = u.company_id
		 WHERE u.id = $1`, userID).
		Scan(&info.RoleLevel, &info.IsHidden, &companyActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	// Пользователь без компании (Администратор системы) считается активным.
	info.CompanyActive = companyActive == nil || *companyActive
	return &info, nil
}
