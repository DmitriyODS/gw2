package postgres

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// vecToStr — pgvector принимает строку "[0.1,0.2,...]".
func vecToStr(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

func (r *Repo) UpsertNoteEmbedding(ctx context.Context, noteID, ownerID int64, vector []float32, model string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO note_embeddings (note_id, owner_id, embedding, model, updated_at)
		VALUES ($1, $2, CAST($3 AS vector), $4, $5)
		ON CONFLICT (note_id) DO UPDATE
		   SET owner_id = EXCLUDED.owner_id, embedding = EXCLUDED.embedding,
		       model = EXCLUDED.model, updated_at = EXCLUDED.updated_at`,
		noteID, ownerID, vecToStr(vector), model, time.Now().UTC())
	return err
}

// SearchNoteEmbeddings — id заметок владельца по близости к вектору (той же
// модели: после смены модели старые векторы в выдачу не попадают).
func (r *Repo) SearchNoteEmbeddings(ctx context.Context, ownerID int64, vector []float32, model string, archived bool, limit int) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT e.note_id
		  FROM note_embeddings e
		  JOIN notes n ON n.id = e.note_id
		 WHERE e.owner_id = $2 AND e.model = $3 AND n.archived = $4
		 ORDER BY e.embedding <=> CAST($1 AS vector)
		 LIMIT $5`, vecToStr(vector), ownerID, model, archived, limit)
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

// ListNotesByIDs — плитки владельца по id, порядок сохраняется как в ids.
func (r *Repo) ListNotesByIDs(ctx context.Context, ownerID int64, ids []int64, archived bool) ([]*domain.Note, error) {
	if len(ids) == 0 {
		return []*domain.Note{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.owner_id, n.title, n.color, n.archived, n.folder_id, n.pinned_at,
		       left(n.text_content, 300), n.created_at, n.updated_at, `+noteTags+`
		  FROM notes n
		 WHERE n.owner_id = $1 AND n.archived = $3 AND n.id = ANY($2::bigint[])`,
		ownerID, ids, archived)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[int64]*domain.Note{}
	for rows.Next() {
		var n domain.Note
		if err := rows.Scan(&n.ID, &n.OwnerID, &n.Title, &n.Color, &n.Archived, &n.FolderID, &n.PinnedAt,
			&n.Excerpt, &n.CreatedAt, &n.UpdatedAt, &n.TagIDs); err != nil {
			return nil, err
		}
		byID[n.ID] = &n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]*domain.Note, 0, len(ids))
	for _, id := range ids {
		if n := byID[id]; n != nil {
			out = append(out, n)
		}
	}
	return out, nil
}
