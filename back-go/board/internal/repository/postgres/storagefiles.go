package postgres

import (
	"encoding/json"

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"
)

/* Сторона досок в контракте владельца файлов (раздел «Хранилище»).

   Картинки лежат внутри сцены, миниатюра — отдельной колонкой; отдельной
   таблицы файлов нет. Доски личные: скоуп — владелец. */

// BoardScenesOf — сцены владельца одним запросом (списочный ListBoards отдаёт
// плитки без сцены).
func (r *Repo) BoardScenesOf(ctx domain.Ctx, ownerID int64) ([]*domain.Board, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, scene, preview_path, created_at
		  FROM boards WHERE owner_id = $1`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Board{}
	for rows.Next() {
		var b domain.Board
		var scene []byte
		if err := rows.Scan(&b.ID, &b.Title, &scene, &b.PreviewPath, &b.CreatedAt); err != nil {
			return nil, err
		}
		b.Scene = json.RawMessage(scene)
		b.OwnerID = ownerID
		out = append(out, &b)
	}
	return out, rows.Err()
}

// UpdateBoardScene — записать вычищенную сцену, не трогая остального.
func (r *Repo) UpdateBoardScene(ctx domain.Ctx, id int64, scene json.RawMessage, textContent string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE boards SET scene = $2, text_content = $3, updated_at = now() WHERE id = $1`,
		id, []byte(scene), textContent)
	return err
}
