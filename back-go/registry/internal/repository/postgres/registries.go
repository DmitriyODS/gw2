package postgres

import (
	"context"
	"errors"
	"strings"

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
	err := row.Scan(&r.ID, &r.OwnerID, &r.CompanyID, &r.Name, &r.Position,
		&r.SectionFieldID, &r.Accounting, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

const registryCols = `id, owner_id, company_id, name, position, section_field_id,
	accounting, created_by, created_at, updated_at`

/*
accessExpr — эффективный уровень доступа одним выражением.

	Уровень приходит человеку тремя путями сразу (он владелец, ему выдали лично,
	выдали его компании), и брать нужно СИЛЬНЕЙШИЙ. Порядок уровней задан здесь
	числом, а не сравнением строк: 'view' > 'edit' лексикографически, и наивный
	MAX(access) молча понижал бы права. Держать в паре с domain/access.go.
*/
const accessExpr = `
	CASE WHEN reg.owner_id = $1 THEN 'owner' ELSE COALESCE((
		SELECT CASE max(CASE sh.access
		            WHEN 'admin' THEN 3 WHEN 'edit' THEN 2 ELSE 1 END)
		         WHEN 3 THEN 'admin' WHEN 2 THEN 'edit' WHEN 1 THEN 'view' END
		  FROM registry_user_shares sh
		 WHERE sh.registry_id = reg.id
		   AND (sh.user_id = $1 OR sh.company_id = ANY($2))
	), '') END`

// scopeCondition — условие вкладки раздела.
func scopeCondition(scope string) string {
	switch scope {
	case domain.ScopeMine:
		return `reg.owner_id = $1`
	case domain.ScopeShared:
		return `EXISTS (SELECT 1 FROM registry_user_shares sh
		                 WHERE sh.registry_id = reg.id AND sh.user_id = $1)
		        AND reg.owner_id <> $1`
	case domain.ScopeCompany:
		return `EXISTS (SELECT 1 FROM registry_user_shares sh
		                 WHERE sh.registry_id = reg.id AND sh.company_id = ANY($2))
		        AND reg.owner_id <> $1`
	default:
		return `(reg.owner_id = $1
		         OR EXISTS (SELECT 1 FROM registry_user_shares sh
		                     WHERE sh.registry_id = reg.id
		                       AND (sh.user_id = $1 OR sh.company_id = ANY($2))))`
	}
}

// ListRegistries — реестры выбранной области вместе с уровнем доступа и именем
// владельца (вкладки «Поделились» и «Компания» обязаны называть хозяина).
func (r *Repo) ListRegistries(ctx context.Context, userID int64, companyIDs []int64, scope string) ([]*domain.Registry, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+prefixed(registryCols, "reg")+`, `+accessExpr+`, COALESCE(u.fio, '')
		  FROM registries reg
		  LEFT JOIN users u ON u.id = reg.owner_id
		 WHERE `+scopeCondition(scope)+`
		 ORDER BY reg.position, reg.id`, userID, companyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Registry{}
	for rows.Next() {
		var reg domain.Registry
		if err := rows.Scan(&reg.ID, &reg.OwnerID, &reg.CompanyID, &reg.Name, &reg.Position,
			&reg.SectionFieldID, &reg.Accounting, &reg.CreatedBy, &reg.CreatedAt, &reg.UpdatedAt,
			&reg.MyAccess, &reg.OwnerName); err != nil {
			return nil, err
		}
		out = append(out, &reg)
	}
	return out, rows.Err()
}

func (r *Repo) GetRegistry(ctx context.Context, id int64) (*domain.Registry, error) {
	return scanRegistry(r.pool.QueryRow(ctx,
		`SELECT `+registryCols+` FROM registries WHERE id = $1`, id))
}

// CountOwned — сколько реестров завёл человек (лимит тарифа).
func (r *Repo) CountOwned(ctx context.Context, ownerID int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM registries WHERE owner_id = $1`, ownerID).Scan(&n)
	return n, err
}

func (r *Repo) CreateRegistry(ctx context.Context, reg *domain.Registry) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO registries (owner_id, company_id, name, position, accounting, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`,
		reg.OwnerID, reg.CompanyID, reg.Name, reg.Position, reg.Accounting, reg.CreatedBy).
		Scan(&reg.ID, &reg.CreatedAt, &reg.UpdatedAt)
}

func (r *Repo) UpdateRegistry(ctx context.Context, id int64, name string, position int, sectionFieldID *int64, accounting bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE registries
		    SET name = $2, position = $3, section_field_id = $4, accounting = $5, updated_at = now()
		  WHERE id = $1`,
		id, name, position, sectionFieldID, accounting)
	return err
}

func (r *Repo) DeleteRegistry(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM registries WHERE id = $1`, id)
	return err
}

func (r *Repo) NextRegistryPosition(ctx context.Context, ownerID int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(position), 0) + 1 FROM registries WHERE owner_id = $1`,
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
