package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
)

const folderCols = `id, owner_id, parent_id, name, color, deleted_at, created_at, updated_at`

func scanFolder(row pgx.Row) (*domain.Folder, error) {
	var f domain.Folder
	err := row.Scan(&f.ID, &f.OwnerID, &f.ParentID, &f.Name, &f.Color,
		&f.DeletedAt, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

// ListFolders — подпапки одного уровня. trash разводит два режима: обычный
// показывает живые папки, корзина — только помеченные удалёнными, причём
// ПЛОСКИМ списком: в корзине важно, что лежит, а не как оно было вложено.
func (r *Repo) ListFolders(ctx domain.Ctx, ownerID int64, parentID *int64, trash bool) ([]*domain.Folder, error) {
	q := `SELECT ` + folderCols + ` FROM drive_folders
	       WHERE owner_id = $1 AND deleted_at IS NULL
	         AND parent_id IS NOT DISTINCT FROM $2
	       ORDER BY name`
	args := []any{ownerID, nullableID(parentID)}
	if trash {
		q = `SELECT ` + folderCols + ` FROM drive_folders
		      WHERE owner_id = $1 AND deleted_at IS NOT NULL
		      ORDER BY deleted_at DESC`
		args = args[:1]
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Folder{}
	for rows.Next() {
		f, err := scanFolder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// SearchFolders — папки по имени по всему диску (регистр не важен: ILIKE).
func (r *Repo) SearchFolders(ctx domain.Ctx, ownerID int64, query string) ([]*domain.Folder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+folderCols+` FROM drive_folders
		 WHERE owner_id = $1 AND deleted_at IS NULL AND name ILIKE $2
		 ORDER BY name LIMIT 50`, ownerID, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Folder{}
	for rows.Next() {
		f, err := scanFolder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repo) GetFolder(ctx domain.Ctx, id int64) (*domain.Folder, error) {
	return scanFolder(r.pool.QueryRow(ctx,
		`SELECT `+folderCols+` FROM drive_folders WHERE id = $1`, id))
}

func (r *Repo) CreateFolder(ctx domain.Ctx, f *domain.Folder) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO drive_folders (owner_id, parent_id, name, color)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`,
		f.OwnerID, nullableID(f.ParentID), f.Name, f.Color,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

func (r *Repo) RenameFolder(ctx domain.Ctx, id int64, name, color string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE drive_folders SET name = $2, color = $3, updated_at = now() WHERE id = $1`,
		id, name, color)
	return err
}

func (r *Repo) MoveFolder(ctx domain.Ctx, id int64, parentID *int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE drive_folders SET parent_id = $2, updated_at = now() WHERE id = $1`,
		id, nullableID(parentID))
	return err
}

// FolderPath — путь от корня до папки: рекурсивный подъём по parent_id одним
// запросом (обход в приложении дал бы запрос на уровень вложенности).
func (r *Repo) FolderPath(ctx domain.Ctx, id int64) ([]*domain.Folder, error) {
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE up AS (
			SELECT `+folderCols+`, 0 AS depth FROM drive_folders WHERE id = $1
			UNION ALL
			SELECT f.id, f.owner_id, f.parent_id, f.name, f.color, f.deleted_at,
			       f.created_at, f.updated_at, up.depth + 1
			  FROM drive_folders f JOIN up ON f.id = up.parent_id
		)
		SELECT `+folderCols+` FROM up ORDER BY depth DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Folder{}
	for rows.Next() {
		f, err := scanFolder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// FolderSubtree — id папки и всех её потомков: корзина, удаление и проверка
// циклов работают поддеревом целиком.
func (r *Repo) FolderSubtree(ctx domain.Ctx, id int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE down AS (
			SELECT id FROM drive_folders WHERE id = $1
			UNION ALL
			SELECT f.id FROM drive_folders f JOIN down ON f.parent_id = down.id
		)
		SELECT id FROM down`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var fid int64
		if err := rows.Scan(&fid); err != nil {
			return nil, err
		}
		out = append(out, fid)
	}
	return out, rows.Err()
}

func (r *Repo) SetFoldersDeleted(ctx domain.Ctx, ids []int64, deleted bool) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE drive_folders
		   SET deleted_at = CASE WHEN $2::bool THEN now() ELSE NULL END,
		       updated_at = now()
		 WHERE id = ANY($1)`, ids, deleted)
	return err
}

func (r *Repo) DeleteFolders(ctx domain.Ctx, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM drive_folders WHERE id = ANY($1)`, ids)
	return err
}
