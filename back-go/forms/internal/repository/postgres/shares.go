package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
)

const shareCols = `sh.id, sh.form_id, sh.code, sh.name, sh.require_auth,
	sh.created_by, sh.created_at`

func (r *Repo) CreateShare(ctx context.Context, s *domain.Share) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO form_shares (form_id, code, name, require_auth, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		s.FormID, s.Code, s.Name, s.RequireAuth, s.CreatedBy).Scan(&s.ID, &s.CreatedAt)
}

// ListShares — ссылки формы со сводкой журнала: сколько раз открывали, когда в
// последний раз и сколько ответов пришло именно через эту ссылку.
func (r *Repo) ListShares(ctx context.Context, formID int64) ([]*domain.Share, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+shareCols+`,
		       (SELECT count(*) FROM form_share_visits v WHERE v.share_id = sh.id),
		       (SELECT max(v.visited_at) FROM form_share_visits v WHERE v.share_id = sh.id),
		       (SELECT count(*) FROM form_responses fr WHERE fr.share_id = sh.id)
		  FROM form_shares sh
		 WHERE sh.form_id = $1
		 ORDER BY sh.created_at DESC`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Share{}
	for rows.Next() {
		var s domain.Share
		if err := rows.Scan(&s.ID, &s.FormID, &s.Code, &s.Name, &s.RequireAuth,
			&s.CreatedBy, &s.CreatedAt, &s.Visits, &s.LastVisitAt, &s.Responses); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *Repo) GetShareByCode(ctx context.Context, code string) (*domain.Share, error) {
	var s domain.Share
	err := r.pool.QueryRow(ctx,
		`SELECT `+shareCols+` FROM form_shares sh WHERE sh.code = $1`, code).
		Scan(&s.ID, &s.FormID, &s.Code, &s.Name, &s.RequireAuth, &s.CreatedBy, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) UpdateShare(ctx context.Context, id, formID int64, name string, requireAuth bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE form_shares SET name = $3, require_auth = $4 WHERE id = $1 AND form_id = $2`,
		id, formID, name, requireAuth)
	return err
}

func (r *Repo) DeleteShare(ctx context.Context, id, formID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM form_shares WHERE id = $1 AND form_id = $2`, id, formID)
	return err
}

func (r *Repo) LogVisit(ctx context.Context, v *domain.ShareVisit) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO form_share_visits (share_id, user_id, ip, user_agent)
		VALUES ($1, $2, $3, $4)
		RETURNING id, visited_at`,
		v.ShareID, v.UserID, v.IP, v.UserAgent).Scan(&v.ID, &v.VisitedAt)
}

func (r *Repo) ListVisits(ctx context.Context, shareID int64, limit int) ([]*domain.ShareVisit, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT v.id, v.share_id, v.user_id, COALESCE(u.fio, ''), COALESCE(u.login, ''),
		       v.ip, v.user_agent, v.visited_at
		  FROM form_share_visits v
		  LEFT JOIN users u ON u.id = v.user_id
		 WHERE v.share_id = $1
		 ORDER BY v.visited_at DESC
		 LIMIT $2`, shareID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.ShareVisit{}
	for rows.Next() {
		var v domain.ShareVisit
		if err := rows.Scan(&v.ID, &v.ShareID, &v.UserID, &v.UserName, &v.UserLogin,
			&v.IP, &v.UserAgent, &v.VisitedAt); err != nil {
			return nil, err
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}
