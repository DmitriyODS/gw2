package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
)

const entryCols = `id, diary_id, entry_date, start_min, end_min, title, description,
	done, linked_task_id, created_at, updated_at`

func scanEntry(row pgx.Row) (*domain.Entry, error) {
	var e domain.Entry
	err := row.Scan(&e.ID, &e.DiaryID, &e.Date, &e.StartMin, &e.EndMin, &e.Title,
		&e.Description, &e.Done, &e.LinkedTaskID, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// buildWhere — условие выборки записей по фильтру (вкладка done + диапазон дат +
// поиск). Возвращает WHERE и аргументы, начиная с diary_id.
func buildWhere(f domain.EntryListFilter) (string, []any) {
	where := `WHERE diary_id = $1 AND done = $2`
	args := []any{f.DiaryID, f.Archived}
	if f.From != nil {
		args = append(args, *f.From)
		where += fmt.Sprintf(` AND entry_date >= $%d::date`, len(args))
	}
	if f.To != nil {
		args = append(args, *f.To)
		where += fmt.Sprintf(` AND entry_date < $%d::date`, len(args))
	}
	if f.Search != "" {
		args = append(args, f.Search)
		where += fmt.Sprintf(` AND search_text ILIKE '%%' || $%d || '%%'`, len(args))
	}
	return where, args
}

// orderBy — активные сортируем по дню и времени начала (без времени — первыми),
// архив — свежие выполненные сверху.
func orderBy(archived bool) string {
	if archived {
		return ` ORDER BY entry_date DESC, id DESC`
	}
	return ` ORDER BY entry_date ASC, COALESCE(start_min, -1) ASC, id ASC`
}

func (r *Repo) queryEntries(ctx context.Context, where, order string, args []any, limit int) ([]*domain.Entry, error) {
	q := `SELECT ` + entryCols + ` FROM diary_records ` + where + order
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
	return r.queryEntries(ctx, where, orderBy(f.Archived), args, f.Limit)
}

func (r *Repo) GetEntry(ctx context.Context, id int64) (*domain.Entry, error) {
	return scanEntry(r.pool.QueryRow(ctx, `SELECT `+entryCols+` FROM diary_records WHERE id = $1`, id))
}

func (r *Repo) CreateEntry(ctx context.Context, e *domain.Entry, searchText string) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO diary_records (diary_id, entry_date, start_min, end_min, title, description, search_text)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, done, linked_task_id, created_at, updated_at`,
		e.DiaryID, e.Date, e.StartMin, e.EndMin, e.Title, e.Description, searchText).
		Scan(&e.ID, &e.Done, &e.LinkedTaskID, &e.CreatedAt, &e.UpdatedAt)
}

func (r *Repo) UpdateEntry(ctx context.Context, e *domain.Entry, searchText string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE diary_records
		    SET entry_date = $2, start_min = $3, end_min = $4, title = $5,
		        description = $6, search_text = $7, updated_at = now()
		  WHERE id = $1`,
		e.ID, e.Date, e.StartMin, e.EndMin, e.Title, e.Description, searchText)
	return err
}

func (r *Repo) SetEntryDone(ctx context.Context, id int64, done bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE diary_records SET done = $2, updated_at = now() WHERE id = $1`, id, done)
	return err
}

func (r *Repo) SetEntryTask(ctx context.Context, id int64, taskID *int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE diary_records SET linked_task_id = $2, updated_at = now() WHERE id = $1`, id, taskID)
	return err
}

func (r *Repo) DeleteEntry(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM diary_records WHERE id = $1`, id)
	return err
}

func (r *Repo) DeleteEntries(ctx context.Context, diaryID int64, ids []int64) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM diary_records WHERE diary_id = $1 AND id = ANY($2)`, diaryID, ids)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *Repo) EntriesForExport(ctx context.Context, f domain.EntryListFilter, ids []int64) ([]*domain.Entry, error) {
	if len(ids) > 0 {
		return r.queryEntries(ctx,
			`WHERE diary_id = $1 AND id = ANY($2)`, orderBy(f.Archived),
			[]any{f.DiaryID, ids}, f.Limit)
	}
	where, args := buildWhere(f)
	return r.queryEntries(ctx, where, orderBy(f.Archived), args, f.Limit)
}
