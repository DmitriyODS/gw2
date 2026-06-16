package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// VerificationStore — коды/ссылки подтверждения email (email_verifications).
type VerificationStore struct {
	pool *pgxpool.Pool
}

func NewVerificationStore(pool *pgxpool.Pool) *VerificationStore {
	return &VerificationStore{pool: pool}
}

var _ domain.VerificationStore = (*VerificationStore)(nil)

// Upsert — записать/перевыпустить код пользователя (одна запись на user_id).
func (r *VerificationStore) Upsert(ctx context.Context, userID int64, code, token string, expiresAt, sentAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO email_verifications (user_id, code, token, attempts, expires_at, last_sent_at)
		VALUES ($1, $2, $3, 0, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		   SET code = EXCLUDED.code, token = EXCLUDED.token, attempts = 0,
		       expires_at = EXCLUDED.expires_at, last_sent_at = EXCLUDED.last_sent_at`,
		userID, code, token, expiresAt, sentAt)
	return err
}

func scanVerification(row pgx.Row) (*domain.Verification, error) {
	var v domain.Verification
	err := row.Scan(&v.UserID, &v.Code, &v.Token, &v.Attempts, &v.ExpiresAt, &v.LastSentAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

const verificationCols = `user_id, code, token, attempts, expires_at, last_sent_at`

func (r *VerificationStore) GetByToken(ctx context.Context, token string) (*domain.Verification, error) {
	return scanVerification(r.pool.QueryRow(ctx,
		`SELECT `+verificationCols+` FROM email_verifications WHERE token = $1`, token))
}

func (r *VerificationStore) GetByUserID(ctx context.Context, userID int64) (*domain.Verification, error) {
	return scanVerification(r.pool.QueryRow(ctx,
		`SELECT `+verificationCols+` FROM email_verifications WHERE user_id = $1`, userID))
}

func (r *VerificationStore) IncAttempts(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_verifications SET attempts = attempts + 1 WHERE user_id = $1`, userID)
	return err
}

func (r *VerificationStore) Delete(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_verifications WHERE user_id = $1`, userID)
	return err
}
