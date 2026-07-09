package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

func (r *Repo) ListGroups(ctx context.Context, ownerID int64) ([]*domain.Group, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT g.id, g.owner_id, g.name, g.position, g.created_at, COUNT(gi.note_id)
		  FROM note_groups g
		  LEFT JOIN note_group_items gi ON gi.group_id = g.id
		 WHERE g.owner_id = $1
		 GROUP BY g.id
		 ORDER BY g.position, g.id`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Group{}
	for rows.Next() {
		var g domain.Group
		if err := rows.Scan(&g.ID, &g.OwnerID, &g.Name, &g.Position, &g.CreatedAt, &g.NotesCount); err != nil {
			return nil, err
		}
		out = append(out, &g)
	}
	return out, rows.Err()
}

func (r *Repo) GetGroup(ctx context.Context, id int64) (*domain.Group, error) {
	var g domain.Group
	err := r.pool.QueryRow(ctx, `
		SELECT id, owner_id, name, position, created_at FROM note_groups WHERE id = $1`, id).
		Scan(&g.ID, &g.OwnerID, &g.Name, &g.Position, &g.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *Repo) CreateGroup(ctx context.Context, g *domain.Group) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO note_groups (owner_id, name, position) VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		g.OwnerID, g.Name, g.Position).Scan(&g.ID, &g.CreatedAt)
}

func (r *Repo) UpdateGroup(ctx context.Context, id int64, name string) error {
	_, err := r.pool.Exec(ctx, `UPDATE note_groups SET name = $2 WHERE id = $1`, id, name)
	return err
}

func (r *Repo) DeleteGroup(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM note_groups WHERE id = $1`, id)
	return err
}

func (r *Repo) NextGroupPosition(ctx context.Context, ownerID int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(position), 0) + 1 FROM note_groups WHERE owner_id = $1`, ownerID).Scan(&pos)
	return pos, err
}

func (r *Repo) OwnedGroupIDs(ctx context.Context, ownerID int64, ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return []int64{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id FROM note_groups WHERE owner_id = $1 AND id = ANY($2::bigint[]) ORDER BY id`,
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
