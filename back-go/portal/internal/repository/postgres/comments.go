package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

const commentCols = `id, post_id, author_id, reply_to_id, text, created_at`

func scanComment(row pgx.Row) (*domain.Comment, error) {
	var c domain.Comment
	err := row.Scan(&c.ID, &c.PostID, &c.AuthorID, &c.ReplyToID, &c.Text, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListComments — обсуждение поста в хронологии; лайки считаются тем же
// запросом (счётчик и «мой лайк» — агрегатами, без N+1). Дерево ответов
// строит клиент по reply_to_id.
func (r *Repo) ListComments(ctx context.Context, postID, viewerID int64) ([]*domain.Comment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.post_id, c.author_id, c.reply_to_id, c.text, c.created_at,
			count(l.user_id) AS like_count,
			bool_or(l.user_id = $2) AS liked
		FROM portal_comments c
		LEFT JOIN portal_comment_likes l ON l.comment_id = c.id
		WHERE c.post_id = $1
		GROUP BY c.id
		ORDER BY c.created_at, c.id`, postID, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Comment{}
	for rows.Next() {
		var c domain.Comment
		var liked *bool // bool_or по пустой группе (лайков нет) — NULL
		if err := rows.Scan(&c.ID, &c.PostID, &c.AuthorID, &c.ReplyToID, &c.Text,
			&c.CreatedAt, &c.LikeCount, &liked); err != nil {
			return nil, err
		}
		c.Liked = liked != nil && *liked
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *Repo) GetComment(ctx context.Context, id int64) (*domain.Comment, error) {
	return scanComment(r.pool.QueryRow(ctx, `SELECT `+commentCols+` FROM portal_comments WHERE id = $1`, id))
}

func (r *Repo) CreateComment(ctx context.Context, c *domain.Comment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO portal_comments (post_id, author_id, reply_to_id, text)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		c.PostID, c.AuthorID, c.ReplyToID, c.Text,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *Repo) DeleteComment(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM portal_comments WHERE id = $1`, id)
	return err
}

// ToggleCommentLike — переключение лайка одной транзакцией: удаляем свою
// строку, а если удалять было нечего — вставляем. Счётчик читается там же,
// поэтому ответ не разъезжается с конкурентными лайками коллег.
func (r *Repo) ToggleCommentLike(ctx context.Context, commentID, userID int64) (bool, int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // после Commit — no-op

	tag, err := tx.Exec(ctx,
		`DELETE FROM portal_comment_likes WHERE comment_id = $1 AND user_id = $2`, commentID, userID)
	if err != nil {
		return false, 0, err
	}
	liked := tag.RowsAffected() == 0
	if liked {
		if _, err := tx.Exec(ctx,
			`INSERT INTO portal_comment_likes (comment_id, user_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`, commentID, userID); err != nil {
			return false, 0, err
		}
	}
	var count int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM portal_comment_likes WHERE comment_id = $1`, commentID).Scan(&count); err != nil {
		return false, 0, err
	}
	return liked, count, tx.Commit(ctx)
}
