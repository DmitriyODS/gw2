package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/spaces"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.DiaryRepository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const diaryCols = `id, owner_id, company_id, team_access, kind, name, position, created_at, updated_at`

func scanDiary(row pgx.Row) (*domain.Diary, error) {
	var d domain.Diary
	err := row.Scan(&d.ID, &d.OwnerID, &d.CompanyID, &d.TeamAccess, &d.Kind, &d.Name, &d.Position,
		&d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

/*
accessExpr — эффективный уровень человека $1 к ежедневнику d.

	Хозяин личного ежедневника, а у ежедневника команды — автор и
	администраторы (пока состоят в ней) распоряжаются им целиком. Участникам
	команды — team_access, адресатам — просмотр или отметка; берётся сильнейшее.
*/
var accessExpr = `
	CASE
	  WHEN d.company_id IS NULL AND d.owner_id = $1 THEN 'owner'
	  WHEN d.company_id IS NOT NULL AND (` + spaces.Admin("$1", "d.company_id") + `
	       OR (d.owner_id = $1 AND ` + spaces.Member("$1", "d.company_id") + `)) THEN 'owner'
	  ELSE COALESCE((
		SELECT CASE max(lvl) WHEN 3 THEN 'edit' WHEN 2 THEN 'check' WHEN 1 THEN 'view' END
		  FROM (
		    SELECT CASE d.team_access WHEN 'edit' THEN 3 WHEN 'check' THEN 2 ELSE 1 END AS lvl
		     WHERE d.company_id IS NOT NULL AND ` + spaces.Member("$1", "d.company_id") + `
		    UNION ALL
		    SELECT CASE WHEN s.can_check THEN 2 ELSE 1 END
		      FROM diary_user_shares s WHERE s.diary_id = d.id AND s.user_id = $1
		  ) lv
	  ), '')
	END`

// inMySpaces — ежедневник лежит в пространстве человека $1.
var inMySpaces = `((d.company_id IS NULL AND d.owner_id = $1)
	OR d.company_id IN (` + spaces.MyTeams("$1") + `))`

// visibleCond — ежедневник доступен человеку $1: в его пространстве или
// открыт ему адресно.
var visibleCond = `(` + inMySpaces + `
	OR EXISTS (SELECT 1 FROM diary_user_shares s WHERE s.diary_id = d.id AND s.user_id = $1))`

// diaryCounts — прогресс списка: активные/выполненные записи на ежедневник.
const diaryCounts = `LEFT JOIN (
		SELECT diary_id,
		       COUNT(*) FILTER (WHERE NOT done) AS active,
		       COUNT(*) FILTER (WHERE done)     AS done
		  FROM diary_records GROUP BY diary_id
	) c ON c.diary_id = d.id`

// ListOwned — ежедневники пространств человека (личные и всех его команд,
// кроме скрытого «Моего дня») с уровнем доступа и названием команды.
func (r *Repo) ListOwned(ctx context.Context, userID int64) ([]*domain.Diary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+prefixed(diaryCols, "d")+`, `+accessExpr+`, COALESCE(co.name, ''),
		       COALESCE(c.active, 0), COALESCE(c.done, 0)
		  FROM diaries d `+diaryCounts+`
		  LEFT JOIN companies co ON co.id = d.company_id
		 WHERE `+inMySpaces+` AND d.kind = 'regular'
		 ORDER BY d.position, d.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Diary{}
	for rows.Next() {
		var d domain.Diary
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.CompanyID, &d.TeamAccess, &d.Kind, &d.Name,
			&d.Position, &d.CreatedAt, &d.UpdatedAt, &d.MyAccess, &d.CompanyName,
			&d.ActiveCount, &d.DoneCount); err != nil {
			return nil, err
		}
		d.CanCheck = domain.AccessAtLeast(d.MyAccess, domain.AccessCheck)
		out = append(out, &d)
	}
	return out, rows.Err()
}

// ListShared — чужие ежедневники, открытые человеку адресно и не лежащие в его
// пространствах (вкладка «Поделились»), с именем/аватаром владельца.
func (r *Repo) ListShared(ctx context.Context, userID int64) ([]*domain.Diary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+prefixed(diaryCols, "d")+`,
		       u.fio, u.avatar_path, s.can_check,
		       COALESCE(c.active, 0), COALESCE(c.done, 0)
		  FROM diary_user_shares s
		  JOIN diaries d ON d.id = s.diary_id
		  JOIN users   u ON u.id = d.owner_id `+diaryCounts+`
		 WHERE s.user_id = $1 AND NOT `+inMySpaces+`
		 ORDER BY u.fio, d.name, d.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Diary{}
	for rows.Next() {
		var d domain.Diary
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.CompanyID, &d.TeamAccess, &d.Kind, &d.Name,
			&d.Position, &d.CreatedAt, &d.UpdatedAt,
			&d.OwnerName, &d.OwnerAvatar, &d.CanCheck,
			&d.ActiveCount, &d.DoneCount); err != nil {
			return nil, err
		}
		d.Shared = true
		d.MyAccess = domain.AccessView
		if d.CanCheck {
			d.MyAccess = domain.AccessCheck
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

// AccessOf — эффективный уровень человека к ежедневнику ("" — доступа нет).
func (r *Repo) AccessOf(ctx context.Context, diaryID, userID int64) (string, error) {
	var access string
	err := r.pool.QueryRow(ctx,
		`SELECT `+accessExpr+` FROM diaries d WHERE d.id = $2`, userID, diaryID).Scan(&access)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AccessNone, nil
	}
	return access, err
}

// Audience — кому адресовать события ежедневника: хозяин или участники
// команды-пространства плюс адресаты.
func (r *Repo) Audience(ctx context.Context, diaryID int64) ([]int64, error) {
	return r.ids(ctx, `
		SELECT owner_id FROM diaries WHERE id = $1 AND company_id IS NULL
		UNION
		SELECT uc.user_id FROM diaries d
		  JOIN user_companies uc ON uc.company_id = d.company_id
		 WHERE d.id = $1
		UNION
		SELECT user_id FROM diary_user_shares WHERE diary_id = $1`, diaryID)
}

// MyDay — скрытый ежедневник «Мой день»: заводится при первом обращении.
// ON CONFLICT по частичному уникальному индексу делает гонку двух вкладок
// безвредной — обе получат одну и ту же строку.
func (r *Repo) MyDay(ctx context.Context, ownerID int64) (*domain.Diary, error) {
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO diaries (owner_id, name, kind) VALUES ($1, 'Мой день', 'my_day')
		ON CONFLICT (owner_id) WHERE kind = 'my_day' DO NOTHING`, ownerID); err != nil {
		return nil, err
	}
	return scanDiary(r.pool.QueryRow(ctx,
		`SELECT `+diaryCols+` FROM diaries WHERE owner_id = $1 AND kind = 'my_day'`, ownerID))
}

func (r *Repo) MoveDiary(ctx context.Context, id, ownerID int64, companyID *int64, teamAccess string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE diaries SET owner_id = $2, company_id = $3, team_access = $4, updated_at = now()
		  WHERE id = $1`, id, ownerID, companyID, teamAccess)
	return err
}

func (r *Repo) ids(ctx context.Context, q string, args ...any) ([]int64, error) {
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

// prefixed — перечень колонок с алиасом таблицы.
func prefixed(cols, alias string) string {
	parts := strings.Split(cols, ",")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}

func (r *Repo) GetDiary(ctx context.Context, id int64) (*domain.Diary, error) {
	return scanDiary(r.pool.QueryRow(ctx, `SELECT `+diaryCols+` FROM diaries WHERE id = $1`, id))
}

// CountDiaries — сколько ежедневников завёл человек (лимит тарифа; скрытый
// «Мой день» не считается).
func (r *Repo) CountDiaries(ctx context.Context, owner_id int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM diaries WHERE owner_id = $1 AND kind = 'regular'`, owner_id).Scan(&n)
	return n, err
}

func (r *Repo) CreateDiary(ctx context.Context, d *domain.Diary) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO diaries (owner_id, company_id, team_access, name, position) VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, kind, created_at, updated_at`,
		d.OwnerID, d.CompanyID, d.TeamAccess, d.Name, d.Position).Scan(&d.ID, &d.Kind, &d.CreatedAt, &d.UpdatedAt)
}

func (r *Repo) UpdateDiary(ctx context.Context, id int64, name string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE diaries SET name = $2, updated_at = now() WHERE id = $1`, id, name)
	return err
}

func (r *Repo) DeleteDiary(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM diaries WHERE id = $1`, id)
	return err
}

func (r *Repo) NextPosition(ctx context.Context, ownerID int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(position), 0) + 1 FROM diaries WHERE owner_id = $1`, ownerID).Scan(&pos)
	return pos, err
}
