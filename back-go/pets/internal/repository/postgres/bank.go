package postgres

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// BankRepo — кудо-банк поверх pets (балансы) и pet_kudos_ledger (выписка).
// Все мутации — атомарные UPDATE с guard'ами в WHERE; переводы и банковские
// операции пишут леджер в той же транзакции, «журнальные» записи прочих
// механик добавляет сервис через AppendLedger (fire-and-forget).
type BankRepo struct {
	pool *pgxpool.Pool
}

var _ domain.BankRepo = (*BankRepo)(nil)

func NewBankRepo(pool *pgxpool.Pool) *BankRepo {
	return &BankRepo{pool: pool}
}

const insertLedger = `
	INSERT INTO pet_kudos_ledger (user_id, company_id, delta, kind, counterparty_id, comment)
	VALUES ($1, $2, $3, $4, $5, $6)`

func (r *BankRepo) AppendLedger(ctx context.Context, e *domain.LedgerEntry) error {
	_, err := r.pool.Exec(ctx, insertLedger,
		e.UserID, e.CompanyID, e.Delta, e.Kind, e.CounterpartyID, e.Comment)
	return err
}

func (r *BankRepo) ListLedger(ctx context.Context, userID, beforeID int64, limit int) ([]*domain.LedgerEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT l.id, l.user_id, l.company_id, l.delta, l.kind, l.counterparty_id,
		       l.comment, l.created_at, u.id, u.fio, u.avatar_path
		FROM pet_kudos_ledger l
		LEFT JOIN users u ON u.id = l.counterparty_id
		WHERE l.user_id = $1 AND ($2 = 0 OR l.id < $2)
		ORDER BY l.id DESC
		LIMIT $3`, userID, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.LedgerEntry
	for rows.Next() {
		var e domain.LedgerEntry
		var uid *int64
		var fio, avatar *string
		if err := rows.Scan(&e.ID, &e.UserID, &e.CompanyID, &e.Delta, &e.Kind,
			&e.CounterpartyID, &e.Comment, &e.CreatedAt, &uid, &fio, &avatar); err != nil {
			return nil, err
		}
		e.Counterparty = userRef(uid, fio, avatar)
		out = append(out, &e)
	}
	return out, rows.Err()
}

// Transfer — списание у отправителя (guard по балансу), зачисление получателю
// и обе записи выписки одной транзакцией: полперевода не бывает.
func (r *BankRepo) Transfer(ctx context.Context, fromID, toID, companyID int64,
	amount int, comment string) (int, bool, error) {

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // после Commit — no-op

	var fromKudos int
	err = tx.QueryRow(ctx, `
		UPDATE pets SET kudos = kudos - $2
		WHERE user_id = $1 AND kudos >= $2
		RETURNING kudos`, fromID, amount).Scan(&fromKudos)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil // не хватает кудосов
		}
		return 0, false, err
	}
	if _, err := tx.Exec(ctx, `UPDATE pets SET kudos = kudos + $2 WHERE user_id = $1`,
		toID, amount); err != nil {
		return 0, false, err
	}
	if _, err := tx.Exec(ctx, insertLedger, fromID, companyID, -amount, "transfer_out", toID, comment); err != nil {
		return 0, false, err
	}
	if _, err := tx.Exec(ctx, insertLedger, toID, companyID, amount, "transfer_in", fromID, comment); err != nil {
		return 0, false, err
	}
	return fromKudos, true, tx.Commit(ctx)
}

func (r *BankRepo) MonthlyTotals(ctx context.Context, userID int64) (int, int, error) {
	var in, out int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(delta) FILTER (WHERE delta > 0), 0),
		       COALESCE(-sum(delta) FILTER (WHERE delta < 0), 0)
		FROM pet_kudos_ledger
		WHERE user_id = $1 AND created_at >= now() - interval '30 days'`,
		userID).Scan(&in, &out)
	return in, out, err
}

func (r *BankRepo) LifetimeEarned(ctx context.Context, userID int64) (int, error) {
	var earned int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(delta), 0)
		FROM pet_kudos_ledger
		WHERE user_id = $1 AND delta > 0 AND kind != ALL($2)`,
		userID, domain.LedgerEarnExcluded).Scan(&earned)
	return earned, err
}

func (r *BankRepo) TopGenerous(ctx context.Context, companyID int64, limit int) ([]domain.GenerousEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT l.user_id, u.fio, u.avatar_path, -sum(l.delta) AS sent
		FROM pet_kudos_ledger l
		JOIN users u ON u.id = l.user_id
		WHERE l.company_id = $1 AND l.kind = 'transfer_out'
		  AND l.created_at >= now() - interval '30 days'
		GROUP BY l.user_id, u.fio, u.avatar_path
		ORDER BY sent DESC
		LIMIT $2`, companyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.GenerousEntry
	for rows.Next() {
		var uid int64
		var fio string
		var avatar *string
		var sent int
		if err := rows.Scan(&uid, &fio, &avatar, &sent); err != nil {
			return nil, err
		}
		out = append(out, domain.GenerousEntry{
			User: &domain.UserRef{ID: uid, FIO: fio, AvatarPath: avatar},
			Sent: sent,
		})
	}
	return out, rows.Err()
}

// bankOp — общий каркас «атомарный UPDATE балансов + запись леджера в одной
// транзакции». scanDest — куда читать RETURNING; no rows → ok=false.
func (r *BankRepo) bankOp(ctx context.Context, sql string, args []any,
	scanDest []any, ledger *domain.LedgerEntry) (bool, error) {

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // после Commit — no-op

	if err := tx.QueryRow(ctx, sql, args...).Scan(scanDest...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if _, err := tx.Exec(ctx, insertLedger, ledger.UserID, ledger.CompanyID,
		ledger.Delta, ledger.Kind, ledger.CounterpartyID, ledger.Comment); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// companyOf — company_id питомца для записи леджера банковских операций.
func (r *BankRepo) companyOf(ctx context.Context, userID int64) (int64, error) {
	var companyID int64
	err := r.pool.QueryRow(ctx, `SELECT company_id FROM pets WHERE user_id = $1`, userID).Scan(&companyID)
	return companyID, err
}

func (r *BankRepo) DepositSavings(ctx context.Context, userID int64, amount int) (int, int, bool, error) {
	companyID, err := r.companyOf(ctx, userID)
	if err != nil {
		return 0, 0, false, err
	}
	var kudos, savings int
	ok, err := r.bankOp(ctx, `
		UPDATE pets SET kudos = kudos - $2, bank_savings = bank_savings + $2,
			bank_savings_accrued_at = CASE WHEN bank_savings = 0 THEN now()
				ELSE bank_savings_accrued_at END
		WHERE user_id = $1 AND kudos >= $2 AND bank_loan = 0
		RETURNING kudos, bank_savings`,
		[]any{userID, amount}, []any{&kudos, &savings},
		&domain.LedgerEntry{UserID: userID, CompanyID: companyID, Delta: -amount, Kind: "bank_deposit"})
	return kudos, savings, ok, err
}

func (r *BankRepo) WithdrawSavings(ctx context.Context, userID int64, amount int) (int, int, bool, error) {
	companyID, err := r.companyOf(ctx, userID)
	if err != nil {
		return 0, 0, false, err
	}
	var kudos, savings int
	ok, err := r.bankOp(ctx, `
		UPDATE pets SET kudos = kudos + $2, bank_savings = bank_savings - $2,
			bank_savings_accrued_at = CASE WHEN bank_savings - $2 <= 0 THEN NULL
				ELSE bank_savings_accrued_at END
		WHERE user_id = $1 AND bank_savings >= $2
		RETURNING kudos, bank_savings`,
		[]any{userID, amount}, []any{&kudos, &savings},
		&domain.LedgerEntry{UserID: userID, CompanyID: companyID, Delta: amount, Kind: "bank_withdraw"})
	return kudos, savings, ok, err
}

// AccrueSavings — простой (некомпаундный) процент за целые прошедшие сутки:
// perday = min(savings·rate%, dailyMax), одним UPDATE через self-join (как
// FinishAdventure) — конкурентный второй вызов увидит сдвинутую отметку и
// начислит 0. Запись выписки — в той же транзакции.
func (r *BankRepo) AccrueSavings(ctx context.Context, userID, companyID int64, ratePct, dailyMax int) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // после Commit — no-op

	var interest int
	err = tx.QueryRow(ctx, `
		UPDATE pets p SET
			bank_savings = old.bank_savings + f.perday * f.days,
			bank_savings_accrued_at = old.bank_savings_accrued_at + make_interval(days => f.days)
		FROM pets old
		CROSS JOIN LATERAL (
			SELECT LEAST(old.bank_savings * $2 / 100, $3) AS perday,
			       floor(extract(epoch FROM (now() - old.bank_savings_accrued_at)) / 86400)::int AS days
		) f
		WHERE p.user_id = $1 AND old.user_id = p.user_id
		  AND old.bank_savings > 0 AND old.bank_savings_accrued_at IS NOT NULL
		  AND f.days >= 1
		RETURNING p.bank_savings - old.bank_savings`,
		userID, ratePct, dailyMax).Scan(&interest)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil // суток не прошло либо вклад пуст
		}
		return 0, err
	}
	if interest > 0 {
		if _, err := tx.Exec(ctx, insertLedger, userID, companyID, interest, "bank_interest", nil, ""); err != nil {
			return 0, err
		}
	}
	return interest, tx.Commit(ctx)
}

func (r *BankRepo) TakeLoan(ctx context.Context, userID int64, amount, debt int) (int, bool, error) {
	companyID, err := r.companyOf(ctx, userID)
	if err != nil {
		return 0, false, err
	}
	var kudos int
	ok, err := r.bankOp(ctx, `
		UPDATE pets SET kudos = kudos + $2, bank_loan = $3
		WHERE user_id = $1 AND bank_loan = 0
		RETURNING kudos`,
		[]any{userID, amount, debt}, []any{&kudos},
		&domain.LedgerEntry{UserID: userID, CompanyID: companyID, Delta: amount, Kind: "loan_taken",
			Comment: "долг " + strconv.Itoa(debt)})
	return kudos, ok, err
}

func (r *BankRepo) RepayLoan(ctx context.Context, userID int64, amount int) (int, int, bool, error) {
	companyID, err := r.companyOf(ctx, userID)
	if err != nil {
		return 0, 0, false, err
	}
	var kudos, loan int
	ok, err := r.bankOp(ctx, `
		UPDATE pets SET kudos = kudos - $2, bank_loan = bank_loan - $2
		WHERE user_id = $1 AND kudos >= $2 AND bank_loan >= $2
		RETURNING kudos, bank_loan`,
		[]any{userID, amount}, []any{&kudos, &loan},
		&domain.LedgerEntry{UserID: userID, CompanyID: companyID, Delta: -amount, Kind: "loan_repaid"})
	return kudos, loan, ok, err
}

func (r *BankRepo) DeleteLedger(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM pet_kudos_ledger WHERE user_id = $1`, userID)
	return err
}
