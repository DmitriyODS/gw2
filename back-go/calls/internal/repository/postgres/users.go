package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
)

type UserReader struct {
	pool *pgxpool.Pool
}

var _ domain.UserReader = (*UserReader)(nil)

func NewUserReader(pool *pgxpool.Pool) *UserReader {
	return &UserReader{pool: pool}
}

func (r *UserReader) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	var companyActive *bool
	err := r.pool.QueryRow(ctx, `
		SELECT u.id, u.fio, u.avatar_path, u.company_id, u.is_hidden, c.is_active
		FROM users u
		LEFT JOIN companies c ON c.id = u.company_id
		WHERE u.id = $1`, id,
	).Scan(&u.ID, &u.FIO, &u.AvatarPath, &u.CompanyID, &u.IsHidden, &companyActive)
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

func (r *UserReader) ListVisibleUsers(ctx context.Context, ids []int64) ([]*domain.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, fio, avatar_path, company_id, is_hidden
		FROM users
		WHERE id = ANY($1) AND is_hidden = FALSE`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.FIO, &u.AvatarPath, &u.CompanyID, &u.IsHidden); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

// prefixed — "id, kind" → "c.id, c.kind" (для запросов с JOIN).
func prefixed(alias, cols string) string {
	parts := strings.Split(cols, ",")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}
