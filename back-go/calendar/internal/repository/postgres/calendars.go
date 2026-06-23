package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.CalendarRepository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func scanCalendar(row pgx.Row) (*domain.Calendar, error) {
	var c domain.Calendar
	err := row.Scan(&c.ID, &c.CompanyID, &c.Name, &c.Position, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

const calendarCols = `id, company_id, name, position, created_by, created_at, updated_at`

func (r *Repo) ListCalendars(ctx context.Context, companyID int64) ([]*domain.Calendar, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+calendarCols+` FROM calendars WHERE company_id = $1 ORDER BY position, id`,
		companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Calendar{}
	for rows.Next() {
		cal, err := scanCalendar(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cal)
	}
	return out, rows.Err()
}

func (r *Repo) GetCalendar(ctx context.Context, id int64) (*domain.Calendar, error) {
	return scanCalendar(r.pool.QueryRow(ctx,
		`SELECT `+calendarCols+` FROM calendars WHERE id = $1`, id))
}

func (r *Repo) CreateCalendar(ctx context.Context, cal *domain.Calendar) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO calendars (company_id, name, position, created_by)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		cal.CompanyID, cal.Name, cal.Position, cal.CreatedBy).
		Scan(&cal.ID, &cal.CreatedAt, &cal.UpdatedAt)
}

func (r *Repo) UpdateCalendar(ctx context.Context, id int64, name string, position int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE calendars SET name = $2, position = $3, updated_at = now() WHERE id = $1`,
		id, name, position)
	return err
}

func (r *Repo) DeleteCalendar(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM calendars WHERE id = $1`, id)
	return err
}

func (r *Repo) NextCalendarPosition(ctx context.Context, companyID int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(position), 0) + 1 FROM calendars WHERE company_id = $1`,
		companyID).Scan(&pos)
	return pos, err
}

// ── Поля ─────────────────────────────────────────────────────────

const fieldCols = `id, calendar_id, label, type, config, position, col_span, row_span,
	show_in_table, visible_field_id, visible_value, created_at`

func scanField(row pgx.Row) (domain.Field, error) {
	var f domain.Field
	err := row.Scan(&f.ID, &f.CalendarID, &f.Label, &f.Type, &f.Config,
		&f.Position, &f.ColSpan, &f.RowSpan, &f.ShowInTable,
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
				        col_span=$6, row_span=$7, show_in_table=$8,
				        visible_field_id=$9, visible_value=$10
				  WHERE id=$1`,
				f.ID, f.Label, f.Type, f.Config, f.Position, f.ColSpan, f.RowSpan, f.ShowInTable,
				f.VisibleFieldID, f.VisibleValue); err != nil {
				return nil, err
			}
			continue
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO calendar_fields
			   (calendar_id, label, type, config, position, col_span, row_span,
			    show_in_table, visible_field_id, visible_value)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id, created_at`,
			calendarID, f.Label, f.Type, f.Config, f.Position, f.ColSpan, f.RowSpan, f.ShowInTable,
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
