package postgres

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.NoteRepository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// noteGroups — агрегат групп заметки для выборок (пустой массив, если связей нет).
const noteGroups = `COALESCE((SELECT array_agg(group_id ORDER BY group_id)
	FROM note_group_items WHERE note_id = n.id), '{}')`

func (r *Repo) ListNotes(ctx context.Context, f domain.NoteListFilter) ([]*domain.Note, error) {
	q := `SELECT n.id, n.owner_id, n.title, n.color, n.archived, left(n.text_content, 300),
	             n.created_at, n.updated_at, ` + noteGroups + `
	        FROM notes n
	       WHERE n.owner_id = $1 AND n.archived = $2`
	args := []any{f.OwnerID, f.Archived}
	if f.GroupID > 0 {
		args = append(args, f.GroupID)
		q += ` AND EXISTS (SELECT 1 FROM note_group_items gi
			WHERE gi.note_id = n.id AND gi.group_id = $` + strconv.Itoa(len(args)) + `)`
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		q += ` AND (n.title || ' ' || n.text_content) ILIKE $` + strconv.Itoa(len(args))
	}
	q += ` ORDER BY n.updated_at DESC, n.id DESC`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Note{}
	for rows.Next() {
		var n domain.Note
		if err := rows.Scan(&n.ID, &n.OwnerID, &n.Title, &n.Color, &n.Archived, &n.Excerpt,
			&n.CreatedAt, &n.UpdatedAt, &n.GroupIDs); err != nil {
			return nil, err
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (r *Repo) GetNote(ctx context.Context, id int64) (*domain.Note, error) {
	var n domain.Note
	err := r.pool.QueryRow(ctx, `
		SELECT n.id, n.owner_id, n.title, n.color, n.archived, n.doc, n.text_content,
		       n.created_at, n.updated_at, `+noteGroups+`
		  FROM notes n WHERE n.id = $1`, id).
		Scan(&n.ID, &n.OwnerID, &n.Title, &n.Color, &n.Archived, &n.Doc, &n.TextContent,
			&n.CreatedAt, &n.UpdatedAt, &n.GroupIDs)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n.Excerpt = n.TextContent
	if r := []rune(n.Excerpt); len(r) > 300 {
		n.Excerpt = string(r[:300])
	}
	return &n, nil
}

func (r *Repo) CreateNote(ctx context.Context, n *domain.Note) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO notes (owner_id, title, color, doc, text_content)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		n.OwnerID, n.Title, n.Color, n.Doc, n.TextContent).
		Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
}

func (r *Repo) UpdateNote(ctx context.Context, n *domain.Note) error {
	return r.pool.QueryRow(ctx, `
		UPDATE notes SET title = $2, color = $3, archived = $4, doc = $5, text_content = $6, updated_at = now()
		 WHERE id = $1 RETURNING updated_at`,
		n.ID, n.Title, n.Color, n.Archived, n.Doc, n.TextContent).Scan(&n.UpdatedAt)
}

func (r *Repo) DeleteNote(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM notes WHERE id = $1`, id)
	return err
}

// SetNoteGroups — полная замена связей заметки с группами (в транзакции).
func (r *Repo) SetNoteGroups(ctx context.Context, noteID int64, groupIDs []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM note_group_items WHERE note_id = $1`, noteID); err != nil {
		return err
	}
	if len(groupIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO note_group_items (note_id, group_id)
			SELECT $1, unnest($2::bigint[]) ON CONFLICT DO NOTHING`, noteID, groupIDs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
