package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

// UserReader — read-only доступ к пользователям платформы (владелец таблицы
// в рантайме — authsvc); профиль в объёме UserDirectorySchema + проверки
// auth-мидлвари (is_hidden, активность компании).
type UserReader struct {
	pool *pgxpool.Pool
}

var _ domain.UserReader = (*UserReader)(nil)

func NewUserReader(pool *pgxpool.Pool) *UserReader {
	return &UserReader{pool: pool}
}

const userCols = `u.id, u.fio, u.login, u.post, u.role_id, r.name, r.level,
	u.company_id, u.phone, u.email, u.avatar_path, u.is_hidden, u.last_seen_at, c.is_active`

const userFrom = `
	FROM users u
	JOIN roles r ON r.id = u.role_id
	LEFT JOIN companies c ON c.id = u.company_id `

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	var companyActive *bool
	err := row.Scan(&u.ID, &u.FIO, &u.Login, &u.Post, &u.RoleID, &u.RoleName, &u.RoleLevel,
		&u.CompanyID, &u.Phone, &u.Email, &u.AvatarPath, &u.IsHidden, &u.LastSeenAt, &companyActive)
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

func (r *UserReader) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT `+userCols+userFrom+`WHERE u.id = $1`, id))
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

// DevChatUserIDs — адресаты событий dev-чата: владелец + все видимые
// Администраторы системы (company_id IS NULL).
func (r *UserReader) DevChatUserIDs(ctx context.Context, ownerID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id FROM users
		WHERE is_hidden = FALSE AND (id = $1 OR company_id IS NULL)`, ownerID)
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
