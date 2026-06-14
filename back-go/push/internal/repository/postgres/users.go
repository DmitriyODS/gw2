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
