package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/push/internal/domain"
)

// TokenStore — таблица device_tokens (схему ведёт migrate-контейнер).
type TokenStore struct {
	pool *pgxpool.Pool
}

func NewTokenStore(pool *pgxpool.Pool) *TokenStore { return &TokenStore{pool: pool} }

// Upsert — токен принадлежит одному пользователю; при повторной регистрации
// (например, токен переехал на другой аккаунт) перепривязываем и обновляем
// время.
func (s *TokenStore) Upsert(ctx context.Context, t domain.DeviceToken) error {
	platform := t.Platform
	if platform == "" {
		platform = "android"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO device_tokens (token, user_id, platform, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (token) DO UPDATE
		   SET user_id = EXCLUDED.user_id,
		       platform = EXCLUDED.platform,
		       updated_at = now()`,
		t.Token, t.UserID, platform)
	return err
}

func (s *TokenStore) Delete(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM device_tokens WHERE token = $1`, token)
	return err
}

func (s *TokenStore) ListByUsers(ctx context.Context, userIDs []int64) ([]domain.DeviceToken, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT token, user_id, platform FROM device_tokens WHERE user_id = ANY($1)`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.DeviceToken
	for rows.Next() {
		var t domain.DeviceToken
		if err := rows.Scan(&t.Token, &t.UserID, &t.Platform); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
