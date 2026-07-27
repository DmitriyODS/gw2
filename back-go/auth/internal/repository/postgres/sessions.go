package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// SessionStore — реестр активных входов (user_sessions).
type SessionStore struct {
	pool *pgxpool.Pool
}

func NewSessionStore(pool *pgxpool.Pool) *SessionStore {
	return &SessionStore{pool: pool}
}

var _ domain.SessionStore = (*SessionStore)(nil)

const sessionCols = `id, user_id, created_at, last_seen_at, platform, client, device, ip, city`

func scanSession(row pgx.Row) (*domain.Session, error) {
	var s domain.Session
	err := row.Scan(&s.ID, &s.UserID, &s.CreatedAt, &s.LastSeenAt,
		&s.Platform, &s.Client, &s.Device, &s.IP, &s.City)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionStore) Create(ctx context.Context, s *domain.Session) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_sessions (user_id, platform, client, device, user_agent, ip, city)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		s.UserID, s.Platform, s.Client, s.Device, s.UserAgent, s.IP, s.City).Scan(&id)
	return id, err
}

func (r *SessionStore) Get(ctx context.Context, id int64) (*domain.Session, error) {
	return scanSession(r.pool.QueryRow(ctx,
		`SELECT `+sessionCols+` FROM user_sessions WHERE id = $1 AND revoked_at IS NULL`, id))
}

// Touch — двигает last_seen_at живой сессии. Условие revoked_at IS NULL здесь
// же: отозванный сеанс не воскрешается, а refresh по нему получает отказ.
func (r *SessionStore) Touch(ctx context.Context, id, userID int64) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE user_sessions SET last_seen_at = now()
		 WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *SessionStore) ListActive(ctx context.Context, userID int64) ([]*domain.Session, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+sessionCols+`
		  FROM user_sessions
		 WHERE user_id = $1 AND revoked_at IS NULL
		 ORDER BY last_seen_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Session
	for rows.Next() {
		var s domain.Session
		if err := rows.Scan(&s.ID, &s.UserID, &s.CreatedAt, &s.LastSeenAt,
			&s.Platform, &s.Client, &s.Device, &s.IP, &s.City); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *SessionStore) Revoke(ctx context.Context, id, userID int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE user_sessions SET revoked_at = now()
		 WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`, id, userID)
	return err
}

func (r *SessionStore) SetCity(ctx context.Context, id int64, city string) error {
	_, err := r.pool.Exec(ctx, `UPDATE user_sessions SET city = $2 WHERE id = $1`, id, city)
	return err
}
