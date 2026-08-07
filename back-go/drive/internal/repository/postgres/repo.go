// Package postgres — персистентность диска (общая БД платформы, pgx, raw SQL).
package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.Repository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// nullableID — *int64 → аргумент запроса (nil становится NULL).
func nullableID(id *int64) any {
	if id == nil {
		return nil
	}
	return *id
}
