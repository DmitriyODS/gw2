package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
)

/* Read-only идентичность: таблицы users и user_companies ведёт authsvc, диск
   их только читает — тот же приём, что у notesvc и remindersvc. */

type UserReader struct {
	pool *pgxpool.Pool
}

var _ domain.UserReader = (*UserReader)(nil)

func NewUserReader(pool *pgxpool.Pool) *UserReader { return &UserReader{pool: pool} }

func (r *UserReader) GetUser(ctx domain.Ctx, id int64) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, fio, login, avatar_path FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.FIO, &u.Login, &u.AvatarPath)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// UserCompanies — компании пользователя: по ним действует доступ, выданный
// не персонально, а всей компании.
func (r *UserReader) UserCompanies(ctx domain.Ctx, userID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT company_id FROM user_companies WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// SearchUsers — выбор адресата доступа: по имени и логину.
func (r *UserReader) SearchUsers(ctx domain.Ctx, query string, limit int) ([]*domain.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, fio, login, avatar_path FROM users
		 WHERE is_active AND (fio ILIKE $1 OR login ILIKE $1)
		 ORDER BY fio LIMIT $2`, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.User{}
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.FIO, &u.Login, &u.AvatarPath); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}
