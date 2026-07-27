package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"
)

// searchWords — «умный» разбор поискового запроса: отдельные слова, каждое
// ищется как подстрока (ILIKE) с логикой И — совпадать должны все слова в любом
// порядке (по заголовку и тексту). Не более 8 слов (защита от абьюза).
func searchWords(q string) []string {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}
	out := []string{}
	for _, w := range strings.Fields(q) {
		if w != "" {
			out = append(out, w)
			if len(out) == 8 {
				break
			}
		}
	}
	return out
}

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.BoardRepository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) ListBoards(ctx context.Context, f domain.BoardListFilter) ([]*domain.Board, error) {
	q := `SELECT n.id, n.owner_id, n.title, n.color, n.archived, n.folder_id, n.pinned_at,
	             left(n.text_content, 300), n.preview_path, n.created_at, n.updated_at
	        FROM boards n
	       WHERE n.archived = $1`
	args := []any{f.Archived}
	if f.OwnerID > 0 {
		args = append(args, f.OwnerID)
		q += ` AND n.owner_id = $` + strconv.Itoa(len(args))
	}
	if f.FolderSet {
		if f.FolderID == nil {
			q += ` AND n.folder_id IS NULL`
		} else {
			args = append(args, *f.FolderID)
			q += ` AND n.folder_id = $` + strconv.Itoa(len(args))
		}
	}
	for _, w := range searchWords(f.Search) {
		args = append(args, "%"+w+"%")
		q += ` AND (n.title || ' ' || n.text_content) ILIKE $` + strconv.Itoa(len(args))
	}
	q += ` ORDER BY n.pinned_at DESC NULLS LAST, n.updated_at DESC, n.id DESC`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Board{}
	for rows.Next() {
		var n domain.Board
		if err := rows.Scan(&n.ID, &n.OwnerID, &n.Title, &n.Color, &n.Archived, &n.FolderID, &n.PinnedAt,
			&n.Excerpt, &n.PreviewPath, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		n.FillPreviewURL()
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (r *Repo) GetBoard(ctx context.Context, id int64) (*domain.Board, error) {
	var n domain.Board
	err := r.pool.QueryRow(ctx, `
		SELECT n.id, n.owner_id, n.title, n.color, n.archived, n.folder_id, n.pinned_at,
		       n.scene, n.text_content, n.preview_path, n.created_at, n.updated_at
		  FROM boards n WHERE n.id = $1`, id).
		Scan(&n.ID, &n.OwnerID, &n.Title, &n.Color, &n.Archived, &n.FolderID, &n.PinnedAt,
			&n.Scene, &n.TextContent, &n.PreviewPath, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n.Excerpt = n.TextContent
	if rr := []rune(n.Excerpt); len(rr) > 300 {
		n.Excerpt = string(rr[:300])
	}
	n.FillPreviewURL()
	return &n, nil
}

// SetBoardPreview — ключ миниатюры холста (снимает клиент при сохранении).
func (r *Repo) SetBoardPreview(ctx context.Context, boardID int64, key string) error {
	_, err := r.pool.Exec(ctx, `UPDATE boards SET preview_path = $2 WHERE id = $1`, boardID, key)
	return err
}

func (r *Repo) CreateBoard(ctx context.Context, n *domain.Board) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO boards (owner_id, folder_id, title, color, scene, text_content)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		n.OwnerID, n.FolderID, n.Title, n.Color, n.Scene, n.TextContent).
		Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
}

func (r *Repo) UpdateBoard(ctx context.Context, n *domain.Board) error {
	return r.pool.QueryRow(ctx, `
		UPDATE boards SET title = $2, color = $3, archived = $4, pinned_at = $5, scene = $6, text_content = $7, updated_at = now()
		 WHERE id = $1 RETURNING updated_at`,
		n.ID, n.Title, n.Color, n.Archived, n.PinnedAt, n.Scene, n.TextContent).Scan(&n.UpdatedAt)
}

func (r *Repo) DeleteBoard(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM boards WHERE id = $1`, id)
	return err
}

func (r *Repo) MoveBoard(ctx context.Context, id int64, folderID *int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE boards SET folder_id = $2, updated_at = now() WHERE id = $1`, id, folderID)
	return err
}

// SharedByMeBoardIDs — множество id досок из ids, у которых есть любой шаринг
// (публичная ссылка / адресат / компания) — для значка на плитке владельца.
func (r *Repo) SharedByMeBoardIDs(ctx context.Context, ids []int64) (map[int64]bool, error) {
	res := map[int64]bool{}
	if len(ids) == 0 {
		return res, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT board_id FROM board_user_shares WHERE board_id = ANY($1::bigint[])
		UNION SELECT board_id FROM board_company_shares WHERE board_id = ANY($1::bigint[])
		UNION SELECT board_id FROM board_shares WHERE board_id = ANY($1::bigint[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		res[id] = true
	}
	return res, rows.Err()
}

// boardGrantsCTE — CTE «grants(board_id, can_edit)»: доски, доступные мне
// адресно (пользователь/компания) или через расшаренную папку-предка. Параметры
// $1 — user_id, $2 — company_ids. Общая для «поделились со мной» и оверлея.
const boardGrantsCTE = `
	WITH RECURSIVE
	roots AS (
		SELECT folder_id AS id, can_edit FROM board_folder_user_shares WHERE user_id = $1
		UNION ALL
		SELECT folder_id AS id, can_edit FROM board_folder_company_shares WHERE company_id = ANY($2::bigint[])
	),
	subtree AS (
		SELECT id, can_edit FROM roots
		UNION ALL
		SELECT f.id, s.can_edit FROM board_folders f JOIN subtree s ON f.parent_id = s.id
	),
	folder_boards AS (
		SELECT n.id AS board_id, bool_or(st.can_edit) AS can_edit
		  FROM boards n JOIN subtree st ON n.folder_id = st.id
		 GROUP BY n.id
	),
	direct AS (
		SELECT board_id, can_edit FROM board_user_shares WHERE user_id = $1
		UNION ALL
		SELECT board_id, can_edit FROM board_company_shares WHERE company_id = ANY($2::bigint[])
	),
	grants AS (
		SELECT board_id, bool_or(can_edit) AS can_edit FROM (
			SELECT board_id, can_edit FROM folder_boards
			UNION ALL
			SELECT board_id, can_edit FROM direct
		) g GROUP BY board_id
	)`

// ListSharedWithMe — плитки чужих досок, доступных мне: адресно (пользователь/
// компания) или через расшаренную папку-предка. Архивные владельца не
// показываются; исключаются доски, которые я уже разместил у себя/в личном
// архиве (есть строка board_recipient_state) — они уходят в моё дерево/архив.
func (r *Repo) ListSharedWithMe(ctx context.Context, userID int64, companyIDs []int64, search string) ([]*domain.Board, error) {
	q := boardGrantsCTE + `
		SELECT n.id, n.owner_id, n.title, n.color, n.archived, n.folder_id,
		       left(n.text_content, 300), n.preview_path, n.created_at, n.updated_at,
		       u.fio, u.avatar_path, g.can_edit
		  FROM grants g
		  JOIN boards n ON n.id = g.board_id
		  JOIN users u ON u.id = n.owner_id
		 WHERE n.owner_id <> $1 AND NOT n.archived
		   AND NOT EXISTS (SELECT 1 FROM board_recipient_state s WHERE s.user_id = $1 AND s.board_id = n.id)`
	args := []any{userID, companyIDs}
	for _, w := range searchWords(search) {
		args = append(args, "%"+w+"%")
		q += ` AND (n.title || ' ' || n.text_content) ILIKE $` + strconv.Itoa(len(args))
	}
	q += ` ORDER BY n.updated_at DESC, n.id DESC`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Board{}
	for rows.Next() {
		var (
			n       domain.Board
			canEdit bool
		)
		if err := rows.Scan(&n.ID, &n.OwnerID, &n.Title, &n.Color, &n.Archived, &n.FolderID, &n.Excerpt,
			&n.PreviewPath, &n.CreatedAt, &n.UpdatedAt, &n.OwnerName, &n.OwnerAvatar, &canEdit); err != nil {
			return nil, err
		}
		n.MyAccess = domain.AccessView
		if canEdit {
			n.MyAccess = domain.AccessEdit
		}
		n.FillPreviewURL()
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (r *Repo) SetBoardRecipientPlacement(ctx context.Context, userID, boardID int64, folderID *int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO board_recipient_state (user_id, board_id, folder_id, archived)
		VALUES ($1, $2, $3, FALSE)
		ON CONFLICT (user_id, board_id)
		DO UPDATE SET folder_id = EXCLUDED.folder_id, archived = FALSE, updated_at = now()`,
		userID, boardID, folderID)
	return err
}

func (r *Repo) SetBoardRecipientArchived(ctx context.Context, userID, boardID int64, archived bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO board_recipient_state (user_id, board_id, archived)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, board_id)
		DO UPDATE SET archived = EXCLUDED.archived, updated_at = now()`,
		userID, boardID, archived)
	return err
}

// ListRecipientBoards — расшаренные мне доски, размещённые в моём scope (папка/
// корень/архив); folder_id и archived плиток берутся из оверлея, а не владельца.
func (r *Repo) ListRecipientBoards(ctx context.Context, userID int64, companyIDs []int64, scope domain.RecipientScope, folderID *int64) ([]*domain.Board, error) {
	args := []any{userID, companyIDs}
	var cond string
	switch scope {
	case domain.RecipientArchive:
		cond = `st.archived`
	case domain.RecipientRoot:
		cond = `st.folder_id IS NULL AND NOT st.archived`
	default: // folder
		args = append(args, folderID)
		cond = `st.folder_id = $3 AND NOT st.archived`
	}
	q := boardGrantsCTE + `
		SELECT n.id, n.owner_id, n.title, n.color, st.archived, st.folder_id,
		       left(n.text_content, 300), n.preview_path, n.created_at, n.updated_at,
		       u.fio, u.avatar_path, g.can_edit
		  FROM grants g
		  JOIN boards n ON n.id = g.board_id
		  JOIN users u ON u.id = n.owner_id
		  JOIN board_recipient_state st ON st.user_id = $1 AND st.board_id = n.id
		 WHERE n.owner_id <> $1 AND ` + cond + `
		 ORDER BY n.updated_at DESC, n.id DESC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Board{}
	for rows.Next() {
		var (
			n       domain.Board
			canEdit bool
		)
		if err := rows.Scan(&n.ID, &n.OwnerID, &n.Title, &n.Color, &n.Archived, &n.FolderID, &n.Excerpt,
			&n.PreviewPath, &n.CreatedAt, &n.UpdatedAt, &n.OwnerName, &n.OwnerAvatar, &canEdit); err != nil {
			return nil, err
		}
		n.MyAccess = domain.AccessView
		if canEdit {
			n.MyAccess = domain.AccessEdit
		}
		n.FillPreviewURL()
		out = append(out, &n)
	}
	return out, rows.Err()
}
