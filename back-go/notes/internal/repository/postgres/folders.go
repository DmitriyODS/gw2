package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// folderCols — общий список колонок папки + число заметок прямо в ней.
const folderCols = `f.id, f.owner_id, f.parent_id, f.name, f.color, f.position,
	f.created_at, f.updated_at,
	COALESCE((SELECT count(*) FROM notes n WHERE n.folder_id = f.id), 0)`

func scanFolder(rows pgx.Rows) (*domain.Folder, error) {
	var f domain.Folder
	if err := rows.Scan(&f.ID, &f.OwnerID, &f.ParentID, &f.Name, &f.Color, &f.Position,
		&f.CreatedAt, &f.UpdatedAt, &f.NotesCount); err != nil {
		return nil, err
	}
	return &f, nil
}

// ListFolders — все папки владельца (плоско, клиент строит дерево). Флаг
// «расшарена мной» проставляется отдельно (SharedByMeFolderIDs).
func (r *Repo) ListFolders(ctx context.Context, ownerID int64) ([]*domain.Folder, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+folderCols+` FROM note_folders f WHERE f.owner_id = $1 ORDER BY f.position, f.name, f.id`, ownerID)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.markSharedFolders(ctx, out)
	return out, nil
}

// ListChildFolders — дочерние папки (любого владельца) для навигации по
// расшаренному поддереву. Доступ проверяет сервис.
func (r *Repo) ListChildFolders(ctx context.Context, parentID int64) ([]*domain.Folder, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+folderCols+` FROM note_folders f WHERE f.parent_id = $1 ORDER BY f.position, f.name, f.id`, parentID)
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

// ListSharedRootFolders — папки, расшаренные мне напрямую (пользователю или моей
// компании): «корни» раздела «Поделились со мной», с owner и my_access.
func (r *Repo) ListSharedRootFolders(ctx context.Context, userID int64, companyIDs []int64) ([]*domain.Folder, error) {
	rows, err := r.pool.Query(ctx, `
		WITH grants AS (
			SELECT folder_id, bool_or(can_edit) AS can_edit FROM (
				SELECT folder_id, can_edit FROM folder_user_shares WHERE user_id = $1
				UNION ALL
				SELECT folder_id, can_edit FROM folder_company_shares WHERE company_id = ANY($2::bigint[])
			) g GROUP BY folder_id
		)
		SELECT `+folderCols+`, u.fio, u.avatar_path, g.can_edit
		  FROM grants g
		  JOIN note_folders f ON f.id = g.folder_id
		  JOIN users u ON u.id = f.owner_id
		 WHERE f.owner_id <> $1
		 ORDER BY f.name, f.id`, userID, companyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Folder{}
	for rows.Next() {
		var (
			f       domain.Folder
			canEdit bool
		)
		if err := rows.Scan(&f.ID, &f.OwnerID, &f.ParentID, &f.Name, &f.Color, &f.Position,
			&f.CreatedAt, &f.UpdatedAt, &f.NotesCount, &f.OwnerName, &f.OwnerAvatar, &canEdit); err != nil {
			return nil, err
		}
		f.MyAccess = domain.AccessView
		if canEdit {
			f.MyAccess = domain.AccessEdit
		}
		out = append(out, &f)
	}
	return out, rows.Err()
}

func (r *Repo) GetFolder(ctx context.Context, id int64) (*domain.Folder, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+folderCols+` FROM note_folders f WHERE f.id = $1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	return scanFolder(rows)
}

func (r *Repo) CreateFolder(ctx context.Context, f *domain.Folder) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO note_folders (owner_id, parent_id, name, color, position)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		f.OwnerID, f.ParentID, f.Name, f.Color, f.Position).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

func (r *Repo) UpdateFolder(ctx context.Context, id int64, name, color string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE note_folders SET name = $2, color = $3, updated_at = now() WHERE id = $1`, id, name, color)
	return err
}

func (r *Repo) MoveFolder(ctx context.Context, id int64, parentID *int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE note_folders SET parent_id = $2, updated_at = now() WHERE id = $1`, id, parentID)
	return err
}

func (r *Repo) DeleteFolder(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM note_folders WHERE id = $1`, id)
	return err
}

func (r *Repo) NextFolderPosition(ctx context.Context, ownerID int64, parentID *int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(position), 0) + 1 FROM note_folders
		 WHERE owner_id = $1 AND parent_id IS NOT DISTINCT FROM $2`, ownerID, parentID).Scan(&pos)
	return pos, err
}

// IsDescendant — folderID лежит в поддереве maybeAncestor (равенство — true):
// используется для защиты от циклов при переносе папки.
func (r *Repo) IsDescendant(ctx context.Context, folderID, maybeAncestor int64) (bool, error) {
	var found bool
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE sub AS (
			SELECT id FROM note_folders WHERE id = $2
			UNION ALL
			SELECT f.id FROM note_folders f JOIN sub ON f.parent_id = sub.id
		)
		SELECT EXISTS(SELECT 1 FROM sub WHERE id = $1)`, folderID, maybeAncestor).Scan(&found)
	return found, err
}

// ReparentChildren — перевесить прямых детей папки (подпапки и заметки) на
// newParent (nil — в корень). Вызывается при удалении папки.
func (r *Repo) ReparentChildren(ctx context.Context, folderID int64, newParent *int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`UPDATE note_folders SET parent_id = $2 WHERE parent_id = $1`, folderID, newParent); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE notes SET folder_id = $2 WHERE folder_id = $1`, folderID, newParent); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CopyFolderTree — глубокая копия поддерева папки со всеми заметками владельца
// (и их тегами). Возвращает id корневой копии. Всё в одной транзакции.
func (r *Repo) CopyFolderTree(ctx context.Context, ownerID, folderID int64, newParent *int64) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rootID, err := copyFolderRec(ctx, tx, ownerID, folderID, newParent)
	if err != nil {
		return 0, err
	}
	return rootID, tx.Commit(ctx)
}

func copyFolderRec(ctx context.Context, tx pgx.Tx, ownerID, srcID int64, parentID *int64) (int64, error) {
	var (
		newID       int64
		name, color string
		pos         int
	)
	if err := tx.QueryRow(ctx, `
		INSERT INTO note_folders (owner_id, parent_id, name, color, position)
		SELECT $1, $2, name, color, position FROM note_folders WHERE id = $3
		RETURNING id, name, color, position`, ownerID, parentID, srcID).
		Scan(&newID, &name, &color, &pos); err != nil {
		return 0, err
	}
	// Копируем заметки этой папки вместе с тегами.
	noteRows, err := tx.Query(ctx, `SELECT id FROM notes WHERE folder_id = $1 AND owner_id = $2`, srcID, ownerID)
	if err != nil {
		return 0, err
	}
	var srcNoteIDs []int64
	for noteRows.Next() {
		var id int64
		if err := noteRows.Scan(&id); err != nil {
			noteRows.Close()
			return 0, err
		}
		srcNoteIDs = append(srcNoteIDs, id)
	}
	noteRows.Close()
	if err := noteRows.Err(); err != nil {
		return 0, err
	}
	for _, srcNote := range srcNoteIDs {
		var newNote int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO notes (owner_id, folder_id, title, color, doc, text_content)
			SELECT owner_id, $2, title, color, doc, text_content FROM notes WHERE id = $1
			RETURNING id`, srcNote, newID).Scan(&newNote); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO note_tag_items (note_id, tag_id)
			SELECT $1, tag_id FROM note_tag_items WHERE note_id = $2`, newNote, srcNote); err != nil {
			return 0, err
		}
	}
	// Рекурсивно копируем подпапки.
	childRows, err := tx.Query(ctx, `SELECT id FROM note_folders WHERE parent_id = $1`, srcID)
	if err != nil {
		return 0, err
	}
	var childIDs []int64
	for childRows.Next() {
		var id int64
		if err := childRows.Scan(&id); err != nil {
			childRows.Close()
			return 0, err
		}
		childIDs = append(childIDs, id)
	}
	childRows.Close()
	if err := childRows.Err(); err != nil {
		return 0, err
	}
	for _, child := range childIDs {
		if _, err := copyFolderRec(ctx, tx, ownerID, child, &newID); err != nil {
			return 0, err
		}
	}
	return newID, nil
}

// markSharedFolders — проставить SharedByMe для папок, у которых есть шары.
func (r *Repo) markSharedFolders(ctx context.Context, folders []*domain.Folder) {
	if len(folders) == 0 {
		return
	}
	ids := make([]int64, len(folders))
	for i, f := range folders {
		ids[i] = f.ID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT folder_id FROM folder_user_shares WHERE folder_id = ANY($1::bigint[])
		UNION
		SELECT folder_id FROM folder_company_shares WHERE folder_id = ANY($1::bigint[])`, ids)
	if err != nil {
		return
	}
	defer rows.Close()
	shared := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			shared[id] = true
		}
	}
	for _, f := range folders {
		if shared[f.ID] {
			f.SharedByMe = true
		}
	}
}
