package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// InstallmentRepo — рассрочки (pet_installments): оплата покупок долями поверх
// pets (кошелёк) и pet_kudos_ledger (выписка платежей).
type InstallmentRepo struct {
	pool *pgxpool.Pool
}

var _ domain.InstallmentRepo = (*InstallmentRepo)(nil)

func NewInstallmentRepo(pool *pgxpool.Pool) *InstallmentRepo {
	return &InstallmentRepo{pool: pool}
}

const installmentCols = `id, user_id, company_id, category, item_key, item_title,
	total, paid, parts, due_at, penalized, created_at`

func scanInstallment(row pgx.Row) (*domain.Installment, error) {
	var i domain.Installment
	err := row.Scan(&i.ID, &i.UserID, &i.CompanyID, &i.Category, &i.ItemKey, &i.ItemTitle,
		&i.Total, &i.Paid, &i.Parts, &i.DueAt, &i.Penalized, &i.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &i, nil
}

func (r *InstallmentRepo) Create(ctx context.Context, i *domain.Installment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO pet_installments (user_id, company_id, category, item_key, item_title,
			total, paid, parts, due_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`,
		i.UserID, i.CompanyID, i.Category, i.ItemKey, i.ItemTitle,
		i.Total, i.Paid, i.Parts, i.DueAt).Scan(&i.ID, &i.CreatedAt)
}

func (r *InstallmentRepo) ListActive(ctx context.Context, userID int64) ([]*domain.Installment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+installmentCols+` FROM pet_installments
		 WHERE user_id = $1 AND paid < total ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Installment{}
	for rows.Next() {
		i, err := scanInstallment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *InstallmentRepo) Get(ctx context.Context, id, userID int64) (*domain.Installment, error) {
	return scanInstallment(r.pool.QueryRow(ctx,
		`SELECT `+installmentCols+` FROM pet_installments WHERE id = $1 AND user_id = $2`, id, userID))
}

func (r *InstallmentRepo) Outstanding(ctx context.Context, userID int64) (int, error) {
	var sum int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(total - paid), 0) FROM pet_installments
		 WHERE user_id = $1 AND paid < total`, userID).Scan(&sum)
	return sum, err
}

// Pay — платёж по рассрочке: списание с кошелька, +paid и запись леджера в одной
// транзакции. Guard'ы (kudos >= amount, рассрочка активна, amount ≤ остатка) —
// в WHERE, no rows → ok=false.
func (r *InstallmentRepo) Pay(ctx context.Context, id, userID int64, amount int) (int, int, int, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, 0, 0, false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // после Commit — no-op

	var companyID int64
	if err := tx.QueryRow(ctx, `SELECT company_id FROM pets WHERE user_id = $1`, userID).Scan(&companyID); err != nil {
		return 0, 0, 0, false, err
	}

	var paid, total, kudos int
	err = tx.QueryRow(ctx, `
		WITH upd AS (
			UPDATE pet_installments SET paid = paid + $3
			WHERE id = $1 AND user_id = $2 AND paid < total AND $3 <= total - paid
			  AND (SELECT kudos FROM pets WHERE user_id = $2) >= $3
			RETURNING paid, total
		), wallet AS (
			UPDATE pets SET kudos = kudos - $3
			WHERE user_id = $2 AND EXISTS (SELECT 1 FROM upd)
			RETURNING kudos
		)
		SELECT upd.paid, upd.total, wallet.kudos FROM upd, wallet`,
		id, userID, amount).Scan(&paid, &total, &kudos)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, 0, false, nil
		}
		return 0, 0, 0, false, err
	}
	if _, err := tx.Exec(ctx, insertLedger, userID, companyID, -amount, "installment_pay", nil, ""); err != nil {
		return 0, 0, 0, false, err
	}
	return paid, total, kudos, true, tx.Commit(ctx)
}

// AddCharge — ленивое еженедельное начисление на остаток рассрочки. Оптимистичный
// guard due_at = oldDue делает начисление идемпотентным при гонке.
func (r *InstallmentRepo) AddCharge(ctx context.Context, id int64, charge int, oldDue, newDue time.Time) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE pet_installments SET total = total + $2, due_at = $4, penalized = TRUE
		WHERE id = $1 AND paid < total AND due_at = $3`,
		id, charge, oldDue, newDue)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *InstallmentRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM pet_installments WHERE id = $1`, id)
	return err
}

func (r *InstallmentRepo) DeleteForUser(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM pet_installments WHERE user_id = $1`, userID)
	return err
}
