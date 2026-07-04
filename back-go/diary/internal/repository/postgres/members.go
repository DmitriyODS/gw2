package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
)

func (r *Repo) ListMembers(ctx context.Context, diaryID int64) ([]*domain.Member, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.user_id, u.fio, u.avatar_path, s.can_check, s.created_at
		  FROM diary_user_shares s
		  JOIN users u ON u.id = s.user_id
		 WHERE s.diary_id = $1
		 ORDER BY u.fio, s.user_id`, diaryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Member{}
	for rows.Next() {
		var m domain.Member
		if err := rows.Scan(&m.UserID, &m.FIO, &m.AvatarPath, &m.CanCheck, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (r *Repo) MemberIDs(ctx context.Context, diaryID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id FROM diary_user_shares WHERE diary_id = $1`, diaryID)
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

func (r *Repo) MemberAccess(ctx context.Context, diaryID, userID int64) (bool, bool, error) {
	var canCheck bool
	err := r.pool.QueryRow(ctx,
		`SELECT can_check FROM diary_user_shares WHERE diary_id = $1 AND user_id = $2`,
		diaryID, userID).Scan(&canCheck)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return true, canCheck, nil
}

func (r *Repo) AddMember(ctx context.Context, diaryID, userID int64, canCheck bool) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO diary_user_shares (diary_id, user_id, can_check) VALUES ($1, $2, $3)
		 ON CONFLICT (diary_id, user_id) DO UPDATE SET can_check = EXCLUDED.can_check`,
		diaryID, userID, canCheck)
	return err
}

func (r *Repo) RemoveMember(ctx context.Context, diaryID, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM diary_user_shares WHERE diary_id = $1 AND user_id = $2`, diaryID, userID)
	return err
}
