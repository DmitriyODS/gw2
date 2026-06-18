package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

const recordCols = `id, registry_id, data, created_by, created_at, updated_at`

func scanRecord(row pgx.Row) (*domain.Record, error) {
	var r domain.Record
	err := row.Scan(&r.ID, &r.RegistryID, &r.Data, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if r.Data == nil {
		r.Data = map[string]any{}
	}
	return &r, nil
}

// orderBy — выражение сортировки. Поле по data->>'<id>' с приведением типа;
// id поля — int64 (проверен на уровне домена), поэтому безопасно встраивается.
func orderBy(f domain.RecordListFilter) string {
	dir := "ASC"
	if f.Desc {
		dir = "DESC"
	}
	if f.SortFieldID <= 0 {
		return "created_at " + dir + ", id " + dir
	}
	key := fmt.Sprintf("data->>'%d'", f.SortFieldID)
	switch f.SortKind {
	case "number":
		return fmt.Sprintf("NULLIF(%s,'')::numeric %s NULLS LAST, id ASC", key, dir)
	case "date":
		return fmt.Sprintf("%s %s NULLS LAST, id ASC", key, dir)
	default:
		return fmt.Sprintf("lower(%s) %s NULLS LAST, id ASC", key, dir)
	}
}

func (r *Repo) ListRecords(ctx context.Context, f domain.RecordListFilter) ([]*domain.Record, int, error) {
	where := `WHERE registry_id = $1`
	args := []any{f.RegistryID}
	if f.Search != "" {
		where += ` AND search_text ILIKE '%' || $2 || '%'`
		args = append(args, f.Search)
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM registry_records `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.PerPage
	if limit <= 0 {
		limit = 30
	}
	offset := (f.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	q := `SELECT ` + recordCols + ` FROM registry_records ` + where +
		` ORDER BY ` + orderBy(f) +
		fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*domain.Record{}
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, rec)
	}
	return out, total, rows.Err()
}

func (r *Repo) GetRecord(ctx context.Context, id int64) (*domain.Record, error) {
	return scanRecord(r.pool.QueryRow(ctx,
		`SELECT `+recordCols+` FROM registry_records WHERE id = $1`, id))
}

func (r *Repo) CreateRecord(ctx context.Context, rec *domain.Record, searchText string) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO registry_records (registry_id, data, search_text, created_by)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		rec.RegistryID, rec.Data, searchText, rec.CreatedBy).
		Scan(&rec.ID, &rec.CreatedAt, &rec.UpdatedAt)
}

func (r *Repo) UpdateRecord(ctx context.Context, id int64, data map[string]any, searchText string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE registry_records SET data = $2, search_text = $3, updated_at = now() WHERE id = $1`,
		id, data, searchText)
	return err
}

func (r *Repo) DeleteRecord(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM registry_records WHERE id = $1`, id)
	return err
}

func (r *Repo) DeleteRecords(ctx context.Context, registryID int64, ids []int64) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM registry_records WHERE registry_id = $1 AND id = ANY($2)`, registryID, ids)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *Repo) RecordsForExport(ctx context.Context, registryID int64, search string, ids []int64) ([]*domain.Record, error) {
	where := `WHERE registry_id = $1`
	args := []any{registryID}
	if len(ids) > 0 {
		where += ` AND id = ANY($2)`
		args = append(args, ids)
	} else if search != "" {
		where += ` AND search_text ILIKE '%' || $2 || '%'`
		args = append(args, search)
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+recordCols+` FROM registry_records `+where+` ORDER BY created_at DESC, id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Record{}
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *Repo) AllRecords(ctx context.Context, registryID int64) ([]*domain.Record, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+recordCols+` FROM registry_records WHERE registry_id = $1`, registryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Record{}
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
