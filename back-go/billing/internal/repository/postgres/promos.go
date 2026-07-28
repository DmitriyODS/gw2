package postgres

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

const promoColumns = `id, code, kind, value, plan_code, applies_to, max_uses,
	per_user_limit, used_count, starts_at, expires_at, is_active, comment, created_at`

func scanPromo(row scanner) (*domain.Promo, error) {
	var p domain.Promo
	if err := row.Scan(&p.ID, &p.Code, &p.Kind, &p.Value, &p.PlanCode, &p.AppliesTo,
		&p.MaxUses, &p.PerUserLimit, &p.UsedCount, &p.StartsAt, &p.ExpiresAt,
		&p.IsActive, &p.Comment, &p.CreatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repo) ListPromos(ctx context.Context) ([]*domain.Promo, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+promoColumns+` FROM billing_promos ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Promo{}
	for rows.Next() {
		p, err := scanPromo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repo) GetPromo(ctx context.Context, id int64) (*domain.Promo, error) {
	p, err := scanPromo(r.pool.QueryRow(ctx, `SELECT `+promoColumns+` FROM billing_promos WHERE id = $1`, id))
	if err != nil {
		if noRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *Repo) GetPromoByCode(ctx context.Context, code string) (*domain.Promo, error) {
	p, err := scanPromo(r.pool.QueryRow(ctx,
		`SELECT `+promoColumns+` FROM billing_promos WHERE upper(code) = upper($1)`, code))
	if err != nil {
		if noRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *Repo) CreatePromo(ctx context.Context, p *domain.Promo) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO billing_promos
		    (code, kind, value, plan_code, applies_to, max_uses, per_user_limit,
		     starts_at, expires_at, is_active, comment, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, used_count, created_at`,
		p.Code, p.Kind, p.Value, p.PlanCode, p.AppliesTo, p.MaxUses, p.PerUserLimit,
		p.StartsAt, p.ExpiresAt, p.IsActive, p.Comment, nil).
		Scan(&p.ID, &p.UsedCount, &p.CreatedAt)
}

func (r *Repo) UpdatePromo(ctx context.Context, p *domain.Promo) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE billing_promos
		   SET kind = $2, value = $3, plan_code = $4, applies_to = $5, max_uses = $6,
		       per_user_limit = $7, starts_at = $8, expires_at = $9, is_active = $10, comment = $11
		 WHERE id = $1`,
		p.ID, p.Kind, p.Value, p.PlanCode, p.AppliesTo, p.MaxUses, p.PerUserLimit,
		p.StartsAt, p.ExpiresAt, p.IsActive, p.Comment)
	return err
}

func (r *Repo) DeletePromo(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM billing_promos WHERE id = $1`, id)
	return err
}

func (r *Repo) CountRedemptions(ctx context.Context, promoID, userID int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM billing_promo_redemptions WHERE promo_id = $1 AND user_id = $2`,
		promoID, userID).Scan(&n)
	return n, err
}

// Redeem — атомарная активация: счётчик наращивается только пока не выбран
// общий лимит, запись активации ложится в той же транзакции.
func (r *Repo) Redeem(ctx context.Context, promoID, userID int64, orderID *int64) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE billing_promos
		   SET used_count = used_count + 1
		 WHERE id = $1 AND is_active
		   AND (max_uses = 0 OR used_count < max_uses)
		   AND (starts_at IS NULL OR starts_at <= now())
		   AND (expires_at IS NULL OR expires_at > now())
		   AND (SELECT count(*) FROM billing_promo_redemptions rd
		         WHERE rd.promo_id = billing_promos.id AND rd.user_id = $2) < per_user_limit`,
		promoID, userID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO billing_promo_redemptions (promo_id, user_id, order_id) VALUES ($1, $2, $3)`,
		promoID, userID, orderID); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}
