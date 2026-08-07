package postgres

import (
	"encoding/json"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

/* Сторона заметок в контракте владельца файлов (раздел «Хранилище»).

   Картинки лежат ссылками ВНУТРИ документа TipTap, отдельной таблицы файлов
   нет — поэтому и список, и удаление работают по самим документам. Заметки
   личные: скоуп — владелец. */

// NoteDocsOf — документы владельца одним запросом (id, название, тело).
// Списочный ListNotes отдаёт плитки без doc, а здесь нужен именно он.
func (r *Repo) NoteDocsOf(ctx domain.Ctx, ownerID int64) ([]*domain.Note, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, doc, created_at FROM notes WHERE owner_id = $1`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Note{}
	for rows.Next() {
		var n domain.Note
		var doc []byte
		if err := rows.Scan(&n.ID, &n.Title, &doc, &n.CreatedAt); err != nil {
			return nil, err
		}
		n.Doc = json.RawMessage(doc)
		n.OwnerID = ownerID
		out = append(out, &n)
	}
	return out, rows.Err()
}

// UpdateNoteDoc — записать вычищенный документ, не трогая остального.
func (r *Repo) UpdateNoteDoc(ctx domain.Ctx, id int64, doc json.RawMessage, textContent string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notes SET doc = $2, text_content = $3, updated_at = now() WHERE id = $1`,
		id, []byte(doc), textContent)
	return err
}
