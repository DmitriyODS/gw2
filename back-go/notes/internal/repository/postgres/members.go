package postgres

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

func (r *Repo) ListMembers(ctx context.Context, noteID int64) ([]*domain.NoteMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.user_id, u.fio, u.avatar_path, s.can_edit, s.created_at
		  FROM note_user_shares s
		  JOIN users u ON u.id = s.user_id
		 WHERE s.note_id = $1
		 ORDER BY u.fio, s.user_id`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.NoteMember{}
	for rows.Next() {
		var m domain.NoteMember
		if err := rows.Scan(&m.UserID, &m.FIO, &m.AvatarPath, &m.CanEdit, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (r *Repo) UpsertMember(ctx context.Context, noteID, userID int64, canEdit bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO note_user_shares (note_id, user_id, can_edit) VALUES ($1, $2, $3)
		ON CONFLICT (note_id, user_id) DO UPDATE SET can_edit = EXCLUDED.can_edit`,
		noteID, userID, canEdit)
	return err
}

func (r *Repo) DeleteMember(ctx context.Context, noteID, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM note_user_shares WHERE note_id = $1 AND user_id = $2`, noteID, userID)
	return err
}

func (r *Repo) GetMember(ctx context.Context, noteID, userID int64) (bool, bool, error) {
	var canEdit bool
	err := r.pool.QueryRow(ctx,
		`SELECT can_edit FROM note_user_shares WHERE note_id = $1 AND user_id = $2`,
		noteID, userID).Scan(&canEdit)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return true, canEdit, nil
}

func (r *Repo) MemberIDs(ctx context.Context, noteID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id FROM note_user_shares WHERE note_id = $1 ORDER BY user_id`, noteID)
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

// ListSharedWithMe — плитки чужих заметок, открытых пользователю адресно.
// Группы владельца не отдаются (личная организация), закрепление не участвует.
func (r *Repo) ListSharedWithMe(ctx context.Context, userID int64, search string) ([]*domain.Note, error) {
	q := `SELECT n.id, n.owner_id, n.title, n.color, n.archived, left(n.text_content, 300),
	             n.created_at, n.updated_at, u.fio, u.avatar_path, s.can_edit
	        FROM note_user_shares s
	        JOIN notes n ON n.id = s.note_id
	        JOIN users u ON u.id = n.owner_id
	       WHERE s.user_id = $1`
	args := []any{userID}
	if search != "" {
		args = append(args, "%"+search+"%")
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
		if err := rows.Scan(&n.ID, &n.OwnerID, &n.Title, &n.Color, &n.Archived, &n.Excerpt,
			&n.CreatedAt, &n.UpdatedAt, &n.OwnerName, &n.OwnerAvatar, &canEdit); err != nil {
			return nil, err
		}
		n.MyAccess = domain.AccessView
		if canEdit {
			n.MyAccess = domain.AccessEdit
		}
		n.GroupIDs = []int64{}
		out = append(out, &n)
	}
	return out, rows.Err()
}
