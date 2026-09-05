package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

// UserReader — read-only идентичность (владелец таблиц в рантайме — authsvc):
// auth-мидлварь и адресаты доступа.
type UserReader struct {
	pool *pgxpool.Pool
}

var _ domain.UserReader = (*UserReader)(nil)

func NewUserReader(pool *pgxpool.Pool) *UserReader { return &UserReader{pool: pool} }

func (r *UserReader) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, fio, avatar_path, is_active, is_super_admin
		  FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.FIO, &u.AvatarPath, &u.IsActive, &u.IsSuperAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// CompaniesOf — компании пользователя: через них приходит доступ к
// расписаниям, розданным компании. Пустой срез, а не nil: он уезжает в SQL как
// ANY($n), где nil означал бы «сравнить не с чем».
func (r *UserReader) CompaniesOf(ctx context.Context, userID int64) ([]int64, error) {
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

// Colleagues — кандидаты в адресаты доступа. Список ограничен компаниями
// спрашивающего: поделиться можно с коллегами, а не с каталогом платформы.
func (r *UserReader) Colleagues(ctx context.Context, userID int64, query string, limit int) ([]*domain.UserRef, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT u.id, u.fio, u.login, u.avatar_path, COALESCE(uc2.post, '')
		  FROM users u
		  JOIN user_companies uc2 ON uc2.user_id = u.id
		 WHERE u.is_active AND u.id <> $1
		   AND uc2.company_id IN (SELECT company_id FROM user_companies WHERE user_id = $1)
		   AND ($2 = '' OR u.fio ILIKE '%' || $2 || '%' OR u.login ILIKE '%' || $2 || '%')
		 ORDER BY u.fio
		 LIMIT $3`, userID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.UserRef{}
	for rows.Next() {
		var u domain.UserRef
		if err := rows.Scan(&u.ID, &u.FIO, &u.Login, &u.AvatarPath, &u.Post); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

// Companies — компании пользователя: им тоже можно открыть расписание целиком.
func (r *UserReader) Companies(ctx context.Context, userID int64) ([]*domain.CompanyRef, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.name
		  FROM companies c
		  JOIN user_companies uc ON uc.company_id = c.id
		 WHERE uc.user_id = $1 AND c.is_active
		 ORDER BY c.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.CompanyRef{}
	for rows.Next() {
		var c domain.CompanyRef
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}
