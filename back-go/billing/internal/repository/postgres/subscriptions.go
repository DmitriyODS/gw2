package postgres

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

const subColumns = `user_id, plan_code, period, source, started_at, expires_at,
	auto_renew, cancelled_at, note`

func scanSubscription(row scanner) (*domain.Subscription, error) {
	var s domain.Subscription
	if err := row.Scan(&s.UserID, &s.PlanCode, &s.Period, &s.Source, &s.StartedAt,
		&s.ExpiresAt, &s.AutoRenew, &s.CancelledAt, &s.Note); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) GetSubscription(ctx context.Context, userID int64) (*domain.Subscription, error) {
	s, err := scanSubscription(r.pool.QueryRow(ctx,
		`SELECT `+subColumns+` FROM billing_subscriptions WHERE user_id = $1`, userID))
	if err != nil {
		if noRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

func (r *Repo) GetSubscriptions(ctx context.Context, userIDs []int64) (map[int64]*domain.Subscription, error) {
	out := map[int64]*domain.Subscription{}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+subColumns+` FROM billing_subscriptions WHERE user_id = ANY($1)`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		s, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out[s.UserID] = s
	}
	return out, rows.Err()
}

func (r *Repo) SaveSubscription(ctx context.Context, s *domain.Subscription) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO billing_subscriptions
		    (user_id, plan_code, period, source, started_at, expires_at, auto_renew, cancelled_at, note, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		ON CONFLICT (user_id) DO UPDATE SET plan_code = EXCLUDED.plan_code,
		    period = EXCLUDED.period, source = EXCLUDED.source, started_at = EXCLUDED.started_at,
		    expires_at = EXCLUDED.expires_at, auto_renew = EXCLUDED.auto_renew,
		    cancelled_at = EXCLUDED.cancelled_at, note = EXCLUDED.note, updated_at = now()`,
		s.UserID, s.PlanCode, s.Period, s.Source, s.StartedAt, s.ExpiresAt,
		s.AutoRenew, s.CancelledAt, s.Note)
	return err
}

func (r *Repo) DeleteSubscription(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM billing_subscriptions WHERE user_id = $1`, userID)
	return err
}

func (r *Repo) ListSubscriptions(ctx context.Context, search, plan string, limit, offset int) ([]*domain.Subscription, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.user_id, s.plan_code, s.period, s.source, s.started_at, s.expires_at,
		       s.auto_renew, s.cancelled_at, s.note, count(*) OVER () AS total
		  FROM billing_subscriptions s
		  JOIN users u ON u.id = s.user_id
		 WHERE ($1 = '' OR u.fio ILIKE '%' || $1 || '%' OR u.login ILIKE '%' || $1 || '%')
		   AND ($2 = '' OR s.plan_code = $2)
		 ORDER BY s.updated_at DESC
		 LIMIT $3 OFFSET $4`, search, plan, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*domain.Subscription{}
	total := 0
	for rows.Next() {
		var s domain.Subscription
		if err := rows.Scan(&s.UserID, &s.PlanCode, &s.Period, &s.Source, &s.StartedAt,
			&s.ExpiresAt, &s.AutoRenew, &s.CancelledAt, &s.Note, &total); err != nil {
			return nil, 0, err
		}
		out = append(out, &s)
	}
	return out, total, rows.Err()
}

func (r *Repo) DueRenewals(ctx context.Context, now time.Time, limit int) ([]*domain.Subscription, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+subColumns+`
		  FROM billing_subscriptions
		 WHERE expires_at IS NOT NULL AND expires_at <= $1
		 ORDER BY expires_at
		 LIMIT $2`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Subscription{}
	for rows.Next() {
		s, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---------- Купленные аддоны ----------

const userAddonColumns = `a.id, a.user_id, a.addon_code, a.kind, a.amount, a.qty,
	a.company_id, a.period, a.started_at, a.expires_at, a.auto_renew, COALESCE(c.name, a.addon_code)`

func scanUserAddon(row scanner) (*domain.UserAddon, error) {
	var a domain.UserAddon
	if err := row.Scan(&a.ID, &a.UserID, &a.AddonCode, &a.Kind, &a.Amount, &a.Qty,
		&a.CompanyID, &a.Period, &a.StartedAt, &a.ExpiresAt, &a.AutoRenew, &a.Name); err != nil {
		return nil, err
	}
	return &a, nil
}

// activeAddons — действующие аддоны (не отменённые и не истёкшие).
const activeAddonsFrom = `FROM billing_user_addons a
	LEFT JOIN billing_addons c ON c.code = a.addon_code
	WHERE a.cancelled_at IS NULL AND (a.expires_at IS NULL OR a.expires_at > now())`

func (r *Repo) ListUserAddons(ctx context.Context, userID int64) ([]*domain.UserAddon, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+userAddonColumns+` `+activeAddonsFrom+` AND a.user_id = $1 ORDER BY a.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.UserAddon{}
	for rows.Next() {
		a, err := scanUserAddon(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repo) ListUserAddonsFor(ctx context.Context, userIDs []int64) (map[int64][]*domain.UserAddon, error) {
	out := map[int64][]*domain.UserAddon{}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+userAddonColumns+` `+activeAddonsFrom+` AND a.user_id = ANY($1) ORDER BY a.id`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		a, err := scanUserAddon(rows)
		if err != nil {
			return nil, err
		}
		out[a.UserID] = append(out[a.UserID], a)
	}
	return out, rows.Err()
}

func (r *Repo) AddAddon(ctx context.Context, a *domain.UserAddon) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO billing_user_addons
		    (user_id, addon_code, kind, amount, qty, company_id, period, expires_at, auto_renew)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, started_at`,
		a.UserID, a.AddonCode, a.Kind, a.Amount, a.Qty, a.CompanyID, a.Period,
		a.ExpiresAt, a.AutoRenew).Scan(&a.ID, &a.StartedAt)
}

func (r *Repo) CancelAddon(ctx context.Context, id, userID int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE billing_user_addons SET cancelled_at = now(), auto_renew = false
		 WHERE id = $1 AND user_id = $2 AND cancelled_at IS NULL`, id, userID)
	if err != nil {
		return err
	}
	// Чужая (или уже отменённая) докупка не найдётся. Отвечать «ок» здесь
	// нельзя: человек решит, что списания прекратились, а они продолжатся.
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repo) DueAddonRenewals(ctx context.Context, now time.Time, limit int) ([]*domain.UserAddon, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.user_id, a.addon_code, a.kind, a.amount, a.qty, a.company_id,
		       a.period, a.started_at, a.expires_at, a.auto_renew, a.addon_code
		  FROM billing_user_addons a
		 WHERE a.cancelled_at IS NULL AND a.expires_at IS NOT NULL AND a.expires_at <= $1
		 ORDER BY a.expires_at
		 LIMIT $2`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.UserAddon{}
	for rows.Next() {
		a, err := scanUserAddon(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repo) RenewAddon(ctx context.Context, id int64, until time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE billing_user_addons SET expires_at = $2 WHERE id = $1`, id, until)
	return err
}
