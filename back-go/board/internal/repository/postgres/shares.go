package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"
)

// ── Публичные ссылки (view/edit по коду-capability) ──────────────────

func (r *Repo) ListShares(ctx context.Context, boardID int64) ([]*domain.Share, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, board_id, code, access, created_at
		  FROM board_shares WHERE board_id = $1 ORDER BY id`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Share{}
	for rows.Next() {
		var s domain.Share
		if err := rows.Scan(&s.ID, &s.BoardID, &s.Code, &s.Access, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *Repo) CreateShare(ctx context.Context, s *domain.Share) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO board_shares (board_id, code, access) VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		s.BoardID, s.Code, s.Access).Scan(&s.ID, &s.CreatedAt)
}

func (r *Repo) GetShareByCode(ctx context.Context, code string) (*domain.Share, error) {
	var s domain.Share
	err := r.pool.QueryRow(ctx, `
		SELECT id, board_id, code, access, created_at FROM board_shares WHERE code = $1`, code).
		Scan(&s.ID, &s.BoardID, &s.Code, &s.Access, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) DeleteShare(ctx context.Context, id, boardID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM board_shares WHERE id = $1 AND board_id = $2`, id, boardID)
	return err
}

// ── Адресный шаринг досок (пользователь и компания) ────────────────

// ListBoardMembers — адресаты доски: пользователи (JOIN users) и компании.
func (r *Repo) ListBoardMembers(ctx context.Context, boardID int64) ([]*domain.Member, error) {
	out := []*domain.Member{}
	rows, err := r.pool.Query(ctx, `
		SELECT s.user_id, u.fio, u.avatar_path, s.can_edit, s.created_at
		  FROM board_user_shares s JOIN users u ON u.id = s.user_id
		 WHERE s.board_id = $1 ORDER BY u.fio, s.user_id`, boardID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		m := &domain.Member{Target: domain.TargetUser}
		if err := rows.Scan(&m.UserID, &m.FIO, &m.AvatarPath, &m.CanEdit, &m.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	crows, err := r.pool.Query(ctx, `
		SELECT company_id, company_name, can_edit, created_at
		  FROM board_company_shares WHERE board_id = $1 ORDER BY company_name, company_id`, boardID)
	if err != nil {
		return nil, err
	}
	defer crows.Close()
	for crows.Next() {
		m := &domain.Member{Target: domain.TargetCompany}
		if err := crows.Scan(&m.CompanyID, &m.CompanyName, &m.CanEdit, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, crows.Err()
}

func (r *Repo) UpsertBoardUserShare(ctx context.Context, boardID, userID int64, canEdit bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO board_user_shares (board_id, user_id, can_edit) VALUES ($1, $2, $3)
		ON CONFLICT (board_id, user_id) DO UPDATE SET can_edit = EXCLUDED.can_edit`, boardID, userID, canEdit)
	return err
}

func (r *Repo) DeleteBoardUserShare(ctx context.Context, boardID, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM board_user_shares WHERE board_id = $1 AND user_id = $2`, boardID, userID)
	return err
}

func (r *Repo) UpsertBoardCompanyShare(ctx context.Context, boardID, companyID int64, name string, canEdit bool, by int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO board_company_shares (board_id, company_id, company_name, can_edit, shared_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (board_id, company_id)
		DO UPDATE SET can_edit = EXCLUDED.can_edit, company_name = EXCLUDED.company_name`,
		boardID, companyID, name, canEdit, by)
	return err
}

func (r *Repo) DeleteBoardCompanyShare(ctx context.Context, boardID, companyID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM board_company_shares WHERE board_id = $1 AND company_id = $2`, boardID, companyID)
	return err
}

// BoardAudienceUserIDs — все, кто видит доску: прямые адресаты (пользователь/
// компания→участники) и аудитория расшаренных папок-предков. Одним запросом.
func (r *Repo) BoardAudienceUserIDs(ctx context.Context, boardID int64) ([]int64, error) {
	return r.scanIDs(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id FROM board_folders WHERE id = (SELECT folder_id FROM boards WHERE id = $1)
			UNION ALL
			SELECT f.id, f.parent_id FROM board_folders f JOIN ancestors a ON f.id = a.parent_id
		),
		comp AS (
			SELECT company_id FROM board_company_shares WHERE board_id = $1
			UNION
			SELECT company_id FROM board_folder_company_shares WHERE folder_id IN (SELECT id FROM ancestors)
		)
		SELECT user_id FROM board_user_shares WHERE board_id = $1
		UNION
		SELECT user_id FROM board_folder_user_shares WHERE folder_id IN (SELECT id FROM ancestors)
		UNION
		SELECT user_id FROM user_companies WHERE company_id IN (SELECT company_id FROM comp)`, boardID)
}

// ── Адресный шаринг папок (пользователь и компания) ──────────────────

func (r *Repo) ListFolderMembers(ctx context.Context, folderID int64) ([]*domain.Member, error) {
	out := []*domain.Member{}
	rows, err := r.pool.Query(ctx, `
		SELECT s.user_id, u.fio, u.avatar_path, s.can_edit, s.created_at
		  FROM board_folder_user_shares s JOIN users u ON u.id = s.user_id
		 WHERE s.folder_id = $1 ORDER BY u.fio, s.user_id`, folderID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		m := &domain.Member{Target: domain.TargetUser}
		if err := rows.Scan(&m.UserID, &m.FIO, &m.AvatarPath, &m.CanEdit, &m.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	crows, err := r.pool.Query(ctx, `
		SELECT company_id, company_name, can_edit, created_at
		  FROM board_folder_company_shares WHERE folder_id = $1 ORDER BY company_name, company_id`, folderID)
	if err != nil {
		return nil, err
	}
	defer crows.Close()
	for crows.Next() {
		m := &domain.Member{Target: domain.TargetCompany}
		if err := crows.Scan(&m.CompanyID, &m.CompanyName, &m.CanEdit, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, crows.Err()
}

func (r *Repo) UpsertFolderUserShare(ctx context.Context, folderID, userID int64, canEdit bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO board_folder_user_shares (folder_id, user_id, can_edit) VALUES ($1, $2, $3)
		ON CONFLICT (folder_id, user_id) DO UPDATE SET can_edit = EXCLUDED.can_edit`, folderID, userID, canEdit)
	return err
}

func (r *Repo) DeleteFolderUserShare(ctx context.Context, folderID, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM board_folder_user_shares WHERE folder_id = $1 AND user_id = $2`, folderID, userID)
	return err
}

func (r *Repo) UpsertFolderCompanyShare(ctx context.Context, folderID, companyID int64, name string, canEdit bool, by int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO board_folder_company_shares (folder_id, company_id, company_name, can_edit, shared_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (folder_id, company_id)
		DO UPDATE SET can_edit = EXCLUDED.can_edit, company_name = EXCLUDED.company_name`,
		folderID, companyID, name, canEdit, by)
	return err
}

func (r *Repo) DeleteFolderCompanyShare(ctx context.Context, folderID, companyID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM board_folder_company_shares WHERE folder_id = $1 AND company_id = $2`, folderID, companyID)
	return err
}

// FolderAudienceUserIDs — все, кто видит папку: её шары (пользователь/компания→
// участники) и шары папок-предков.
func (r *Repo) FolderAudienceUserIDs(ctx context.Context, folderID int64) ([]int64, error) {
	return r.scanIDs(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id FROM board_folders WHERE id = $1
			UNION ALL
			SELECT f.id, f.parent_id FROM board_folders f JOIN ancestors a ON f.id = a.parent_id
		),
		comp AS (
			SELECT company_id FROM board_folder_company_shares WHERE folder_id IN (SELECT id FROM ancestors)
		)
		SELECT user_id FROM board_folder_user_shares WHERE folder_id IN (SELECT id FROM ancestors)
		UNION
		SELECT user_id FROM user_companies WHERE company_id IN (SELECT company_id FROM comp)`, folderID)
}

// ── Разрешение эффективного доступа ──────────────────────────────────

// BoardAccess — доступ пользователя к доске с учётом прямых шар (пользователь/
// компания) и расшаренных папок-предков. folderID nil — доска в корне (папок-
// предков нет). Возвращает (найден доступ, можно ли править).
func (r *Repo) BoardAccess(ctx context.Context, userID int64, companyIDs []int64, boardID int64, folderID *int64) (bool, bool, error) {
	var (
		found   bool
		canEdit bool
	)
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id FROM board_folders WHERE id = $4
			UNION ALL
			SELECT f.id, f.parent_id FROM board_folders f JOIN ancestors a ON f.id = a.parent_id
		),
		grants AS (
			SELECT can_edit FROM board_user_shares WHERE board_id = $3 AND user_id = $1
			UNION ALL
			SELECT can_edit FROM board_company_shares WHERE board_id = $3 AND company_id = ANY($2::bigint[])
			UNION ALL
			SELECT can_edit FROM board_folder_user_shares WHERE user_id = $1 AND folder_id IN (SELECT id FROM ancestors)
			UNION ALL
			SELECT can_edit FROM board_folder_company_shares WHERE company_id = ANY($2::bigint[]) AND folder_id IN (SELECT id FROM ancestors)
		)
		SELECT count(*) > 0, COALESCE(bool_or(can_edit), false) FROM grants`,
		userID, companyIDs, boardID, folderID).Scan(&found, &canEdit)
	return found, canEdit, err
}

// FolderAccess — доступ пользователя к папке: сама папка или любой её предок
// расшарены мне (пользователю/компании).
func (r *Repo) FolderAccess(ctx context.Context, userID int64, companyIDs []int64, folderID int64) (bool, bool, error) {
	var (
		found   bool
		canEdit bool
	)
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id FROM board_folders WHERE id = $3
			UNION ALL
			SELECT f.id, f.parent_id FROM board_folders f JOIN ancestors a ON f.id = a.parent_id
		),
		grants AS (
			SELECT can_edit FROM board_folder_user_shares WHERE user_id = $1 AND folder_id IN (SELECT id FROM ancestors)
			UNION ALL
			SELECT can_edit FROM board_folder_company_shares WHERE company_id = ANY($2::bigint[]) AND folder_id IN (SELECT id FROM ancestors)
		)
		SELECT count(*) > 0, COALESCE(bool_or(can_edit), false) FROM grants`,
		userID, companyIDs, folderID).Scan(&found, &canEdit)
	return found, canEdit, err
}

func (r *Repo) scanIDs(ctx context.Context, q string, args ...any) ([]int64, error) {
	rows, err := r.pool.Query(ctx, q, args...)
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
