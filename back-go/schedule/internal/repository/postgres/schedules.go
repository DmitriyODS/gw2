// Package postgres — персистентность schedulesvc (pgx, raw SQL).
//
// ИНВАРИАНТ: внешние данные попадают в запрос только плейсхолдерами. В тексте
// остаются константы файла и номера аргументов.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.ScheduleRepository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// ignoreNoRows — «строки нет» здесь не сбой: несуществующее расписание для
// спрашивающего неотличимо от чужого.
func ignoreNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

const scheduleCols = `s.id, s.owner_id, s.name, s.cycle_weeks, s.cycle_anchor,
	s.week_labels, s.timezone, s.gap_min, s.position, s.created_at, s.updated_at`

func scheduleTargets(s *domain.Schedule) []any {
	return []any{&s.ID, &s.OwnerID, &s.Name, &s.CycleWeeks, &s.CycleAnchor,
		&s.WeekLabels, &s.Timezone, &s.GapMin, &s.Position, &s.CreatedAt, &s.UpdatedAt}
}

// sharedCondition — расписание открыто пользователю адресно: лично либо одной
// из его компаний. Владение проверяется отдельно, по owner_id.
const sharedCondition = `EXISTS (SELECT 1 FROM schedule_user_shares sh
	 WHERE sh.schedule_id = s.id AND (sh.user_id = $1 OR sh.company_id = ANY($2)))`

// itemCountExpr — сколько занятий в расписании (подпись в списке).
const itemCountExpr = `(SELECT count(*) FROM schedule_items i WHERE i.schedule_id = s.id)`

func (r *Repo) ListOwned(ctx context.Context, ownerID int64) ([]*domain.Schedule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+scheduleCols+`, `+itemCountExpr+`
		  FROM schedules s
		 WHERE s.owner_id = $1
		 ORDER BY s.position, s.id`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Schedule{}
	for rows.Next() {
		var s domain.Schedule
		if err := rows.Scan(append(scheduleTargets(&s), &s.ItemCount)...); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

// ListShared — чужие расписания, открытые пользователю: имя владельца приходит
// сразу, чтобы список не досчитывал его вторым запросом.
func (r *Repo) ListShared(ctx context.Context, userID int64, companyIDs []int64) ([]*domain.Schedule, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+scheduleCols+`, `+itemCountExpr+`, COALESCE(u.fio, ''), u.avatar_path
		  FROM schedules s
		  LEFT JOIN users u ON u.id = s.owner_id
		 WHERE s.owner_id <> $1 AND `+sharedCondition+`
		 ORDER BY s.updated_at DESC, s.id DESC`, userID, companyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Schedule{}
	for rows.Next() {
		var s domain.Schedule
		targets := append(scheduleTargets(&s), &s.ItemCount, &s.OwnerName, &s.OwnerAvatar)
		if err := rows.Scan(targets...); err != nil {
			return nil, err
		}
		s.Shared = true
		out = append(out, &s)
	}
	return out, rows.Err()
}

func (r *Repo) GetSchedule(ctx context.Context, id int64) (*domain.Schedule, error) {
	var s domain.Schedule
	err := r.pool.QueryRow(ctx, `
		SELECT `+scheduleCols+`, `+itemCountExpr+`, COALESCE(u.fio, ''), u.avatar_path
		  FROM schedules s
		  LEFT JOIN users u ON u.id = s.owner_id
		 WHERE s.id = $1`, id).
		Scan(append(scheduleTargets(&s), &s.ItemCount, &s.OwnerName, &s.OwnerAvatar)...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repo) HasAccess(ctx context.Context, scheduleID, userID int64, companyIDs []int64) (bool, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT `+sharedCondition+` FROM schedules s WHERE s.id = $3`,
		userID, companyIDs, scheduleID).Scan(&ok)
	if err != nil {
		return false, ignoreNoRows(err)
	}
	return ok, nil
}

func (r *Repo) NextPosition(ctx context.Context, ownerID int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(max(position), 0) + 1 FROM schedules WHERE owner_id = $1`, ownerID).Scan(&pos)
	return pos, err
}

func (r *Repo) CreateSchedule(ctx context.Context, s *domain.Schedule) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO schedules (owner_id, name, cycle_weeks, cycle_anchor, week_labels,
		                       timezone, gap_min, position)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`,
		s.OwnerID, s.Name, s.CycleWeeks, s.CycleAnchor, s.WeekLabels,
		s.Timezone, s.GapMin, s.Position).
		Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *Repo) UpdateSchedule(ctx context.Context, s *domain.Schedule) error {
	return r.pool.QueryRow(ctx, `
		UPDATE schedules
		   SET name = $2, cycle_weeks = $3, cycle_anchor = $4, week_labels = $5,
		       timezone = $6, gap_min = $7, updated_at = now()
		 WHERE id = $1
		RETURNING updated_at`,
		s.ID, s.Name, s.CycleWeeks, s.CycleAnchor, s.WeekLabels, s.Timezone, s.GapMin).
		Scan(&s.UpdatedAt)
}

func (r *Repo) DeleteSchedule(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM schedules WHERE id = $1`, id)
	return err
}

// Audience — кому адресовать сокет-события расписания: владелец, адресаты
// личных шар и участники компаний, которым оно роздано. Событие уходит
// поимённо (комнаты user_{id}): расписание не принадлежит компании.
func (r *Repo) Audience(ctx context.Context, scheduleID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT owner_id FROM schedules WHERE id = $1
		UNION
		SELECT sh.user_id FROM schedule_user_shares sh
		 WHERE sh.schedule_id = $1 AND sh.user_id IS NOT NULL
		UNION
		SELECT uc.user_id FROM schedule_user_shares sh
		  JOIN user_companies uc ON uc.company_id = sh.company_id
		 WHERE sh.schedule_id = $1 AND sh.company_id IS NOT NULL`, scheduleID)
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
