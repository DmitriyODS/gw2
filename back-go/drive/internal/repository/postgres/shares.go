package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
)

/* Шаринг диска — тот же приём, что в заметках и досках.

   Ключевой инвариант: доступ, выданный на ПАПКУ, действует на всё её
   поддерево. Поэтому эффективный доступ считает СЕРВЕР рекурсивным подъёмом
   по parent_id, а не клиент: иначе пришлось бы тянуть всё дерево на фронт и
   повторять там правила. */

func (r *Repo) CreateShare(ctx domain.Ctx, s *domain.Share) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO drive_shares (file_id, folder_id, code, created_by)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		nullableID(s.FileID), nullableID(s.FolderID), s.Code, s.CreatedBy,
	).Scan(&s.ID, &s.CreatedAt)
}

func (r *Repo) GetShareByCode(ctx domain.Ctx, code string) (*domain.Share, error) {
	var s domain.Share
	err := r.pool.QueryRow(ctx, `
		SELECT id, file_id, folder_id, code, created_at
		  FROM drive_shares WHERE code = $1`, code,
	).Scan(&s.ID, &s.FileID, &s.FolderID, &s.Code, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repo) ListShares(ctx domain.Ctx, fileID, folderID *int64) ([]*domain.Share, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, file_id, folder_id, code, created_at FROM drive_shares
		 WHERE file_id IS NOT DISTINCT FROM $1 AND folder_id IS NOT DISTINCT FROM $2
		 ORDER BY id`, nullableID(fileID), nullableID(folderID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Share{}
	for rows.Next() {
		var s domain.Share
		if err := rows.Scan(&s.ID, &s.FileID, &s.FolderID, &s.Code, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

// DeleteShare — снять публичную ссылку. Владелец цели проверяется здесь же:
// чужую ссылку не убрать, зная её id.
func (r *Repo) DeleteShare(ctx domain.Ctx, id, ownerID int64) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM drive_shares s
		 WHERE s.id = $1 AND (
		   EXISTS (SELECT 1 FROM drive_files f WHERE f.id = s.file_id AND f.owner_id = $2)
		   OR EXISTS (SELECT 1 FROM drive_folders d WHERE d.id = s.folder_id AND d.owner_id = $2))`,
		id, ownerID)
	return err
}

// UpsertUserShare — выдать доступ; повторная выдача тому же адресату меняет
// права, а не плодит строки (частичные UNIQUE-индексы в миграции).
func (r *Repo) UpsertUserShare(ctx domain.Ctx, s *domain.UserShare) error {
	target, who := "file_id", "user_id"
	if s.FileID == nil {
		target = "folder_id"
	}
	if s.UserID == nil {
		who = "company_id"
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO drive_user_shares (file_id, folder_id, user_id, company_id, can_edit)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (`+target+`, `+who+`) WHERE `+target+` IS NOT NULL AND `+who+` IS NOT NULL
		DO UPDATE SET can_edit = EXCLUDED.can_edit
		RETURNING id, created_at`,
		nullableID(s.FileID), nullableID(s.FolderID),
		nullableID(s.UserID), nullableID(s.CompanyID), s.CanEdit,
	).Scan(&s.ID, &s.CreatedAt)
}

func (r *Repo) ListUserShares(ctx domain.Ctx, fileID, folderID *int64) ([]*domain.UserShare, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.file_id, s.folder_id, s.user_id, s.company_id, s.can_edit, s.created_at,
		       COALESCE(u.fio, ''), COALESCE(c.name, '')
		  FROM drive_user_shares s
		  LEFT JOIN users u ON u.id = s.user_id
		  LEFT JOIN companies c ON c.id = s.company_id
		 WHERE s.file_id IS NOT DISTINCT FROM $1 AND s.folder_id IS NOT DISTINCT FROM $2
		 ORDER BY s.id`, nullableID(fileID), nullableID(folderID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.UserShare{}
	for rows.Next() {
		var s domain.UserShare
		if err := rows.Scan(&s.ID, &s.FileID, &s.FolderID, &s.UserID, &s.CompanyID,
			&s.CanEdit, &s.CreatedAt, &s.UserName, &s.CompanyName); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *Repo) DeleteUserShare(ctx domain.Ctx, id, ownerID int64) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM drive_user_shares s
		 WHERE s.id = $1 AND (
		   EXISTS (SELECT 1 FROM drive_files f WHERE f.id = s.file_id AND f.owner_id = $2)
		   OR EXISTS (SELECT 1 FROM drive_folders d WHERE d.id = s.folder_id AND d.owner_id = $2))`,
		id, ownerID)
	return err
}

// FileAccess — эффективный доступ к файлу: прямая выдача либо доступ,
// унаследованный от любой папки-предка. Возвращает лучший из найденных
// уровней; пустая строка — доступа нет.
func (r *Repo) FileAccess(ctx domain.Ctx, fileID, userID int64, companyIDs []int64) (string, error) {
	var owner int64
	var folderID *int64
	err := r.pool.QueryRow(ctx,
		`SELECT owner_id, folder_id FROM drive_files WHERE id = $1`, fileID).Scan(&owner, &folderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if owner == userID {
		return domain.AccessOwner, nil
	}

	var canEdit *bool
	err = r.pool.QueryRow(ctx, `
		SELECT bool_or(can_edit) FROM drive_user_shares
		 WHERE file_id = $1 AND (user_id = $2 OR company_id = ANY($3))`,
		fileID, userID, companyIDs).Scan(&canEdit)
	if err != nil {
		return "", err
	}
	best := accessFrom(canEdit)
	if best == domain.AccessEdit || folderID == nil {
		return best, nil
	}
	// Не нашли прямой выдачи (или нашли только просмотр) — поднимаемся по папкам.
	inherited, err := r.FolderAccess(ctx, *folderID, userID, companyIDs)
	if err != nil {
		return "", err
	}
	if domain.AccessAtLeast(inherited, domain.AccessEdit) || best == "" {
		return inherited, nil
	}
	return best, nil
}

// FolderAccess — доступ к папке с подъёмом по дереву: выдача на предка
// действует на всех потомков.
func (r *Repo) FolderAccess(ctx domain.Ctx, folderID, userID int64, companyIDs []int64) (string, error) {
	var owner int64
	if err := r.pool.QueryRow(ctx,
		`SELECT owner_id FROM drive_folders WHERE id = $1`, folderID).Scan(&owner); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if owner == userID {
		return domain.AccessOwner, nil
	}

	var canEdit *bool
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE up AS (
			SELECT id, parent_id FROM drive_folders WHERE id = $1
			UNION ALL
			SELECT f.id, f.parent_id FROM drive_folders f JOIN up ON f.id = up.parent_id
		)
		SELECT bool_or(s.can_edit)
		  FROM drive_user_shares s
		  JOIN up ON up.id = s.folder_id
		 WHERE s.user_id = $2 OR s.company_id = ANY($3)`,
		folderID, userID, companyIDs).Scan(&canEdit)
	if err != nil {
		return "", err
	}
	return accessFrom(canEdit), nil
}

// accessFrom — NULL (выдачи нет) → пусто, true → правка, false → просмотр.
func accessFrom(canEdit *bool) string {
	switch {
	case canEdit == nil:
		return ""
	case *canEdit:
		return domain.AccessEdit
	default:
		return domain.AccessView
	}
}

// SharedWithMe — чужое, доступное мне лично или моей компании. Папку показываем
// только верхнюю: её содержимое человек откроет обычной навигацией.
func (r *Repo) SharedWithMe(ctx domain.Ctx, userID int64, companyIDs []int64) ([]*domain.Folder, []*domain.File, error) {
	frows, err := r.pool.Query(ctx, `
		SELECT f.id, f.owner_id, f.parent_id, f.name, f.color, f.deleted_at,
		       f.created_at, f.updated_at, s.can_edit, COALESCE(u.fio, '')
		  FROM drive_user_shares s
		  JOIN drive_folders f ON f.id = s.folder_id
		  LEFT JOIN users u ON u.id = f.owner_id
		 WHERE (s.user_id = $1 OR s.company_id = ANY($2))
		   AND f.owner_id <> $1 AND f.deleted_at IS NULL
		 ORDER BY f.name`, userID, companyIDs)
	if err != nil {
		return nil, nil, err
	}
	folders := []*domain.Folder{}
	for frows.Next() {
		var f domain.Folder
		var canEdit bool
		if err := frows.Scan(&f.ID, &f.OwnerID, &f.ParentID, &f.Name, &f.Color, &f.DeletedAt,
			&f.CreatedAt, &f.UpdatedAt, &canEdit, &f.OwnerName); err != nil {
			frows.Close()
			return nil, nil, err
		}
		f.MyAccess = accessFrom(&canEdit)
		folders = append(folders, &f)
	}
	frows.Close()
	if err := frows.Err(); err != nil {
		return nil, nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT f.id, f.owner_id, f.folder_id, f.name, f.storage_key, f.mime, f.size_bytes,
		       f.starred, f.deleted_at, f.created_at, f.updated_at, s.can_edit, COALESCE(u.fio, '')
		  FROM drive_user_shares s
		  JOIN drive_files f ON f.id = s.file_id
		  LEFT JOIN users u ON u.id = f.owner_id
		 WHERE (s.user_id = $1 OR s.company_id = ANY($2))
		   AND f.owner_id <> $1 AND f.deleted_at IS NULL
		 ORDER BY f.name`, userID, companyIDs)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	files := []*domain.File{}
	for rows.Next() {
		var f domain.File
		var canEdit bool
		if err := rows.Scan(&f.ID, &f.OwnerID, &f.FolderID, &f.Name, &f.Key, &f.Mime, &f.Size,
			&f.Starred, &f.DeletedAt, &f.CreatedAt, &f.UpdatedAt, &canEdit, &f.OwnerName); err != nil {
			return nil, nil, err
		}
		f.MyAccess = accessFrom(&canEdit)
		files = append(files, &f)
	}
	return folders, files, rows.Err()
}

// SharedByMe — значок «открыт доступ» на плитках: одним запросом на список,
// а не по запросу на элемент.
func (r *Repo) SharedByMe(ctx domain.Ctx, fileIDs, folderIDs []int64) (map[int64]bool, map[int64]bool, error) {
	files, folders := map[int64]bool{}, map[int64]bool{}
	if len(fileIDs) == 0 && len(folderIDs) == 0 {
		return files, folders, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT file_id, folder_id FROM drive_shares
		 WHERE file_id = ANY($1) OR folder_id = ANY($2)
		UNION
		SELECT file_id, folder_id FROM drive_user_shares
		 WHERE file_id = ANY($1) OR folder_id = ANY($2)`, fileIDs, folderIDs)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var fileID, folderID *int64
		if err := rows.Scan(&fileID, &folderID); err != nil {
			return nil, nil, err
		}
		if fileID != nil {
			files[*fileID] = true
		}
		if folderID != nil {
			folders[*folderID] = true
		}
	}
	return files, folders, rows.Err()
}
