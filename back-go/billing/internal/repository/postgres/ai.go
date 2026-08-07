package postgres

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

// GetBalance — баланс как есть. Строки нет — nil (её заводит первое списание).
func (r *Repo) GetBalance(ctx context.Context, userID int64) (*domain.AIBalance, error) {
	var b domain.AIBalance
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, plan_tokens, used_tokens, extra_tokens, period_start, period_end
		  FROM billing_ai_balances WHERE user_id = $1`, userID).
		Scan(&b.UserID, &b.PlanTokens, &b.UsedTokens, &b.ExtraTokens, &b.PeriodStart, &b.PeriodEnd)
	if err != nil {
		if noRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

// EnsureBalance — баланс с ленивым ролловером периода: если период кончился,
// расход обнуляется, квота выставляется заново, а докупленные токены остаются.
// Отдельного шедулера для этого не нужно — баланс всё равно читается перед
// каждым обращением к модели. Границу периода считает домен (AIQuota).
func (r *Repo) EnsureBalance(ctx context.Context, userID int64, planTokens int64, now, periodEnd time.Time) (*domain.AIBalance, error) {
	var b domain.AIBalance
	err := r.pool.QueryRow(ctx, `
		INSERT INTO billing_ai_balances (user_id, plan_tokens, period_start, period_end)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		   SET plan_tokens = CASE WHEN billing_ai_balances.period_end <= $3 THEN EXCLUDED.plan_tokens
		                          ELSE $2 END,
		       used_tokens = CASE WHEN billing_ai_balances.period_end <= $3 THEN 0
		                          ELSE billing_ai_balances.used_tokens END,
		       period_start = CASE WHEN billing_ai_balances.period_end <= $3 THEN $3
		                           ELSE billing_ai_balances.period_start END,
		       period_end = CASE WHEN billing_ai_balances.period_end <= $3 THEN $4
		                         ELSE billing_ai_balances.period_end END,
		       updated_at = now()
		RETURNING user_id, plan_tokens, used_tokens, extra_tokens, period_start, period_end`,
		userID, planTokens, now, periodEnd).
		Scan(&b.UserID, &b.PlanTokens, &b.UsedTokens, &b.ExtraTokens, &b.PeriodStart, &b.PeriodEnd)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// Consume — атомарное списание: сначала тратится квота тарифа, дальше идут
// докупленные токены. Гейт «хватает ли» стоит в WHERE, поэтому параллельные
// обращения не уводят баланс в минус.
func (r *Repo) Consume(ctx context.Context, userID int64, tokens int64) (bool, *domain.AIBalance, error) {
	var b domain.AIBalance
	err := r.pool.QueryRow(ctx, `
		UPDATE billing_ai_balances
		   SET used_tokens = used_tokens + LEAST($2, GREATEST(plan_tokens - used_tokens, 0)),
		       extra_tokens = extra_tokens - GREATEST($2 - GREATEST(plan_tokens - used_tokens, 0), 0),
		       updated_at = now()
		 WHERE user_id = $1
		   AND (plan_tokens - used_tokens) + extra_tokens >= $2
		RETURNING user_id, plan_tokens, used_tokens, extra_tokens, period_start, period_end`,
		userID, tokens).
		Scan(&b.UserID, &b.PlanTokens, &b.UsedTokens, &b.ExtraTokens, &b.PeriodStart, &b.PeriodEnd)
	if err != nil {
		if noRows(err) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, &b, nil
}

func (r *Repo) AddExtraTokens(ctx context.Context, userID int64, tokens int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO billing_ai_balances (user_id, extra_tokens) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE
		   SET extra_tokens = GREATEST(billing_ai_balances.extra_tokens + $2, 0), updated_at = now()`,
		userID, tokens)
	return err
}

func (r *Repo) SetBalance(ctx context.Context, userID int64, planTokens, usedTokens, extraTokens int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO billing_ai_balances (user_id, plan_tokens, used_tokens, extra_tokens)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET plan_tokens = EXCLUDED.plan_tokens,
		    used_tokens = EXCLUDED.used_tokens, extra_tokens = EXCLUDED.extra_tokens, updated_at = now()`,
		userID, planTokens, usedTokens, extraTokens)
	return err
}

func (r *Repo) LogUsage(ctx context.Context, rec domain.AIUsageRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO billing_ai_usage
		    (user_id, company_id, actor_id, feature, model, prompt_tokens, completion_tokens, billed_tokens, own_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		rec.UserID, rec.CompanyID, rec.ActorID, rec.Feature, rec.Model,
		rec.PromptTokens, rec.CompletionTokens, rec.BilledTokens, rec.OwnKey)
	return err
}

func (r *Repo) UsageByFeature(ctx context.Context, userID int64, from time.Time) (map[string]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT feature, COALESCE(sum(billed_tokens), 0)
		  FROM billing_ai_usage
		 WHERE user_id = $1 AND created_at >= $2
		 GROUP BY feature`, userID, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var feature string
		var tokens int64
		if err := rows.Scan(&feature, &tokens); err != nil {
			return nil, err
		}
		out[feature] = tokens
	}
	return out, rows.Err()
}

func (r *Repo) TotalUsage(ctx context.Context, from time.Time) (map[string]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT model, COALESCE(sum(billed_tokens), 0)
		  FROM billing_ai_usage
		 WHERE created_at >= $1
		 GROUP BY model
		 ORDER BY 2 DESC`, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var model string
		var tokens int64
		if err := rows.Scan(&model, &tokens); err != nil {
			return nil, err
		}
		out[model] = tokens
	}
	return out, rows.Err()
}
