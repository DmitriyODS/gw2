// Package postgres — персистентность мессенджера (pgx, raw SQL по таблицам,
// схему которых ведёт Alembic во Flask) + read-only лукапы pets/tasks/calls.
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.Repository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// querier — pool или активная транзакция из контекста.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type txKey struct{}

func (r *Repo) q(ctx context.Context) querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return r.pool
}

// RunInTx — fn в одной транзакции; вложенные вызовы переиспользуют её же.
func (r *Repo) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback после commit — no-op
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// hiddenCol / pinCol — колонка стороны ('a'/'b'); side валидируется доменом.
func hiddenCol(side string) string {
	if side == domain.SideB {
		return "hidden_for_b"
	}
	return "hidden_for_a"
}

func pinCol(side string) string {
	if side == domain.SideB {
		return "pinned_at_b"
	}
	return "pinned_at_a"
}
