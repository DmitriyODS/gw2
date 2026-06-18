package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.RegistryRepository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func scanRegistry(row pgx.Row) (*domain.Registry, error) {
	var r domain.Registry
	err := row.Scan(&r.ID, &r.CompanyID, &r.Name, &r.Position, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

const registryCols = `id, company_id, name, position, created_by, created_at, updated_at`

func (r *Repo) ListRegistries(ctx context.Context, companyID int64) ([]*domain.Registry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+registryCols+` FROM registries WHERE company_id = $1 ORDER BY position, id`,
		companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Registry{}
	for rows.Next() {
		reg, err := scanRegistry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, reg)
	}
	return out, rows.Err()
}

func (r *Repo) GetRegistry(ctx context.Context, id int64) (*domain.Registry, error) {
	return scanRegistry(r.pool.QueryRow(ctx,
		`SELECT `+registryCols+` FROM registries WHERE id = $1`, id))
}

func (r *Repo) CreateRegistry(ctx context.Context, reg *domain.Registry) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO registries (company_id, name, position, created_by)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		reg.CompanyID, reg.Name, reg.Position, reg.CreatedBy).
		Scan(&reg.ID, &reg.CreatedAt, &reg.UpdatedAt)
}

func (r *Repo) UpdateRegistry(ctx context.Context, id int64, name string, position int) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE registries SET name = $2, position = $3, updated_at = now() WHERE id = $1`,
		id, name, position)
	return err
}

func (r *Repo) DeleteRegistry(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM registries WHERE id = $1`, id)
	return err
}

func (r *Repo) NextRegistryPosition(ctx context.Context, companyID int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(position), 0) + 1 FROM registries WHERE company_id = $1`,
		companyID).Scan(&pos)
	return pos, err
}

// ── Поля ─────────────────────────────────────────────────────────

const fieldCols = `id, registry_id, label, type, config, position, col_span, row_span, show_in_table, created_at`

func scanField(row pgx.Row) (domain.Field, error) {
	var f domain.Field
	err := row.Scan(&f.ID, &f.RegistryID, &f.Label, &f.Type, &f.Config,
		&f.Position, &f.ColSpan, &f.RowSpan, &f.ShowInTable, &f.CreatedAt)
	if f.Config == nil {
		f.Config = map[string]any{}
	}
	return f, err
}

func (r *Repo) ListFields(ctx context.Context, registryID int64) ([]domain.Field, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+fieldCols+` FROM registry_fields WHERE registry_id = $1 ORDER BY position, id`,
		registryID)
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

func (r *Repo) FieldsByRegistries(ctx context.Context, registryIDs []int64) (map[int64][]domain.Field, error) {
	out := map[int64][]domain.Field{}
	if len(registryIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+fieldCols+` FROM registry_fields WHERE registry_id = ANY($1) ORDER BY position, id`,
		registryIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		f, err := scanField(rows)
		if err != nil {
			return nil, err
		}
		out[f.RegistryID] = append(out[f.RegistryID], f)
	}
	return out, rows.Err()
}

// ReplaceFields — синхронизация набора полей в транзакции: поля с ID>0
// обновляются, ID==0 вставляются, отсутствующие в новом наборе — удаляются.
func (r *Repo) ReplaceFields(ctx context.Context, registryID int64, fields []domain.Field) ([]int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	existing := map[int64]bool{}
	rows, err := tx.Query(ctx, `SELECT id FROM registry_fields WHERE registry_id = $1`, registryID)
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
				`UPDATE registry_fields
				    SET label=$2, type=$3, config=$4, position=$5,
				        col_span=$6, row_span=$7, show_in_table=$8
				  WHERE id=$1`,
				f.ID, f.Label, f.Type, f.Config, f.Position, f.ColSpan, f.RowSpan, f.ShowInTable); err != nil {
				return nil, err
			}
			continue
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO registry_fields
			   (registry_id, label, type, config, position, col_span, row_span, show_in_table)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at`,
			registryID, f.Label, f.Type, f.Config, f.Position, f.ColSpan, f.RowSpan, f.ShowInTable).
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
		if _, err := tx.Exec(ctx, `DELETE FROM registry_fields WHERE id = ANY($1)`, removed); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE registries SET updated_at = now() WHERE id = $1`, registryID); err != nil {
		return nil, err
	}
	return removed, tx.Commit(ctx)
}
