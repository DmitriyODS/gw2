package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// PasswordResetStore — токены сброса пароля (password_resets).
type PasswordResetStore struct {
	pool *pgxpool.Pool
}

func NewPasswordResetStore(pool *pgxpool.Pool) *PasswordResetStore {
	return &PasswordResetStore{pool: pool}
}

var _ domain.PasswordResetStore = (*PasswordResetStore)(nil)

func (r *PasswordResetStore) Upsert(ctx context.Context, userID int64, token string, expiresAt, sentAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO password_resets (user_id, token, expires_at, last_sent_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		   SET token = EXCLUDED.token, expires_at = EXCLUDED.expires_at, last_sent_at = EXCLUDED.last_sent_at`,
		userID, token, expiresAt, sentAt)
	return err
}

func scanReset(row pgx.Row) (*domain.PasswordReset, error) {
	var p domain.PasswordReset
	err := row.Scan(&p.UserID, &p.Token, &p.ExpiresAt, &p.LastSentAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

const resetCols = `user_id, token, expires_at, last_sent_at`

func (r *PasswordResetStore) GetByToken(ctx context.Context, token string) (*domain.PasswordReset, error) {
	return scanReset(r.pool.QueryRow(ctx,
		`SELECT `+resetCols+` FROM password_resets WHERE token = $1`, token))
}

func (r *PasswordResetStore) GetByUserID(ctx context.Context, userID int64) (*domain.PasswordReset, error) {
	return scanReset(r.pool.QueryRow(ctx,
		`SELECT `+resetCols+` FROM password_resets WHERE user_id = $1`, userID))
}

func (r *PasswordResetStore) Delete(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM password_resets WHERE user_id = $1`, userID)
	return err
}
