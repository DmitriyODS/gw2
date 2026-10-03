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

	"github.com/DmitriyODS/gw2/back-go/pkg/spaces"
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

const scheduleCols = `s.id, s.owner_id, s.company_id, s.team_access, s.name, s.cycle_weeks,
	s.cycle_anchor, s.week_labels, s.timezone, s.gap_min, s.position, s.created_at, s.updated_at`

func scheduleTargets(s *domain.Schedule) []any {
	return []any{&s.ID, &s.OwnerID, &s.CompanyID, &s.TeamAccess, &s.Name, &s.CycleWeeks,
		&s.CycleAnchor, &s.WeekLabels, &s.Timezone, &s.GapMin, &s.Position, &s.CreatedAt, &s.UpdatedAt}
}

// sharedCondition — расписание открыто пользователю $1 адресно: лично либо
// одной из его команд ($2).
const sharedCondition = `EXISTS (SELECT 1 FROM schedule_user_shares sh
	 WHERE sh.schedule_id = s.id AND (sh.user_id = $1 OR sh.company_id = ANY($2)))`

// inMySpaces — расписание лежит в пространстве человека $1: личное его либо
// одной из его команд ($2).
const inMySpaces = `((s.company_id IS NULL AND s.owner_id = $1) OR s.company_id = ANY($2))`

// visibleCond — расписание доступно человеку: в его пространстве или открыто
// ему адресно.
const visibleCond = `(` + inMySpaces + ` OR ` + sharedCondition + `)`

/*
accessExpr — уровень человека $1 (команды $2) к расписанию s.

	Хозяин личного, а у расписания команды — автор и администраторы
	распоряжаются им; участники команды получают team_access, адресаты —
	просмотр. Держать в паре с domain/access.go.
*/
var accessExpr = `
	CASE
	  WHEN s.company_id IS NULL AND s.owner_id = $1 THEN 'owner'
	  WHEN s.company_id = ANY($2) AND (s.owner_id = $1 OR ` + spaces.Admin("$1", "s.company_id") + `) THEN 'owner'
	  WHEN s.company_id = ANY($2) THEN s.team_access
	  WHEN ` + sharedCondition + ` THEN 'view'
	  ELSE ''
	END`

// itemCountExpr — сколько занятий в расписании (подпись в списке).
const itemCountExpr = `(SELECT count(*) FROM schedule_items i WHERE i.schedule_id = s.id)`

// ListOwned — расписания пространств человека: личные и всех его команд, с
// уровнем доступа и названием команды.
func (r *Repo) ListOwned(ctx context.Context, userID int64, companyIDs []int64) ([]*domain.Schedule, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+scheduleCols+`, `+itemCountExpr+`, `+accessExpr+`, COALESCE(co.name, '')
		  FROM schedules s
		  LEFT JOIN companies co ON co.id = s.company_id
		 WHERE `+inMySpaces+`
		 ORDER BY s.position, s.id`, userID, companyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Schedule{}
	for rows.Next() {
		var s domain.Schedule
		if err := rows.Scan(append(scheduleTargets(&s), &s.ItemCount, &s.MyAccess, &s.CompanyName)...); err != nil {
			return nil, err
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

// ListShared — чужие расписания, открытые пользователю адресно и не лежащие в
// его пространствах: имя владельца приходит сразу.
func (r *Repo) ListShared(ctx context.Context, userID int64, companyIDs []int64) ([]*domain.Schedule, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+scheduleCols+`, `+itemCountExpr+`, COALESCE(u.fio, ''), u.avatar_path
		  FROM schedules s
		  LEFT JOIN users u ON u.id = s.owner_id
		 WHERE NOT `+inMySpaces+` AND `+sharedCondition+`
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
		s.MyAccess = domain.AccessView
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

// AccessOf — уровень человека к расписанию ("" — доступа нет).
func (r *Repo) AccessOf(ctx context.Context, scheduleID, userID int64, companyIDs []int64) (string, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	var access string
	err := r.pool.QueryRow(ctx,
		`SELECT `+accessExpr+` FROM schedules s WHERE s.id = $3`,
		userID, companyIDs, scheduleID).Scan(&access)
	if err != nil {
		return domain.AccessNone, ignoreNoRows(err)
	}
	return access, nil
}

func (r *Repo) NextPosition(ctx context.Context, ownerID int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(max(position), 0) + 1 FROM schedules WHERE owner_id = $1`, ownerID).Scan(&pos)
	return pos, err
}

func (r *Repo) CreateSchedule(ctx context.Context, s *domain.Schedule) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO schedules (owner_id, company_id, team_access, name, cycle_weeks, cycle_anchor,
		                       week_labels, timezone, gap_min, position)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`,
		s.OwnerID, s.CompanyID, s.TeamAccess, s.Name, s.CycleWeeks, s.CycleAnchor, s.WeekLabels,
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

func (r *Repo) MoveSchedule(ctx context.Context, id, ownerID int64, companyID *int64, teamAccess string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE schedules SET owner_id = $2, company_id = $3, team_access = $4, updated_at = now()
		  WHERE id = $1`, id, ownerID, companyID, teamAccess)
	return err
}

func (r *Repo) DeleteSchedule(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM schedules WHERE id = $1`, id)
	return err
}

// Audience — кому адресовать сокет-события расписания: автор, участники
// команды-пространства, адресаты личных шар и участники команд, которым оно
// роздано. Событие уходит поимённо (комнаты user_{id}).
func (r *Repo) Audience(ctx context.Context, scheduleID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT owner_id FROM schedules WHERE id = $1
		UNION
		SELECT uc.user_id FROM schedules s
		  JOIN user_companies uc ON uc.company_id = s.company_id
		 WHERE s.id = $1
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
