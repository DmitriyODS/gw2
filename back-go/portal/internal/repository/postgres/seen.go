package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repo) SeenAt(ctx context.Context, userID, companyID int64) (*time.Time, error) {
	var at time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT seen_at FROM portal_seen WHERE user_id = $1 AND company_id = $2`,
		userID, companyID).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &at, nil
}

func (r *Repo) MarkSeen(ctx context.Context, userID, companyID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO portal_seen (user_id, company_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, company_id) DO UPDATE SET seen_at = now()`,
		userID, companyID)
	return err
}

func (r *Repo) CountPostsAfter(ctx context.Context, companyID, excludeAuthorID int64, after *time.Time) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM portal_posts
		WHERE company_id = $1 AND author_id <> $2 AND ($3::timestamptz IS NULL OR created_at > $3)`,
		companyID, excludeAuthorID, after).Scan(&n)
	return n, err
}
