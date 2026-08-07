// Package postgres — хранилище billingsvc (pgx, raw SQL).
package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

var (
	_ domain.CatalogRepository      = (*Repo)(nil)
	_ domain.SubscriptionRepository = (*Repo)(nil)
	_ domain.OrderRepository        = (*Repo)(nil)
	_ domain.PromoRepository        = (*Repo)(nil)
	_ domain.ProductRepository      = (*Repo)(nil)
	_ domain.AIRepository           = (*Repo)(nil)
	_ domain.StorageRepository      = (*Repo)(nil)
	_ domain.SettingsRepository     = (*Repo)(nil)
	_ domain.AuditRepository        = (*Repo)(nil)
)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// scanner — общий интерфейс pgx.Row/pgx.Rows для одного разбора строки.
type scanner interface{ Scan(dest ...any) error }

// noRows — пустая выборка это «не найдено», а не ошибка.
func noRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// jsonMap — jsonb-колонка в map (NULL и мусор дают пустую карту).
func jsonMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func jsonBytes(m map[string]any) []byte {
	if m == nil {
		return []byte("{}")
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return []byte("{}")
	}
	return raw
}

// ---------- Тарифы и аддоны ----------

const planColumns = `code, name, tagline, price_month, price_year, sort, is_active, updated_at`

func scanPlan(row scanner) (*domain.Plan, error) {
	var p domain.Plan
	if err := row.Scan(&p.Code, &p.Name, &p.Tagline, &p.PriceMonth, &p.PriceYear,
		&p.Sort, &p.IsActive, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repo) ListPlans(ctx context.Context, onlyActive bool) ([]*domain.Plan, error) {
	q := `SELECT ` + planColumns + ` FROM billing_plans`
	if onlyActive {
		q += ` WHERE is_active`
	}
	q += ` ORDER BY sort, code`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Plan{}
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repo) GetPlan(ctx context.Context, code string) (*domain.Plan, error) {
	p, err := scanPlan(r.pool.QueryRow(ctx, `SELECT `+planColumns+` FROM billing_plans WHERE code = $1`, code))
	if err != nil {
		if noRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *Repo) UpdatePlan(ctx context.Context, p *domain.Plan) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE billing_plans
		   SET name = $2, tagline = $3, price_month = $4, price_year = $5,
		       sort = $6, is_active = $7, updated_at = now()
		 WHERE code = $1`,
		p.Code, p.Name, p.Tagline, p.PriceMonth, p.PriceYear, p.Sort, p.IsActive)
	return err
}

const addonColumns = `code, kind, name, description, amount, price_month, price_year, recurring, sort, is_active`

func scanAddon(row scanner) (*domain.Addon, error) {
	var a domain.Addon
	if err := row.Scan(&a.Code, &a.Kind, &a.Name, &a.Description, &a.Amount,
		&a.PriceMonth, &a.PriceYear, &a.Recurring, &a.Sort, &a.IsActive); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repo) ListAddons(ctx context.Context, onlyActive bool) ([]*domain.Addon, error) {
	q := `SELECT ` + addonColumns + ` FROM billing_addons`
	if onlyActive {
		q += ` WHERE is_active`
	}
	q += ` ORDER BY sort, code`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Addon{}
	for rows.Next() {
		a, err := scanAddon(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repo) GetAddon(ctx context.Context, code string) (*domain.Addon, error) {
	a, err := scanAddon(r.pool.QueryRow(ctx, `SELECT `+addonColumns+` FROM billing_addons WHERE code = $1`, code))
	if err != nil {
		if noRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

func (r *Repo) UpdateAddon(ctx context.Context, a *domain.Addon) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE billing_addons
		   SET name = $2, description = $3, amount = $4, price_month = $5,
		       price_year = $6, recurring = $7, sort = $8, is_active = $9, updated_at = now()
		 WHERE code = $1`,
		a.Code, a.Name, a.Description, a.Amount, a.PriceMonth, a.PriceYear,
		a.Recurring, a.Sort, a.IsActive)
	return err
}

// ---------- Настройки платформы ----------

func (r *Repo) GetSettings(ctx context.Context) (*domain.Settings, error) {
	var s domain.Settings
	err := r.pool.QueryRow(ctx, `
		SELECT commission_pct, payment_provider, payment_enabled, store_enabled
		  FROM billing_settings WHERE id = 1`).
		Scan(&s.CommissionPct, &s.PaymentProvider, &s.PaymentEnabled, &s.StoreEnabled)
	if err != nil {
		if noRows(err) {
			return &domain.Settings{CommissionPct: 10, PaymentProvider: "manual", StoreEnabled: true}, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repo) UpdateSettings(ctx context.Context, s *domain.Settings) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO billing_settings (id, commission_pct, payment_provider, payment_enabled, store_enabled, updated_at)
		VALUES (1, $1, $2, $3, $4, now())
		ON CONFLICT (id) DO UPDATE SET commission_pct = EXCLUDED.commission_pct,
		    payment_provider = EXCLUDED.payment_provider,
		    payment_enabled = EXCLUDED.payment_enabled,
		    store_enabled = EXCLUDED.store_enabled,
		    updated_at = now()`,
		s.CommissionPct, s.PaymentProvider, s.PaymentEnabled, s.StoreEnabled)
	return err
}

// ---------- Журнал действий ----------

func (r *Repo) LogAction(ctx context.Context, e *domain.AuditEntry) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO platform_audit_log (actor_id, action, target_kind, target_id, summary, payload)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		e.ActorID, e.Action, e.TargetKind, e.TargetID, e.Summary, jsonBytes(e.Payload))
	return err
}

func (r *Repo) ListAudit(ctx context.Context, action string, limit, offset int) ([]*domain.AuditEntry, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT l.id, l.actor_id, COALESCE(u.fio, ''), l.action, l.target_kind, l.target_id,
		       l.summary, l.payload, l.created_at, count(*) OVER () AS total
		  FROM platform_audit_log l
		  LEFT JOIN users u ON u.id = l.actor_id
		 WHERE ($1 = '' OR l.action = $1)
		 ORDER BY l.created_at DESC, l.id DESC
		 LIMIT $2 OFFSET $3`, action, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*domain.AuditEntry{}
	total := 0
	for rows.Next() {
		var e domain.AuditEntry
		var payload []byte
		if err := rows.Scan(&e.ID, &e.ActorID, &e.ActorName, &e.Action, &e.TargetKind,
			&e.TargetID, &e.Summary, &payload, &e.CreatedAt, &total); err != nil {
			return nil, 0, err
		}
		e.Payload = jsonMap(payload)
		out = append(out, &e)
	}
	return out, total, rows.Err()
}
