package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// ActivityRepo — приватная история активности питомца (pet_activity_log).
type ActivityRepo struct {
	pool *pgxpool.Pool
}

var _ domain.ActivityRepo = (*ActivityRepo)(nil)

func NewActivityRepo(pool *pgxpool.Pool) *ActivityRepo {
	return &ActivityRepo{pool: pool}
}

func (r *ActivityRepo) Append(ctx context.Context, petUserID int64, kind string, payload map[string]any) error {
	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO pet_activity_log (pet_user_id, kind, payload, created_at)
		VALUES ($1, $2, $3, now())`, petUserID, kind, raw)
	return err
}

func (r *ActivityRepo) ListForPet(ctx context.Context, petUserID int64, limit int) ([]*domain.ActivityLogEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, pet_user_id, kind, payload, created_at
		FROM pet_activity_log
		WHERE pet_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, petUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.ActivityLogEntry
	for rows.Next() {
		var e domain.ActivityLogEntry
		var raw []byte
		if err := rows.Scan(&e.ID, &e.PetUserID, &e.Kind, &raw, &e.CreatedAt); err != nil {
			return nil, err
		}
		payload := map[string]any{}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &payload)
		}
		e.Payload = payload
		out = append(out, &e)
	}
	return out, rows.Err()
}
