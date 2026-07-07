package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

const commentCols = `id, post_id, author_id, text, created_at`

func scanComment(row pgx.Row) (*domain.Comment, error) {
	var c domain.Comment
	err := row.Scan(&c.ID, &c.PostID, &c.AuthorID, &c.Text, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repo) ListComments(ctx context.Context, postID int64) ([]*domain.Comment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+commentCols+` FROM portal_comments WHERE post_id = $1 ORDER BY created_at, id`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Comment{}
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repo) GetComment(ctx context.Context, id int64) (*domain.Comment, error) {
	return scanComment(r.pool.QueryRow(ctx, `SELECT `+commentCols+` FROM portal_comments WHERE id = $1`, id))
}

func (r *Repo) CreateComment(ctx context.Context, c *domain.Comment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO portal_comments (post_id, author_id, text)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		c.PostID, c.AuthorID, c.Text,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *Repo) DeleteComment(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM portal_comments WHERE id = $1`, id)
	return err
}
