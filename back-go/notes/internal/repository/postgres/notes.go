package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
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

var _ domain.NoteRepository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// noteTags — агрегат тегов заметки для выборок (пустой массив, если связей нет).
const noteTags = `COALESCE((SELECT array_agg(tag_id ORDER BY tag_id)
	FROM note_tag_items WHERE note_id = n.id), '{}')`

func (r *Repo) ListNotes(ctx context.Context, f domain.NoteListFilter) ([]*domain.Note, error) {
	q := `SELECT n.id, n.owner_id, n.title, n.color, n.archived, n.folder_id, n.pinned_at,
	             left(n.text_content, 300), n.created_at, n.updated_at, ` + noteTags + `
	        FROM notes n
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
	if len(f.TagIDs) > 0 {
		args = append(args, f.TagIDs)
		q += ` AND EXISTS (SELECT 1 FROM note_tag_items ti
			WHERE ti.note_id = n.id AND ti.tag_id = ANY($` + strconv.Itoa(len(args)) + `::bigint[]))`
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
	out := []*domain.Note{}
	for rows.Next() {
		var n domain.Note
		if err := rows.Scan(&n.ID, &n.OwnerID, &n.Title, &n.Color, &n.Archived, &n.FolderID, &n.PinnedAt,
			&n.Excerpt, &n.CreatedAt, &n.UpdatedAt, &n.TagIDs); err != nil {
			return nil, err
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (r *Repo) GetNote(ctx context.Context, id int64) (*domain.Note, error) {
	var n domain.Note
	err := r.pool.QueryRow(ctx, `
		SELECT n.id, n.owner_id, n.title, n.color, n.archived, n.folder_id, n.pinned_at,
		       n.doc, n.text_content, n.created_at, n.updated_at, `+noteTags+`
		  FROM notes n WHERE n.id = $1`, id).
		Scan(&n.ID, &n.OwnerID, &n.Title, &n.Color, &n.Archived, &n.FolderID, &n.PinnedAt,
			&n.Doc, &n.TextContent, &n.CreatedAt, &n.UpdatedAt, &n.TagIDs)
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
	return &n, nil
}

func (r *Repo) CreateNote(ctx context.Context, n *domain.Note) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO notes (owner_id, folder_id, title, color, doc, text_content)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		n.OwnerID, n.FolderID, n.Title, n.Color, n.Doc, n.TextContent).
		Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
}

func (r *Repo) UpdateNote(ctx context.Context, n *domain.Note) error {
	return r.pool.QueryRow(ctx, `
		UPDATE notes SET title = $2, color = $3, archived = $4, pinned_at = $5, doc = $6, text_content = $7, updated_at = now()
		 WHERE id = $1 RETURNING updated_at`,
		n.ID, n.Title, n.Color, n.Archived, n.PinnedAt, n.Doc, n.TextContent).Scan(&n.UpdatedAt)
}

func (r *Repo) DeleteNote(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM notes WHERE id = $1`, id)
	return err
}

func (r *Repo) MoveNote(ctx context.Context, id int64, folderID *int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE notes SET folder_id = $2, updated_at = now() WHERE id = $1`, id, folderID)
	return err
}

// SetNoteTags — полная замена связей заметки с тегами (в транзакции).
func (r *Repo) SetNoteTags(ctx context.Context, noteID int64, tagIDs []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM note_tag_items WHERE note_id = $1`, noteID); err != nil {
		return err
	}
	if len(tagIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO note_tag_items (note_id, tag_id)
			SELECT $1, unnest($2::bigint[]) ON CONFLICT DO NOTHING`, noteID, tagIDs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// SharedByMeNoteIDs — множество id заметок из ids, у которых есть любой шаринг
// (публичная ссылка / адресат / компания) — для значка на плитке владельца.
func (r *Repo) SharedByMeNoteIDs(ctx context.Context, ids []int64) (map[int64]bool, error) {
	res := map[int64]bool{}
	if len(ids) == 0 {
		return res, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT note_id FROM note_user_shares WHERE note_id = ANY($1::bigint[])
		UNION SELECT note_id FROM note_company_shares WHERE note_id = ANY($1::bigint[])
		UNION SELECT note_id FROM note_shares WHERE note_id = ANY($1::bigint[])`, ids)
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

// noteGrantsCTE — CTE «grants(note_id, can_edit)»: заметки, доступные мне
// адресно (пользователь/компания) или через расшаренную папку-предка. Параметры
// $1 — user_id, $2 — company_ids. Общая для «поделились со мной» и оверлея.
const noteGrantsCTE = `
	WITH RECURSIVE
	roots AS (
		SELECT folder_id AS id, can_edit FROM folder_user_shares WHERE user_id = $1
		UNION ALL
		SELECT folder_id AS id, can_edit FROM folder_company_shares WHERE company_id = ANY($2::bigint[])
	),
	subtree AS (
		SELECT id, can_edit FROM roots
		UNION ALL
		SELECT f.id, s.can_edit FROM note_folders f JOIN subtree s ON f.parent_id = s.id
	),
	folder_notes AS (
		SELECT n.id AS note_id, bool_or(st.can_edit) AS can_edit
		  FROM notes n JOIN subtree st ON n.folder_id = st.id
		 GROUP BY n.id
	),
	direct AS (
		SELECT note_id, can_edit FROM note_user_shares WHERE user_id = $1
		UNION ALL
		SELECT note_id, can_edit FROM note_company_shares WHERE company_id = ANY($2::bigint[])
	),
	grants AS (
		SELECT note_id, bool_or(can_edit) AS can_edit FROM (
			SELECT note_id, can_edit FROM folder_notes
			UNION ALL
			SELECT note_id, can_edit FROM direct
		) g GROUP BY note_id
	)`

// ListSharedWithMe — плитки чужих заметок, доступных мне: адресно (пользователь/
// компания) или через расшаренную папку-предка. Архивные владельца не
// показываются; исключаются заметки, которые я уже разместил у себя/в личном
// архиве (есть строка note_recipient_state) — они уходят в моё дерево/архив.
func (r *Repo) ListSharedWithMe(ctx context.Context, userID int64, companyIDs []int64, search string) ([]*domain.Note, error) {
	q := noteGrantsCTE + `
		SELECT n.id, n.owner_id, n.title, n.color, n.archived, n.folder_id,
		       left(n.text_content, 300), n.created_at, n.updated_at,
		       u.fio, u.avatar_path, g.can_edit
		  FROM grants g
		  JOIN notes n ON n.id = g.note_id
		  JOIN users u ON u.id = n.owner_id
		 WHERE n.owner_id <> $1 AND NOT n.archived
		   AND NOT EXISTS (SELECT 1 FROM note_recipient_state s WHERE s.user_id = $1 AND s.note_id = n.id)`
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
	out := []*domain.Note{}
	for rows.Next() {
		var (
			n       domain.Note
			canEdit bool
		)
		if err := rows.Scan(&n.ID, &n.OwnerID, &n.Title, &n.Color, &n.Archived, &n.FolderID, &n.Excerpt,
			&n.CreatedAt, &n.UpdatedAt, &n.OwnerName, &n.OwnerAvatar, &canEdit); err != nil {
			return nil, err
		}
		n.MyAccess = domain.AccessView
		if canEdit {
			n.MyAccess = domain.AccessEdit
		}
		n.TagIDs = []int64{}
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (r *Repo) SetNoteRecipientPlacement(ctx context.Context, userID, noteID int64, folderID *int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO note_recipient_state (user_id, note_id, folder_id, archived)
		VALUES ($1, $2, $3, FALSE)
		ON CONFLICT (user_id, note_id)
		DO UPDATE SET folder_id = EXCLUDED.folder_id, archived = FALSE, updated_at = now()`,
		userID, noteID, folderID)
	return err
}

func (r *Repo) SetNoteRecipientArchived(ctx context.Context, userID, noteID int64, archived bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO note_recipient_state (user_id, note_id, archived)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, note_id)
		DO UPDATE SET archived = EXCLUDED.archived, updated_at = now()`,
		userID, noteID, archived)
	return err
}

// ListRecipientNotes — расшаренные мне заметки, размещённые в моём scope (папка/
// корень/архив); folder_id и archived плиток берутся из оверлея, а не владельца.
func (r *Repo) ListRecipientNotes(ctx context.Context, userID int64, companyIDs []int64, scope domain.RecipientScope, folderID *int64) ([]*domain.Note, error) {
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
	q := noteGrantsCTE + `
		SELECT n.id, n.owner_id, n.title, n.color, st.archived, st.folder_id,
		       left(n.text_content, 300), n.created_at, n.updated_at,
		       u.fio, u.avatar_path, g.can_edit
		  FROM grants g
		  JOIN notes n ON n.id = g.note_id
		  JOIN users u ON u.id = n.owner_id
		  JOIN note_recipient_state st ON st.user_id = $1 AND st.note_id = n.id
		 WHERE n.owner_id <> $1 AND ` + cond + `
		 ORDER BY n.updated_at DESC, n.id DESC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Note{}
	for rows.Next() {
		var (
			n       domain.Note
			canEdit bool
		)
		if err := rows.Scan(&n.ID, &n.OwnerID, &n.Title, &n.Color, &n.Archived, &n.FolderID, &n.Excerpt,
			&n.CreatedAt, &n.UpdatedAt, &n.OwnerName, &n.OwnerAvatar, &canEdit); err != nil {
			return nil, err
		}
		n.MyAccess = domain.AccessView
		if canEdit {
			n.MyAccess = domain.AccessEdit
		}
		n.TagIDs = []int64{}
		out = append(out, &n)
	}
	return out, rows.Err()
}
