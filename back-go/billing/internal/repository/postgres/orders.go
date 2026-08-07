package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

const orderColumns = `id, user_id, kind, item_code, product_id, period, qty, company_id,
	amount, base_amount, discount, promo_id, status, title, created_at, paid_at, applied_at, meta`

func scanOrder(row scanner) (*domain.Order, error) {
	var o domain.Order
	var meta []byte
	if err := row.Scan(&o.ID, &o.UserID, &o.Kind, &o.ItemCode, &o.ProductID, &o.Period,
		&o.Qty, &o.CompanyID, &o.Amount, &o.BaseAmount, &o.Discount, &o.PromoID,
		&o.Status, &o.Title, &o.CreatedAt, &o.PaidAt, &o.AppliedAt, &meta); err != nil {
		return nil, err
	}
	o.Meta = jsonMap(meta)
	return &o, nil
}

func (r *Repo) CreateOrder(ctx context.Context, o *domain.Order) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO billing_orders
		    (user_id, kind, item_code, product_id, period, qty, company_id,
		     amount, base_amount, discount, promo_id, status, title, meta)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at`,
		o.UserID, o.Kind, o.ItemCode, o.ProductID, o.Period, o.Qty, o.CompanyID,
		o.Amount, o.BaseAmount, o.Discount, o.PromoID, o.Status, o.Title, jsonBytes(o.Meta)).
		Scan(&o.ID, &o.CreatedAt)
}

func (r *Repo) GetOrder(ctx context.Context, id int64) (*domain.Order, error) {
	o, err := scanOrder(r.pool.QueryRow(ctx, `SELECT `+orderColumns+` FROM billing_orders WHERE id = $1`, id))
	if err != nil {
		if noRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return o, nil
}

func (r *Repo) ListOrders(ctx context.Context, userID int64, limit, offset int) ([]*domain.Order, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+orderColumns+`, count(*) OVER () AS total
		  FROM billing_orders WHERE user_id = $1
		 ORDER BY created_at DESC, id DESC
		 LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return collectOrders(rows)
}

func (r *Repo) ListAllOrders(ctx context.Context, status string, limit, offset int) ([]*domain.Order, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+orderColumns+`, count(*) OVER () AS total
		  FROM billing_orders WHERE ($1 = '' OR status = $1)
		 ORDER BY created_at DESC, id DESC
		 LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return collectOrders(rows)
}

// collectOrders — разбор выборки заказов со счётчиком total последним полем.
func collectOrders(rows pgx.Rows) ([]*domain.Order, int, error) {
	defer rows.Close()
	out := []*domain.Order{}
	total := 0
	for rows.Next() {
		var o domain.Order
		var meta []byte
		if err := rows.Scan(&o.ID, &o.UserID, &o.Kind, &o.ItemCode, &o.ProductID, &o.Period,
			&o.Qty, &o.CompanyID, &o.Amount, &o.BaseAmount, &o.Discount, &o.PromoID,
			&o.Status, &o.Title, &o.CreatedAt, &o.PaidAt, &o.AppliedAt, &meta, &total); err != nil {
			return nil, 0, err
		}
		o.Meta = jsonMap(meta)
		out = append(out, &o)
	}
	return out, total, rows.Err()
}

func (r *Repo) SetOrderStatus(ctx context.Context, id int64, status string, paidAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE billing_orders SET status = $2, paid_at = COALESCE($3, paid_at) WHERE id = $1`,
		id, status, paidAt)
	return err
}

// MarkApplied — гейт «применить заказ ровно один раз»: условие applied_at IS
// NULL стоит в WHERE, поэтому гонка двух вебхуков не удвоит подписку.
func (r *Repo) MarkApplied(ctx context.Context, id int64) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE billing_orders SET applied_at = now() WHERE id = $1 AND applied_at IS NULL`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ---------- Платежи ----------

const paymentColumns = `id, order_id, provider, provider_payment_id, amount, status,
	method, confirmation_url, webhook_secret, created_at`

func scanPayment(row scanner) (*domain.Payment, error) {
	var p domain.Payment
	if err := row.Scan(&p.ID, &p.OrderID, &p.Provider, &p.ProviderPaymentID, &p.Amount,
		&p.Status, &p.Method, &p.ConfirmationURL, &p.WebhookSecret, &p.CreatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repo) CreatePayment(ctx context.Context, p *domain.Payment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO billing_payments
		    (order_id, provider, provider_payment_id, amount, status, method, confirmation_url, webhook_secret)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`,
		p.OrderID, p.Provider, p.ProviderPaymentID, p.Amount, p.Status, p.Method,
		p.ConfirmationURL, p.WebhookSecret).Scan(&p.ID, &p.CreatedAt)
}

func (r *Repo) GetPayment(ctx context.Context, id int64) (*domain.Payment, error) {
	p, err := scanPayment(r.pool.QueryRow(ctx, `SELECT `+paymentColumns+` FROM billing_payments WHERE id = $1`, id))
	if err != nil {
		if noRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *Repo) GetPaymentByOrder(ctx context.Context, orderID int64) (*domain.Payment, error) {
	p, err := scanPayment(r.pool.QueryRow(ctx,
		`SELECT `+paymentColumns+` FROM billing_payments WHERE order_id = $1 ORDER BY id DESC LIMIT 1`, orderID))
	if err != nil {
		if noRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *Repo) SetPaymentStatus(ctx context.Context, id int64, status string, raw map[string]any) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE billing_payments SET status = $2, raw = $3, updated_at = now() WHERE id = $1`,
		id, status, jsonBytes(raw))
	return err
}
