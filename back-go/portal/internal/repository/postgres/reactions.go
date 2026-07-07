package postgres

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

// AddReaction — идемпотентно (ON CONFLICT DO NOTHING): повторный тап тем же
// эмодзи не плодит дубли и не падает на уникальном ключе.
func (r *Repo) AddReaction(ctx context.Context, react *domain.Reaction) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO portal_reactions (post_id, user_id, emoji)
		VALUES ($1, $2, $3)
		ON CONFLICT (post_id, user_id, emoji) DO NOTHING`,
		react.PostID, react.UserID, react.Emoji)
	return err
}

func (r *Repo) RemoveReaction(ctx context.Context, postID, userID int64, emoji string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM portal_reactions WHERE post_id = $1 AND user_id = $2 AND emoji = $3`,
		postID, userID, emoji)
	return err
}
