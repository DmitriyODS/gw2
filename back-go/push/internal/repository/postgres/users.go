package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserDirectory — read-only ФИО пользователей (таблицу ведёт authsvc).
type UserDirectory struct {
	pool *pgxpool.Pool
}

func NewUserDirectory(pool *pgxpool.Pool) *UserDirectory { return &UserDirectory{pool: pool} }

func (d *UserDirectory) Names(ctx context.Context, ids []int64) (map[int64]string, error) {
	out := map[int64]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := d.pool.Query(ctx, `SELECT id, fio FROM users WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var fio string
		if err := rows.Scan(&id, &fio); err != nil {
			return nil, err
		}
		out[id] = fio
	}
	return out, rows.Err()
}

// MembersOf — участники компании (user_companies ведёт authsvc; read-only —
// company-wide пуши вроде поста портала адресуются всей компании).
func (d *UserDirectory) MembersOf(ctx context.Context, companyID int64) ([]int64, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT user_id FROM user_companies WHERE company_id = $1`, companyID)
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
