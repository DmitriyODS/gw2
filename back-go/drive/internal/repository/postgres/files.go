package postgres

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
)

const fileCols = `id, owner_id, folder_id, name, storage_key, mime, size_bytes,
	starred, deleted_at, created_at, updated_at`

func scanFile(row pgx.Row) (*domain.File, error) {
	var f domain.File
	err := row.Scan(&f.ID, &f.OwnerID, &f.FolderID, &f.Name, &f.Key, &f.Mime,
		&f.Size, &f.Starred, &f.DeletedAt, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

// ListFiles — содержимое папки либо одна из сквозных выборок.
//
// Поиск ГЛОБАЛЬНЫЙ (папка не ограничивает выдачу): искомое чаще лежит не там,
// где сейчас открыто — тот же выбор, что в заметках. Регистр не важен: ILIKE
// плюс триграммный индекс по имени.
func (r *Repo) ListFiles(ctx domain.Ctx, f domain.ListFilter) ([]*domain.File, error) {
	var (
		where = []string{"owner_id = $1"}
		args  = []any{f.OwnerID}
		order = "name"
	)
	add := func(cond string, arg any) {
		args = append(args, arg)
		where = append(where, strings.Replace(cond, "?", "$"+strconv.Itoa(len(args)), 1))
	}

	switch {
	case f.Trash:
		where = append(where, "deleted_at IS NOT NULL")
		order = "deleted_at DESC"
	default:
		where = append(where, "deleted_at IS NULL")
	}

	switch {
	case f.Search != "":
		add("name ILIKE ?", "%"+f.Search+"%")
	case f.Starred:
		where = append(where, "starred")
	case f.Recent:
		order = "updated_at DESC"
	case !f.Trash:
		add("folder_id IS NOT DISTINCT FROM ?", nullableID(f.FolderID))
	}

	q := `SELECT ` + fileCols + ` FROM drive_files WHERE ` +
		strings.Join(where, " AND ") + ` ORDER BY ` + order
	if f.Recent {
		q += " LIMIT 50"
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.File{}
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, file)
	}
	return out, rows.Err()
}

func (r *Repo) GetFile(ctx domain.Ctx, id int64) (*domain.File, error) {
	return scanFile(r.pool.QueryRow(ctx, `SELECT `+fileCols+` FROM drive_files WHERE id = $1`, id))
}

func (r *Repo) CreateFile(ctx domain.Ctx, f *domain.File) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO drive_files (owner_id, folder_id, name, storage_key, mime, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		f.OwnerID, nullableID(f.FolderID), f.Name, f.Key, f.Mime, f.Size,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

func (r *Repo) RenameFile(ctx domain.Ctx, id int64, name string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE drive_files SET name = $2, updated_at = now() WHERE id = $1`, id, name)
	return err
}

func (r *Repo) MoveFile(ctx domain.Ctx, id int64, folderID *int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE drive_files SET folder_id = $2, updated_at = now() WHERE id = $1`,
		id, nullableID(folderID))
	return err
}

func (r *Repo) SetFileStarred(ctx domain.Ctx, id int64, starred bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE drive_files SET starred = $2, updated_at = now() WHERE id = $1`, id, starred)
	return err
}

func (r *Repo) SetFilesDeleted(ctx domain.Ctx, ids []int64, deleted bool) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE drive_files
		   SET deleted_at = CASE WHEN $2::bool THEN now() ELSE NULL END,
		       updated_at = now()
		 WHERE id = ANY($1)`, ids, deleted)
	return err
}

// SetFilesDeletedByFolders — содержимое папок уезжает в корзину вместе с ними.
func (r *Repo) SetFilesDeletedByFolders(ctx domain.Ctx, folderIDs []int64, deleted bool) error {
	if len(folderIDs) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE drive_files
		   SET deleted_at = CASE WHEN $2::bool THEN now() ELSE NULL END,
		       updated_at = now()
		 WHERE folder_id = ANY($1)`, folderIDs, deleted)
	return err
}

func (r *Repo) FilesOfFolders(ctx domain.Ctx, folderIDs []int64) ([]*domain.File, error) {
	if len(folderIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+fileCols+` FROM drive_files WHERE folder_id = ANY($1)`, folderIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.File{}
	for rows.Next() {
		f, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repo) DeleteFiles(ctx domain.Ctx, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM drive_files WHERE id = ANY($1)`, ids)
	return err
}

// ExpiredTrash — что пролежало в корзине дольше срока: файлы (их объекты ещё
// нужно удалить из хранилища) и id папок.
func (r *Repo) ExpiredTrash(ctx domain.Ctx) ([]*domain.File, []int64, error) {
	edge := time.Now().AddDate(0, 0, -domain.TrashKeepDays)

	rows, err := r.pool.Query(ctx,
		`SELECT `+fileCols+` FROM drive_files WHERE deleted_at < $1`, edge)
	if err != nil {
		return nil, nil, err
	}
	files := []*domain.File{}
	for rows.Next() {
		f, err := scanFile(rows)
		if err != nil {
			rows.Close()
			return nil, nil, err
		}
		files = append(files, f)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	frows, err := r.pool.Query(ctx,
		`SELECT id FROM drive_folders WHERE deleted_at < $1`, edge)
	if err != nil {
		return nil, nil, err
	}
	defer frows.Close()
	folders := []int64{}
	for frows.Next() {
		var id int64
		if err := frows.Scan(&id); err != nil {
			return nil, nil, err
		}
		folders = append(folders, id)
	}
	return files, folders, frows.Err()
}
