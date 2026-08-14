package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

const shareCols = `id, registry_id, code, name, access, require_auth, created_by, created_at`

func scanShare(row pgx.Row) (*domain.Share, error) {
	var s domain.Share
	err := row.Scan(&s.ID, &s.RegistryID, &s.Code, &s.Name, &s.Access,
		&s.RequireAuth, &s.CreatedBy, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) CreateShare(ctx context.Context, s *domain.Share) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO registry_shares (registry_id, code, name, access, require_auth, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`,
		s.RegistryID, s.Code, s.Name, s.Access, s.RequireAuth, s.CreatedBy).
		Scan(&s.ID, &s.CreatedAt)
}

// ListShares — ссылки реестра со сводкой журнала переходов: сколько раз
// открывали и когда в последний. Считаем одним запросом — карточек ссылок
// немного, а второй круг за счётчиками дал бы N+1.
func (r *Repo) ListShares(ctx context.Context, registryID int64) ([]*domain.Share, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+prefixed(shareCols, "s")+`,
		       COALESCE(v.visits, 0), v.last_visit_at
		  FROM registry_shares s
		  LEFT JOIN (
		        SELECT share_id, count(*) AS visits, max(visited_at) AS last_visit_at
		          FROM registry_share_visits GROUP BY share_id
		  ) v ON v.share_id = s.id
		 WHERE s.registry_id = $1
		 ORDER BY s.created_at DESC`, registryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Share{}
	for rows.Next() {
		var s domain.Share
		if err := rows.Scan(&s.ID, &s.RegistryID, &s.Code, &s.Name, &s.Access,
			&s.RequireAuth, &s.CreatedBy, &s.CreatedAt, &s.Visits, &s.LastVisitAt); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *Repo) GetShareByCode(ctx context.Context, code string) (*domain.Share, error) {
	return scanShare(r.pool.QueryRow(ctx,
		`SELECT `+shareCols+` FROM registry_shares WHERE code = $1`, code))
}

func (r *Repo) UpdateShare(ctx context.Context, id, registryID int64, name, access string, requireAuth bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE registry_shares SET name = $3, access = $4, require_auth = $5
		  WHERE id = $1 AND registry_id = $2`,
		id, registryID, name, access, requireAuth)
	return err
}

func (r *Repo) DeleteShare(ctx context.Context, id, registryID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM registry_shares WHERE id = $1 AND registry_id = $2`, id, registryID)
	return err
}

// LogVisit — записать переход по ссылке. Журнал ведётся ради ответа на вопрос
// «кто это видел», поэтому пишется на КАЖДОМ открытии, а не раз на посетителя.
func (r *Repo) LogVisit(ctx context.Context, v *domain.ShareVisit) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO registry_share_visits (share_id, user_id, ip, user_agent)
		 VALUES ($1, $2, $3, $4) RETURNING id, visited_at`,
		v.ShareID, v.UserID, v.IP, v.UserAgent).Scan(&v.ID, &v.VisitedAt)
}

// ListVisits — журнал переходов ссылки, свежие первыми. У вошедшего есть имя и
// логин (по ним карточка ведёт в профиль), у гостя — только адрес и время.
func (r *Repo) ListVisits(ctx context.Context, shareID int64, limit int) ([]*domain.ShareVisit, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT v.id, v.share_id, v.user_id, COALESCE(u.fio, ''), COALESCE(u.login, ''),
		       v.ip, v.user_agent, v.visited_at
		  FROM registry_share_visits v
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
