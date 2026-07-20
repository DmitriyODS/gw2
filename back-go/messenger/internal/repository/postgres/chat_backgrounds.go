package postgres

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

// ListChatBackgrounds — все рецепты пользователя: строка с conversation_id NULL
// (общий дефолт) и переопределения по чатам.
func (r *Repo) ListChatBackgrounds(ctx context.Context, userID int64) ([]*domain.ChatBackground, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT conversation_id, recipe FROM chat_backgrounds WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.ChatBackground
	for rows.Next() {
		bg := &domain.ChatBackground{}
		if err := rows.Scan(&bg.ConversationID, &bg.Recipe); err != nil {
			return nil, err
		}
		out = append(out, bg)
	}
	return out, rows.Err()
}

// UpsertChatBackground — сохранить рецепт. Дефолт (convID nil) и переопределение
// по чату лежат под разными partial unique index'ами — инференс конфликта разный.
func (r *Repo) UpsertChatBackground(ctx context.Context, userID int64, convID *int64, recipe []byte) error {
	if convID == nil {
		_, err := r.q(ctx).Exec(ctx,
			`INSERT INTO chat_backgrounds (user_id, conversation_id, recipe)
			 VALUES ($1, NULL, $2)
			 ON CONFLICT (user_id) WHERE conversation_id IS NULL
			 DO UPDATE SET recipe = EXCLUDED.recipe, updated_at = now()`,
			userID, recipe)
		return err
	}
	_, err := r.q(ctx).Exec(ctx,
		`INSERT INTO chat_backgrounds (user_id, conversation_id, recipe)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, conversation_id) WHERE conversation_id IS NOT NULL
		 DO UPDATE SET recipe = EXCLUDED.recipe, updated_at = now()`,
		userID, *convID, recipe)
	return err
}

// DeleteChatBackground — снять рецепт (convID nil — дефолт). IS NOT DISTINCT FROM
// корректно матчит NULL.
func (r *Repo) DeleteChatBackground(ctx context.Context, userID int64, convID *int64) error {
	_, err := r.q(ctx).Exec(ctx,
		`DELETE FROM chat_backgrounds
		 WHERE user_id = $1 AND conversation_id IS NOT DISTINCT FROM $2`,
		userID, convID)
	return err
}
