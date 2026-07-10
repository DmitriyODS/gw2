package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// Копилки-цели, благотворительные сборы и статистика кудо-банка. Мутации —
// по образцу bank.go: атомарные UPDATE с guard'ами в WHERE + записи леджера
// в той же транзакции.

const goalColumns = `id, user_id, company_id, title, emoji, target, saved, created_at, achieved_at`

func scanGoal(row pgx.Row) (*domain.BankGoal, error) {
	var g domain.BankGoal
	err := row.Scan(&g.ID, &g.UserID, &g.CompanyID, &g.Title, &g.Emoji,
		&g.Target, &g.Saved, &g.CreatedAt, &g.AchievedAt)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *BankRepo) ListGoals(ctx context.Context, userID int64) ([]*domain.BankGoal, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+goalColumns+` FROM pet_bank_goals WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.BankGoal
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *BankRepo) CreateGoal(ctx context.Context, g *domain.BankGoal) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO pet_bank_goals (user_id, company_id, title, emoji, target)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		g.UserID, g.CompanyID, g.Title, g.Emoji, g.Target).Scan(&g.ID, &g.CreatedAt)
}

// GoalDeposit — кошелёк → копилка одной транзакцией; achieved_at ставится
// однажды (RETURNING старого значения различает «достигнута именно сейчас»).
func (r *BankRepo) GoalDeposit(ctx context.Context, userID, goalID int64, amount int) (*domain.BankGoal, bool, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, false, false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // после Commit — no-op

	var companyID int64
	err = tx.QueryRow(ctx, `
		UPDATE pets SET kudos = kudos - $2
		WHERE user_id = $1 AND kudos >= $2
		RETURNING company_id`, userID, amount).Scan(&companyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, false, nil // не хватает кудосов
		}
		return nil, false, false, err
	}
	var wasAchieved bool
	g := &domain.BankGoal{}
	err = tx.QueryRow(ctx, `
		UPDATE pet_bank_goals SET saved = saved + $3,
			achieved_at = CASE WHEN achieved_at IS NULL AND saved + $3 >= target
				THEN now() ELSE achieved_at END
		WHERE id = $2 AND user_id = $1
		RETURNING `+goalColumns+`, (SELECT achieved_at IS NOT NULL FROM pet_bank_goals WHERE id = $2) AS was`,
		userID, goalID, amount).Scan(&g.ID, &g.UserID, &g.CompanyID, &g.Title, &g.Emoji,
		&g.Target, &g.Saved, &g.CreatedAt, &g.AchievedAt, &wasAchieved)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, false, nil // копилка не найдена / чужая
		}
		return nil, false, false, err
	}
	if _, err := tx.Exec(ctx, insertLedger, userID, companyID, -amount, "goal_deposit", nil,
		g.Emoji+" "+g.Title); err != nil {
		return nil, false, false, err
	}
	achievedNow := g.AchievedAt != nil && !wasAchieved
	return g, achievedNow, true, tx.Commit(ctx)
}

func (r *BankRepo) GoalWithdraw(ctx context.Context, userID, goalID int64, amount int) (*domain.BankGoal, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // после Commit — no-op

	g := &domain.BankGoal{}
	err = tx.QueryRow(ctx, `
		UPDATE pet_bank_goals SET saved = saved - $3
		WHERE id = $2 AND user_id = $1 AND saved >= $3
		RETURNING `+goalColumns, userID, goalID, amount).
		Scan(&g.ID, &g.UserID, &g.CompanyID, &g.Title, &g.Emoji,
			&g.Target, &g.Saved, &g.CreatedAt, &g.AchievedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if _, err := tx.Exec(ctx, `UPDATE pets SET kudos = kudos + $2 WHERE user_id = $1`,
		userID, amount); err != nil {
		return nil, false, err
	}
	if _, err := tx.Exec(ctx, insertLedger, userID, g.CompanyID, amount, "goal_withdraw", nil,
		g.Emoji+" "+g.Title); err != nil {
		return nil, false, err
	}
	return g, true, tx.Commit(ctx)
}

// DeleteGoal — удаление копилки; остаток возвращается в кошелёк той же
// транзакцией (запись леджера — только если остаток был).
func (r *BankRepo) DeleteGoal(ctx context.Context, userID, goalID int64) (int, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // после Commit — no-op

	var refund int
	var companyID int64
	var title, emoji string
	err = tx.QueryRow(ctx, `
		DELETE FROM pet_bank_goals WHERE id = $2 AND user_id = $1
		RETURNING saved, company_id, title, emoji`, userID, goalID).
		Scan(&refund, &companyID, &title, &emoji)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if refund > 0 {
		if _, err := tx.Exec(ctx, `UPDATE pets SET kudos = kudos + $2 WHERE user_id = $1`,
			userID, refund); err != nil {
			return 0, false, err
		}
		if _, err := tx.Exec(ctx, insertLedger, userID, companyID, refund, "goal_withdraw", nil,
			emoji+" "+title); err != nil {
			return 0, false, err
		}
	}
	return refund, true, tx.Commit(ctx)
}

const fundColumns = `f.id, f.company_id, f.created_by, f.title, f.description, f.emoji,
	f.target, f.collected, f.status, f.created_at, f.finished_at`

func scanFund(row pgx.Row, f *domain.BankFund) error {
	return row.Scan(&f.ID, &f.CompanyID, &f.CreatedBy, &f.Title, &f.Description, &f.Emoji,
		&f.Target, &f.Collected, &f.Status, &f.CreatedAt, &f.FinishedAt)
}

func (r *BankRepo) ListFunds(ctx context.Context, companyID, viewerID int64, finishedShown int) ([]*domain.BankFund, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+fundColumns+`, u.id, u.fio, u.avatar_path,
		       COALESCE(d.donors, 0), COALESCE(d.mine, 0)
		FROM pet_bank_funds f
		LEFT JOIN users u ON u.id = f.created_by
		LEFT JOIN LATERAL (
			SELECT count(DISTINCT user_id) AS donors,
			       COALESCE(sum(amount) FILTER (WHERE user_id = $2), 0) AS mine
			FROM pet_bank_fund_donations WHERE fund_id = f.id
		) d ON true
		WHERE f.company_id = $1 AND (f.status = 'active' OR f.id IN (
			SELECT id FROM pet_bank_funds
			WHERE company_id = $1 AND status != 'active'
			ORDER BY finished_at DESC NULLS LAST LIMIT $3
		))
		ORDER BY (f.status = 'active') DESC, f.id DESC`,
		companyID, viewerID, finishedShown)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.BankFund
	for rows.Next() {
		var f domain.BankFund
		var uid *int64
		var fio, avatar *string
		if err := rows.Scan(&f.ID, &f.CompanyID, &f.CreatedBy, &f.Title, &f.Description, &f.Emoji,
			&f.Target, &f.Collected, &f.Status, &f.CreatedAt, &f.FinishedAt,
			&uid, &fio, &avatar, &f.DonorsCount, &f.MyDonated); err != nil {
			return nil, err
		}
		f.Creator = userRef(uid, fio, avatar)
		out = append(out, &f)
	}
	return out, rows.Err()
}

func (r *BankRepo) CreateFund(ctx context.Context, f *domain.BankFund) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO pet_bank_funds (company_id, created_by, title, description, emoji, target)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		f.CompanyID, f.CreatedBy, f.Title, f.Description, f.Emoji, f.Target).
		Scan(&f.ID, &f.CreatedAt)
}

// Donate — пожертвование одной транзакцией: сбор (guard active) → кошелёк
// (guard баланса) → запись донации → леджер. Достижение цели переводит сбор
// в done прямо в том же UPDATE (completedNow — ровно один раз).
func (r *BankRepo) Donate(ctx context.Context, userID, fundID, companyID int64, amount int) (*domain.BankFund, bool, bool, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, false, false, false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // после Commit — no-op

	f := &domain.BankFund{}
	err = scanFund(tx.QueryRow(ctx, `
		UPDATE pet_bank_funds f SET collected = f.collected + $3,
			status = CASE WHEN f.collected + $3 >= f.target THEN 'done' ELSE f.status END,
			finished_at = CASE WHEN f.collected + $3 >= f.target THEN now() ELSE f.finished_at END
		WHERE f.id = $1 AND f.company_id = $2 AND f.status = 'active'
		RETURNING `+fundColumns), f)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, false, false, nil // сбор не найден / не активен
		}
		return nil, false, false, false, err
	}
	var kudos int
	err = tx.QueryRow(ctx, `
		UPDATE pets SET kudos = kudos - $2
		WHERE user_id = $1 AND kudos >= $2
		RETURNING kudos`, userID, amount).Scan(&kudos)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, true, false, false, nil // не хватает кудосов
		}
		return nil, false, false, false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO pet_bank_fund_donations (fund_id, user_id, amount)
		VALUES ($1, $2, $3)`, fundID, userID, amount); err != nil {
		return nil, false, false, false, err
	}
	if _, err := tx.Exec(ctx, insertLedger, userID, companyID, -amount, "charity", nil,
		f.Emoji+" "+f.Title); err != nil {
		return nil, false, false, false, err
	}
	return f, true, true, f.Status == "done", tx.Commit(ctx)
}

func (r *BankRepo) CloseFund(ctx context.Context, fundID, companyID int64) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE pet_bank_funds SET status = 'closed', finished_at = now()
		WHERE id = $1 AND company_id = $2 AND status = 'active'`, fundID, companyID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *BankRepo) FundTopDonors(ctx context.Context, fundID int64, limit int) ([]domain.GenerousEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.user_id, u.fio, u.avatar_path, sum(d.amount) AS donated
		FROM pet_bank_fund_donations d
		JOIN users u ON u.id = d.user_id
		WHERE d.fund_id = $1
		GROUP BY d.user_id, u.fio, u.avatar_path
		ORDER BY donated DESC
		LIMIT $2`, fundID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.GenerousEntry
	for rows.Next() {
		var uid int64
		var fio string
		var avatar *string
		var donated int
		if err := rows.Scan(&uid, &fio, &avatar, &donated); err != nil {
			return nil, err
		}
		out = append(out, domain.GenerousEntry{
			User: &domain.UserRef{ID: uid, FIO: fio, AvatarPath: avatar},
			Sent: donated,
		})
	}
	return out, rows.Err()
}

// DailyTotals — приход/расход по календарным дням МСК за последние days дней
// (дни без операций не возвращаются — нули дорисовывает клиент).
func (r *BankRepo) DailyTotals(ctx context.Context, userID int64, days int) ([]domain.BankDayStat, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT date_trunc('day', created_at AT TIME ZONE 'Europe/Moscow') AS day,
		       COALESCE(sum(delta) FILTER (WHERE delta > 0), 0),
		       COALESCE(-sum(delta) FILTER (WHERE delta < 0), 0)
		FROM pet_kudos_ledger
		WHERE user_id = $1 AND created_at >= now() - make_interval(days => $2)
		GROUP BY day
		ORDER BY day`, userID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.BankDayStat
	for rows.Next() {
		var s domain.BankDayStat
		if err := rows.Scan(&s.Day, &s.In, &s.Out); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *BankRepo) KindTotals(ctx context.Context, userID int64, days int) ([]domain.BankKindStat, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT kind,
		       COALESCE(sum(delta) FILTER (WHERE delta > 0), 0),
		       COALESCE(-sum(delta) FILTER (WHERE delta < 0), 0)
		FROM pet_kudos_ledger
		WHERE user_id = $1 AND created_at >= now() - make_interval(days => $2)
		GROUP BY kind`, userID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.BankKindStat
	for rows.Next() {
		var s domain.BankKindStat
		if err := rows.Scan(&s.Kind, &s.In, &s.Out); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
