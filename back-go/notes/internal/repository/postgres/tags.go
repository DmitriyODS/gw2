package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

func (r *Repo) ListTags(ctx context.Context, ownerID int64) ([]*domain.Tag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.owner_id, t.name, t.color, t.position, t.created_at, COUNT(ti.note_id)
		  FROM note_tags t
		  LEFT JOIN note_tag_items ti ON ti.tag_id = t.id
		 WHERE t.owner_id = $1
		 GROUP BY t.id
		 ORDER BY t.position, t.id`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Tag{}
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.OwnerID, &t.Name, &t.Color, &t.Position, &t.CreatedAt, &t.NotesCount); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

func (r *Repo) GetTag(ctx context.Context, id int64) (*domain.Tag, error) {
	var t domain.Tag
	err := r.pool.QueryRow(ctx, `
		SELECT id, owner_id, name, color, position, created_at FROM note_tags WHERE id = $1`, id).
		Scan(&t.ID, &t.OwnerID, &t.Name, &t.Color, &t.Position, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repo) CreateTag(ctx context.Context, t *domain.Tag) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO note_tags (owner_id, name, color, position) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		t.OwnerID, t.Name, t.Color, t.Position).Scan(&t.ID, &t.CreatedAt)
}

func (r *Repo) UpdateTag(ctx context.Context, id int64, name, color string) error {
	_, err := r.pool.Exec(ctx, `UPDATE note_tags SET name = $2, color = $3 WHERE id = $1`, id, name, color)
	return err
}

func (r *Repo) DeleteTag(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM note_tags WHERE id = $1`, id)
	return err
}

func (r *Repo) NextTagPosition(ctx context.Context, ownerID int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(position), 0) + 1 FROM note_tags WHERE owner_id = $1`, ownerID).Scan(&pos)
	return pos, err
}

func (r *Repo) OwnedTagIDs(ctx context.Context, ownerID int64, ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return []int64{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id FROM note_tags WHERE owner_id = $1 AND id = ANY($2::bigint[]) ORDER BY id`,
		ownerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
