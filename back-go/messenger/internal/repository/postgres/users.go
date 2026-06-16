package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

// UserReader — read-only доступ к пользователям платформы (владелец таблицы
// в рантайме — authsvc). Грузит ТОЛЬКО идентичность из users; роль и компания
// развязаны с users (живут в user_companies) и приходят из токена в authSource.
type UserReader struct {
	pool *pgxpool.Pool
}

var _ domain.UserReader = (*UserReader)(nil)

func NewUserReader(pool *pgxpool.Pool) *UserReader {
	return &UserReader{pool: pool}
}

const userCols = `u.id, u.fio, u.login, u.avatar_path, u.phone, u.email,
	u.is_active, u.is_super_admin, u.last_seen_at`

const userFrom = ` FROM users u `

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.FIO, &u.Login, &u.AvatarPath, &u.Phone, &u.Email,
		&u.IsActive, &u.IsSuperAdmin, &u.LastSeenAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserReader) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+userFrom+`WHERE u.id = $1`, id))
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

func (r *UserReader) ListUsers(ctx context.Context, ids []int64) ([]*domain.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT `+userCols+userFrom+`WHERE u.id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// DevChatUserIDs — адресаты событий dev-чата: владелец + все активные
// супер-админы (техподдержка).
func (r *UserReader) DevChatUserIDs(ctx context.Context, ownerID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id FROM users
		WHERE is_active = TRUE AND (id = $1 OR is_super_admin = TRUE)`, ownerID)
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
