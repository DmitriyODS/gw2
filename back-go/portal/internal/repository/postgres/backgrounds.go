package postgres

import (
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

// GetPortalBackground — рецепт оформления пользователя (nil — не задан).
func (r *Repo) GetPortalBackground(ctx domain.Ctx, userID int64) (json.RawMessage, error) {
	var recipe json.RawMessage
	err := r.pool.QueryRow(ctx,
		`SELECT recipe FROM portal_backgrounds WHERE user_id = $1`, userID).Scan(&recipe)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return recipe, nil
}

// UpsertPortalBackground — сохранить рецепт (одна строка на пользователя).
func (r *Repo) UpsertPortalBackground(ctx domain.Ctx, userID int64, recipe []byte) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO portal_backgrounds (user_id, recipe) VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET recipe = EXCLUDED.recipe, updated_at = now()`,
		userID, recipe)
	return err
}

// DeletePortalBackground — снять рецепт пользователя.
func (r *Repo) DeletePortalBackground(ctx domain.Ctx, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM portal_backgrounds WHERE user_id = $1`, userID)
	return err
}
