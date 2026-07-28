package postgres

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

const productColumns = `p.id, p.kind, p.title, p.description, p.price, p.author_id,
	COALESCE(u.fio, ''), p.status, p.reject_reason, p.cover_path, p.payload,
	p.sales_count, p.sort, p.created_at, p.updated_at, p.published_at`

func scanProduct(row scanner) (*domain.Product, error) {
	var p domain.Product
	var payload []byte
	if err := row.Scan(&p.ID, &p.Kind, &p.Title, &p.Description, &p.Price, &p.AuthorID,
		&p.AuthorName, &p.Status, &p.RejectReason, &p.CoverPath, &payload,
		&p.SalesCount, &p.Sort, &p.CreatedAt, &p.UpdatedAt, &p.PublishedAt); err != nil {
		return nil, err
	}
	p.Payload = jsonMap(payload)
	return &p, nil
}

func (r *Repo) ListShowcase(ctx context.Context, kind, search string, viewerID int64, limit, offset int) ([]*domain.Product, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+productColumns+`,
		       EXISTS (SELECT 1 FROM billing_product_purchases pp
		                WHERE pp.product_id = p.id AND pp.user_id = $3) AS owned,
		       count(*) OVER () AS total
		  FROM billing_products p
		  LEFT JOIN users u ON u.id = p.author_id
		 WHERE p.status = 'published'
		   AND ($1 = '' OR p.kind = $1)
		   AND ($2 = '' OR p.title ILIKE '%' || $2 || '%' OR p.description ILIKE '%' || $2 || '%')
		 ORDER BY p.sort, p.published_at DESC NULLS LAST, p.id DESC
		 LIMIT $4 OFFSET $5`, kind, search, viewerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*domain.Product{}
	total := 0
	for rows.Next() {
		var p domain.Product
		var payload []byte
		if err := rows.Scan(&p.ID, &p.Kind, &p.Title, &p.Description, &p.Price, &p.AuthorID,
			&p.AuthorName, &p.Status, &p.RejectReason, &p.CoverPath, &payload,
			&p.SalesCount, &p.Sort, &p.CreatedAt, &p.UpdatedAt, &p.PublishedAt,
			&p.Owned, &total); err != nil {
			return nil, 0, err
		}
		p.Payload = jsonMap(payload)
		out = append(out, &p)
	}
	return out, total, rows.Err()
}

func (r *Repo) GetProduct(ctx context.Context, id int64) (*domain.Product, error) {
	p, err := scanProduct(r.pool.QueryRow(ctx, `
		SELECT `+productColumns+`
		  FROM billing_products p
		  LEFT JOIN users u ON u.id = p.author_id
		 WHERE p.id = $1`, id))
	if err != nil {
		if noRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *Repo) CreateProduct(ctx context.Context, p *domain.Product) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO billing_products (kind, title, description, price, author_id, status, cover_path, payload, sort)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`,
		p.Kind, p.Title, p.Description, p.Price, p.AuthorID, p.Status, p.CoverPath,
		jsonBytes(p.Payload), p.Sort).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *Repo) UpdateProduct(ctx context.Context, p *domain.Product) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE billing_products
		   SET kind = $2, title = $3, description = $4, price = $5, cover_path = $6,
		       payload = $7, sort = $8, updated_at = now()
		 WHERE id = $1`,
		p.ID, p.Kind, p.Title, p.Description, p.Price, p.CoverPath, jsonBytes(p.Payload), p.Sort)
	return err
}

func (r *Repo) SetProductStatus(ctx context.Context, id int64, status, reason string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE billing_products
		   SET status = $2, reject_reason = $3, updated_at = now(),
		       published_at = CASE WHEN $2 = 'published' THEN COALESCE(published_at, now()) ELSE published_at END
		 WHERE id = $1`, id, status, reason)
	return err
}

func (r *Repo) DeleteProduct(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM billing_products WHERE id = $1`, id)
	return err
}

func (r *Repo) ListByAuthor(ctx context.Context, authorID int64) ([]*domain.Product, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+productColumns+`
		  FROM billing_products p
		  LEFT JOIN users u ON u.id = p.author_id
		 WHERE p.author_id = $1
		 ORDER BY p.updated_at DESC, p.id DESC`, authorID)
	if err != nil {
		return nil, err
	}
	return collectProducts(rows)
}

func (r *Repo) ListModeration(ctx context.Context) ([]*domain.Product, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+productColumns+`
		  FROM billing_products p
		  LEFT JOIN users u ON u.id = p.author_id
		 WHERE p.status IN ('review', 'published', 'rejected')
		 ORDER BY CASE p.status WHEN 'review' THEN 0 ELSE 1 END, p.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	return collectProducts(rows)
}

type productRows interface {
	Next() bool
	Scan(...any) error
	Close()
	Err() error
}

func collectProducts(rows productRows) ([]*domain.Product, error) {
	defer rows.Close()
	out := []*domain.Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// PurchaseProduct — покупка одной транзакцией: запись покупки, счётчик продаж и
// зачисление выручки автору (за вычетом комиссии платформы).
func (r *Repo) PurchaseProduct(ctx context.Context, productID, userID int64, orderID *int64,
	amount, authorShare int64, authorID *int64) error {

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO billing_product_purchases (product_id, user_id, order_id, amount, author_share)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (product_id, user_id) DO NOTHING`,
		productID, userID, orderID, amount, authorShare); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE billing_products SET sales_count = sales_count + 1 WHERE id = $1`, productID); err != nil {
		return err
	}
	if authorID != nil && authorShare > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO billing_seller_balances (user_id, balance, total_earned, updated_at)
			VALUES ($1, $2, $2, now())
			ON CONFLICT (user_id) DO UPDATE SET balance = billing_seller_balances.balance + $2,
			    total_earned = billing_seller_balances.total_earned + $2, updated_at = now()`,
			*authorID, authorShare); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repo) ListPurchases(ctx context.Context, userID int64) ([]*domain.ProductPurchase, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pp.id, pp.product_id, pp.user_id, pp.amount, pp.created_at,
		       `+productColumns+`
		  FROM billing_product_purchases pp
		  JOIN billing_products p ON p.id = pp.product_id
		  LEFT JOIN users u ON u.id = p.author_id
		 WHERE pp.user_id = $1
		 ORDER BY pp.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.ProductPurchase{}
	for rows.Next() {
		var pu domain.ProductPurchase
		var p domain.Product
		var payload []byte
		if err := rows.Scan(&pu.ID, &pu.ProductID, &pu.UserID, &pu.Amount, &pu.CreatedAt,
			&p.ID, &p.Kind, &p.Title, &p.Description, &p.Price, &p.AuthorID, &p.AuthorName,
			&p.Status, &p.RejectReason, &p.CoverPath, &payload, &p.SalesCount, &p.Sort,
			&p.CreatedAt, &p.UpdatedAt, &p.PublishedAt); err != nil {
			return nil, err
		}
		p.Payload = jsonMap(payload)
		p.Owned = true
		pu.Product = &p
		out = append(out, &pu)
	}
	return out, rows.Err()
}

func (r *Repo) IsOwned(ctx context.Context, productID, userID int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM billing_product_purchases WHERE product_id = $1 AND user_id = $2)`,
		productID, userID).Scan(&ok)
	return ok, err
}

// ---------- Кошелёк автора ----------

func (r *Repo) GetSellerBalance(ctx context.Context, userID int64) (*domain.SellerBalance, error) {
	b := &domain.SellerBalance{UserID: userID}
	err := r.pool.QueryRow(ctx,
		`SELECT balance, total_earned FROM billing_seller_balances WHERE user_id = $1`, userID).
		Scan(&b.Balance, &b.TotalEarned)
	if err != nil {
		if noRows(err) {
			return b, nil
		}
		return nil, err
	}
	return b, nil
}

// CreatePayout — заявка на вывод: сумма резервируется списанием с кошелька в
// той же транзакции (гейт баланса — в WHERE).
func (r *Repo) CreatePayout(ctx context.Context, p *domain.Payout) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE billing_seller_balances SET balance = balance - $2, updated_at = now()
		 WHERE user_id = $1 AND balance >= $2`, p.UserID, p.Amount)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotEnoughFunds
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO billing_payouts (user_id, amount, requisites) VALUES ($1, $2, $3)
		RETURNING id, status, created_at`,
		p.UserID, p.Amount, p.Requisites).Scan(&p.ID, &p.Status, &p.CreatedAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repo) ListPayouts(ctx context.Context, userID int64, all bool) ([]*domain.Payout, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.user_id, COALESCE(u.fio, ''), p.amount, p.status, p.requisites,
		       p.note, p.created_at, p.processed_at
		  FROM billing_payouts p
		  LEFT JOIN users u ON u.id = p.user_id
		 WHERE $2 OR p.user_id = $1
		 ORDER BY p.created_at DESC`, userID, all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Payout{}
	for rows.Next() {
		var p domain.Payout
		if err := rows.Scan(&p.ID, &p.UserID, &p.UserName, &p.Amount, &p.Status,
			&p.Requisites, &p.Note, &p.CreatedAt, &p.ProcessedAt); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// ProcessPayout — решение супер-админа. Отказ возвращает сумму на кошелёк.
func (r *Repo) ProcessPayout(ctx context.Context, id int64, status, note string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID, amount int64
	err = tx.QueryRow(ctx, `
		UPDATE billing_payouts SET status = $2, note = $3, processed_at = now()
		 WHERE id = $1 AND status = 'requested'
		RETURNING user_id, amount`, id, status, note).Scan(&userID, &amount)
	if err != nil {
		if noRows(err) {
			return domain.ErrNotFound
		}
		return err
	}
	if status == "rejected" {
		if _, err := tx.Exec(ctx, `
			UPDATE billing_seller_balances SET balance = balance + $2, updated_at = now()
			 WHERE user_id = $1`, userID, amount); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
