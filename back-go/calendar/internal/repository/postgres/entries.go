package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
)

const entryCols = `id, calendar_id, event_at, data, created_by, created_at, updated_at`

func scanEntry(row pgx.Row) (*domain.Entry, error) {
	var e domain.Entry
	err := row.Scan(&e.ID, &e.CalendarID, &e.EventAt, &e.Data, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if e.Data == nil {
		e.Data = map[string]any{}
	}
	return &e, nil
}

// buildWhere — условие выборки записей по фильтру (диапазон + поиск).
// Возвращает строку WHERE и срез аргументов, начиная с calendar_id.
func buildWhere(f domain.EntryListFilter) (string, []any) {
	where := `WHERE calendar_id = $1`
	args := []any{f.CalendarID}
	if f.From != nil {
		args = append(args, *f.From)
		where += fmt.Sprintf(` AND event_at >= $%d`, len(args))
	}
	if f.To != nil {
		args = append(args, *f.To)
		where += fmt.Sprintf(` AND event_at < $%d`, len(args))
	}
	if f.Search != "" {
		args = append(args, f.Search)
		where += fmt.Sprintf(` AND search_text ILIKE '%%' || $%d || '%%'`, len(args))
	}
	return where, args
}

func (r *Repo) queryEntries(ctx context.Context, where string, args []any, limit int) ([]*domain.Entry, error) {
	q := `SELECT ` + entryCols + ` FROM calendar_records ` + where + ` ORDER BY event_at ASC, id ASC`
	if limit > 0 {
		args = append(args, limit)
		q += fmt.Sprintf(` LIMIT $%d`, len(args))
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Entry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repo) ListEntries(ctx context.Context, f domain.EntryListFilter) ([]*domain.Entry, error) {
	where, args := buildWhere(f)
	return r.queryEntries(ctx, where, args, f.Limit)
}

func (r *Repo) GetEntry(ctx context.Context, id int64) (*domain.Entry, error) {
	return scanEntry(r.pool.QueryRow(ctx,
		`SELECT `+entryCols+` FROM calendar_records WHERE id = $1`, id))
}

func (r *Repo) CreateEntry(ctx context.Context, e *domain.Entry, searchText string) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO calendar_records (calendar_id, event_at, data, search_text, created_by)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`,
		e.CalendarID, e.EventAt, e.Data, searchText, e.CreatedBy).
		Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
}

// UpdateEntry — обновить запись. eventAt: time.Time — поменять дату/время, nil —
// оставить прежней (используется при чистке данных удалённого поля).
func (r *Repo) UpdateEntry(ctx context.Context, id int64, eventAt any, data map[string]any, searchText string) error {
	if at, ok := eventAt.(time.Time); ok {
		_, err := r.pool.Exec(ctx,
			`UPDATE calendar_records SET event_at = $2, data = $3, search_text = $4, updated_at = now() WHERE id = $1`,
			id, at, data, searchText)
		return err
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE calendar_records SET data = $2, search_text = $3, updated_at = now() WHERE id = $1`,
		id, data, searchText)
	return err
}

func (r *Repo) DeleteEntry(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM calendar_records WHERE id = $1`, id)
	return err
}

func (r *Repo) DeleteEntries(ctx context.Context, calendarID int64, ids []int64) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM calendar_records WHERE calendar_id = $1 AND id = ANY($2)`, calendarID, ids)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *Repo) AllEntries(ctx context.Context, calendarID int64) ([]*domain.Entry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+entryCols+` FROM calendar_records WHERE calendar_id = $1`, calendarID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Entry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// EntriesForExport — записи для выгрузки: при непустом ids — только они (без
// учёта диапазона), иначе все по фильтру (диапазон дат + поиск).
func (r *Repo) EntriesForExport(ctx context.Context, f domain.EntryListFilter, ids []int64) ([]*domain.Entry, error) {
	if len(ids) > 0 {
		return r.queryEntries(ctx,
			`WHERE calendar_id = $1 AND id = ANY($2)`,
			[]any{f.CalendarID, ids}, f.Limit)
	}
	where, args := buildWhere(f)
	return r.queryEntries(ctx, where, args, f.Limit)
}
