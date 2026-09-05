package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

/* Шаринг расписания двумя путями: публичной ссылкой (код-capability в URL) и
   адресной выдачей человеку либо компании. Уровень один — ЧТЕНИЕ: вести
   занятия может владелец. */

const shareCols = `id, schedule_id, code, created_by, created_at`

func (r *Repo) CreateShare(ctx context.Context, s *domain.Share) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO schedule_shares (schedule_id, code, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`, s.ScheduleID, s.Code, s.CreatedBy).
		Scan(&s.ID, &s.CreatedAt)
}

func (r *Repo) ListShares(ctx context.Context, scheduleID int64) ([]*domain.Share, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+shareCols+` FROM schedule_shares
		  WHERE schedule_id = $1 ORDER BY created_at DESC`, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Share{}
	for rows.Next() {
		var s domain.Share
		if err := rows.Scan(&s.ID, &s.ScheduleID, &s.Code, &s.CreatedBy, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *Repo) GetShareByCode(ctx context.Context, code string) (*domain.Share, error) {
	var s domain.Share
	err := r.pool.QueryRow(ctx,
		`SELECT `+shareCols+` FROM schedule_shares WHERE code = $1`, code).
		Scan(&s.ID, &s.ScheduleID, &s.Code, &s.CreatedBy, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repo) DeleteShare(ctx context.Context, id, scheduleID int64) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM schedule_shares WHERE id = $1 AND schedule_id = $2`, id, scheduleID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrShareNotFound
	}
	return nil
}

// ListUserShares — кому открыто расписание: люди и компании одним списком с
// именами, чтобы карточка «Поделиться» не досчитывала их вторым запросом.
func (r *Repo) ListUserShares(ctx context.Context, scheduleID int64) ([]*domain.UserShare, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sh.id, sh.schedule_id, sh.user_id, sh.company_id, sh.created_by, sh.created_at,
		       COALESCE(u.fio, c.name, ''), u.avatar_path, COALESCE(u.login, '')
		  FROM schedule_user_shares sh
		  LEFT JOIN users u ON u.id = sh.user_id
		  LEFT JOIN companies c ON c.id = sh.company_id
		 WHERE sh.schedule_id = $1
		 ORDER BY sh.company_id NULLS LAST, sh.created_at`, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.UserShare{}
	for rows.Next() {
		var s domain.UserShare
		if err := rows.Scan(&s.ID, &s.ScheduleID, &s.UserID, &s.CompanyID, &s.CreatedBy,
			&s.CreatedAt, &s.Name, &s.AvatarPath, &s.Login); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

// PutUserShare — открыть доступ; повторная выдача не плодит строки (частичные
// уникальные индексы миграции 00088). created_by — тот, КТО выдал.
func (r *Repo) PutUserShare(ctx context.Context, s *domain.UserShare) error {
	if s.UserID != nil {
		return r.pool.QueryRow(ctx, `
			INSERT INTO schedule_user_shares (schedule_id, user_id, created_by)
			VALUES ($1, $2, $3)
			ON CONFLICT (schedule_id, user_id) WHERE user_id IS NOT NULL
			DO UPDATE SET created_by = EXCLUDED.created_by
			RETURNING id, created_at`, s.ScheduleID, *s.UserID, s.CreatedBy).
			Scan(&s.ID, &s.CreatedAt)
	}
	return r.pool.QueryRow(ctx, `
		INSERT INTO schedule_user_shares (schedule_id, company_id, created_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (schedule_id, company_id) WHERE company_id IS NOT NULL
		DO UPDATE SET created_by = EXCLUDED.created_by
		RETURNING id, created_at`, s.ScheduleID, *s.CompanyID, s.CreatedBy).
		Scan(&s.ID, &s.CreatedAt)
}

func (r *Repo) DeleteUserShare(ctx context.Context, scheduleID int64, userID, companyID *int64) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM schedule_user_shares
		 WHERE schedule_id = $1
		   AND ($2::bigint IS NULL OR user_id = $2)
		   AND ($3::bigint IS NULL OR company_id = $3)
		   AND (($2::bigint IS NOT NULL AND user_id IS NOT NULL)
		     OR ($3::bigint IS NOT NULL AND company_id IS NOT NULL))`,
		scheduleID, userID, companyID)
	return err
}
