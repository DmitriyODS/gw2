package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/spaces"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.CalendarRepository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func scanCalendar(row pgx.Row) (*domain.Calendar, error) {
	var c domain.Calendar
	err := row.Scan(&c.ID, &c.OwnerID, &c.CompanyID, &c.TeamAccess, &c.Name, &c.Position,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

const calendarCols = `id, owner_id, company_id, team_access, name, position, created_by, created_at, updated_at`

/*
accessExpr — эффективный уровень человека $1 к календарю cal.

	Хозяин личного календаря, а у календаря команды — автор и администраторы
	(пока состоят в ней) распоряжаются им целиком; остальные участники команды
	получают team_access. Держать в паре с domain/access.go.
*/
var accessExpr = `
	CASE
	  WHEN cal.company_id IS NULL THEN CASE WHEN cal.owner_id = $1 THEN 'owner' ELSE '' END
	  WHEN ` + spaces.Admin("$1", "cal.company_id") + ` THEN 'owner'
	  WHEN ` + spaces.Member("$1", "cal.company_id") + `
	       THEN CASE WHEN cal.owner_id = $1 THEN 'owner' ELSE cal.team_access END
	  ELSE ''
	END`

// visibleCond — календарь лежит в пространстве человека $1: личный его либо в
// одной из его команд.
var visibleCond = `((cal.company_id IS NULL AND cal.owner_id = $1)
	OR cal.company_id IN (` + spaces.MyTeams("$1") + `))`

func (r *Repo) ListCalendars(ctx context.Context, userID int64) ([]*domain.Calendar, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+prefixed(calendarCols, "cal")+`, `+accessExpr+`, COALESCE(c.name, '')
		  FROM calendars cal
		  LEFT JOIN companies c ON c.id = cal.company_id
		 WHERE `+visibleCond+`
		 ORDER BY cal.position, cal.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Calendar{}
	for rows.Next() {
		var c domain.Calendar
		if err := rows.Scan(&c.ID, &c.OwnerID, &c.CompanyID, &c.TeamAccess, &c.Name, &c.Position,
			&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt, &c.MyAccess, &c.CompanyName); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *Repo) GetCalendar(ctx context.Context, id int64) (*domain.Calendar, error) {
	return scanCalendar(r.pool.QueryRow(ctx,
		`SELECT `+calendarCols+` FROM calendars WHERE id = $1`, id))
}

func (r *Repo) AccessOf(ctx context.Context, calendarID, userID int64) (string, error) {
	var access string
	err := r.pool.QueryRow(ctx,
		`SELECT `+accessExpr+` FROM calendars cal WHERE cal.id = $2`, userID, calendarID).Scan(&access)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AccessNone, nil
	}
	return access, err
}

// Audience — хозяин личного календаря либо все участники его команды.
func (r *Repo) Audience(ctx context.Context, calendarID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT owner_id FROM calendars WHERE id = $1 AND company_id IS NULL
		UNION
		SELECT uc.user_id FROM calendars cal
		  JOIN user_companies uc ON uc.company_id = cal.company_id
		 WHERE cal.id = $1`, calendarID)
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

// CountOwned — сколько календарей завёл человек (лимит тарифа).
func (r *Repo) CountOwned(ctx context.Context, ownerID int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM calendars WHERE owner_id = $1`, ownerID).Scan(&n)
	return n, err
}

func (r *Repo) CreateCalendar(ctx context.Context, cal *domain.Calendar) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO calendars (owner_id, company_id, team_access, name, position, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`,
		cal.OwnerID, cal.CompanyID, cal.TeamAccess, cal.Name, cal.Position, cal.CreatedBy).
		Scan(&cal.ID, &cal.CreatedAt, &cal.UpdatedAt)
}

func (r *Repo) UpdateCalendar(ctx context.Context, id int64, name string, position int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE calendars SET name = $2, position = $3, updated_at = now() WHERE id = $1`,
		id, name, position)
	return err
}

func (r *Repo) MoveCalendar(ctx context.Context, id, ownerID int64, companyID *int64, teamAccess string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE calendars
		    SET owner_id = $2, company_id = $3, team_access = $4, updated_at = now()
		  WHERE id = $1`,
		id, ownerID, companyID, teamAccess)
	return err
}

func (r *Repo) DeleteCalendar(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM calendars WHERE id = $1`, id)
	return err
}

func (r *Repo) NextCalendarPosition(ctx context.Context, ownerID int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(position), 0) + 1 FROM calendars WHERE owner_id = $1`,
		ownerID).Scan(&pos)
	return pos, err
}

// prefixed — перечень колонок с алиасом таблицы: список полей один, а запросы
// с JOIN требуют квалификации.
func prefixed(cols, alias string) string {
	parts := strings.Split(cols, ",")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}

// ── Поля ─────────────────────────────────────────────────────────

const fieldCols = `id, calendar_id, label, type, config, position, col_span, row_span,
	show_in_table, show_in_card, visible_field_id, visible_value, created_at`

func scanField(row pgx.Row) (domain.Field, error) {
	var f domain.Field
	err := row.Scan(&f.ID, &f.CalendarID, &f.Label, &f.Type, &f.Config,
		&f.Position, &f.ColSpan, &f.RowSpan, &f.ShowInTable, &f.ShowInCard,
		&f.VisibleFieldID, &f.VisibleValue, &f.CreatedAt)
	if f.Config == nil {
		f.Config = map[string]any{}
	}
	return f, err
}

func (r *Repo) ListFields(ctx context.Context, calendarID int64) ([]domain.Field, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+fieldCols+` FROM calendar_fields WHERE calendar_id = $1 ORDER BY position, id`,
		calendarID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Field{}
	for rows.Next() {
		f, err := scanField(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repo) FieldsByCalendars(ctx context.Context, calendarIDs []int64) (map[int64][]domain.Field, error) {
	out := map[int64][]domain.Field{}
	if len(calendarIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+fieldCols+` FROM calendar_fields WHERE calendar_id = ANY($1) ORDER BY position, id`,
		calendarIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		f, err := scanField(rows)
		if err != nil {
			return nil, err
		}
		out[f.CalendarID] = append(out[f.CalendarID], f)
	}
	return out, rows.Err()
}

// ReplaceFields — синхронизация набора полей в транзакции: поля с ID>0
// обновляются, ID==0 вставляются, отсутствующие в новом наборе — удаляются.
func (r *Repo) ReplaceFields(ctx context.Context, calendarID int64, fields []domain.Field) ([]int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	existing := map[int64]bool{}
	rows, err := tx.Query(ctx, `SELECT id FROM calendar_fields WHERE calendar_id = $1`, calendarID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		existing[id] = true
	}
	rows.Close()

	keep := map[int64]bool{}
	for i := range fields {
		f := &fields[i]
		f.Position = i
		if f.ID > 0 && existing[f.ID] {
			keep[f.ID] = true
			if _, err := tx.Exec(ctx,
				`UPDATE calendar_fields
				    SET label=$2, type=$3, config=$4, position=$5,
				        col_span=$6, row_span=$7, show_in_table=$8, show_in_card=$9,
				        visible_field_id=$10, visible_value=$11
				  WHERE id=$1`,
				f.ID, f.Label, f.Type, f.Config, f.Position, f.ColSpan, f.RowSpan, f.ShowInTable, f.ShowInCard,
				f.VisibleFieldID, f.VisibleValue); err != nil {
				return nil, err
			}
			continue
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO calendar_fields
			   (calendar_id, label, type, config, position, col_span, row_span,
			    show_in_table, show_in_card, visible_field_id, visible_value)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id, created_at`,
			calendarID, f.Label, f.Type, f.Config, f.Position, f.ColSpan, f.RowSpan, f.ShowInTable, f.ShowInCard,
			f.VisibleFieldID, f.VisibleValue).
			Scan(&f.ID, &f.CreatedAt); err != nil {
			return nil, err
		}
	}

	removed := []int64{}
	for id := range existing {
		if !keep[id] {
			removed = append(removed, id)
		}
	}
	if len(removed) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM calendar_fields WHERE id = ANY($1)`, removed); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE calendars SET updated_at = now() WHERE id = $1`, calendarID); err != nil {
		return nil, err
	}
	return removed, tx.Commit(ctx)
}
