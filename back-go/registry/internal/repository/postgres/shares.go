package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

const shareCols = `id, registry_id, code, created_by, created_at`

func scanShare(row pgx.Row) (*domain.Share, error) {
	var s domain.Share
	err := row.Scan(&s.ID, &s.RegistryID, &s.Code, &s.CreatedBy, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) CreateShare(ctx context.Context, s *domain.Share) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO registry_shares (registry_id, code, created_by)
		 VALUES ($1, $2, $3) RETURNING id, created_at`,
		s.RegistryID, s.Code, s.CreatedBy).Scan(&s.ID, &s.CreatedAt)
}

func (r *Repo) ListShares(ctx context.Context, registryID int64) ([]*domain.Share, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+shareCols+` FROM registry_shares WHERE registry_id = $1 ORDER BY created_at DESC`,
		registryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Share{}
	for rows.Next() {
		s, err := scanShare(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repo) GetShareByCode(ctx context.Context, code string) (*domain.Share, error) {
	return scanShare(r.pool.QueryRow(ctx,
		`SELECT `+shareCols+` FROM registry_shares WHERE code = $1`, code))
}

func (r *Repo) DeleteShare(ctx context.Context, id, registryID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM registry_shares WHERE id = $1 AND registry_id = $2`, id, registryID)
	return err
}
