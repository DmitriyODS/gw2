package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

func (r *Repo) ListShares(ctx context.Context, noteID int64) ([]*domain.Share, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, note_id, code, access, created_at
		  FROM note_shares WHERE note_id = $1 ORDER BY id`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Share{}
	for rows.Next() {
		var s domain.Share
		if err := rows.Scan(&s.ID, &s.NoteID, &s.Code, &s.Access, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *Repo) CreateShare(ctx context.Context, s *domain.Share) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO note_shares (note_id, code, access) VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		s.NoteID, s.Code, s.Access).Scan(&s.ID, &s.CreatedAt)
}

func (r *Repo) GetShareByCode(ctx context.Context, code string) (*domain.Share, error) {
	var s domain.Share
	err := r.pool.QueryRow(ctx, `
		SELECT id, note_id, code, access, created_at FROM note_shares WHERE code = $1`, code).
		Scan(&s.ID, &s.NoteID, &s.Code, &s.Access, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) DeleteShare(ctx context.Context, id, noteID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM note_shares WHERE id = $1 AND note_id = $2`, id, noteID)
	return err
}
