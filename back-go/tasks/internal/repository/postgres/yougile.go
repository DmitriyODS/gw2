package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

var _ domain.YougileRepository = (*Repo)(nil)

// ygAccountCols — поля привязки; is_active выбирает, чьим ключом работает
// интеграция сейчас (аккаунтов у человека бывает несколько).
const ygAccountCols = `id, user_id, company_id, yg_company_id, yg_user_id, yg_login,
	key_ciphertext, key_fingerprint, last_validated_at, is_active`

func scanYgAccount(row pgx.Row) (*domain.YougileAccount, error) {
	var a domain.YougileAccount
	err := row.Scan(&a.ID, &a.UserID, &a.CompanyID, &a.YgCompanyID, &a.YgUserID, &a.YgLogin,
		&a.KeyCiphertext, &a.KeyFingerprint, &a.LastValidatedAt, &a.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// GetYougileAccount — АКТИВНАЯ привязка пользователя: ею работают импорт и
// экспорт карточек. Остальные подключения ждут переключения.
func (r *Repo) GetYougileAccount(ctx context.Context, userID int64) (*domain.YougileAccount, error) {
	return scanYgAccount(r.pool.QueryRow(ctx,
		`SELECT `+ygAccountCols+` FROM user_yougile_accounts
		  WHERE user_id = $1 AND is_active`, userID))
}

// ListYougileAccounts — все подключения пользователя (карточка «Аккаунт»).
func (r *Repo) ListYougileAccounts(ctx context.Context, userID int64) ([]*domain.YougileAccount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+ygAccountCols+` FROM user_yougile_accounts
		  WHERE user_id = $1 ORDER BY is_active DESC, yg_login`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.YougileAccount{}
	for rows.Next() {
		a, err := scanYgAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

/* SetActiveYougileAccount — переключиться на другое подключение. Двумя
   UPDATE в транзакции: частичный уникальный индекс не даст двум строкам быть
   активными одновременно, поэтому сначала снимаем флаг со всех. Чужой id не
   найдётся — user_id стоит в условии. */
func (r *Repo) SetActiveYougileAccount(ctx context.Context, userID, accountID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE user_yougile_accounts SET is_active = FALSE, updated_at = now()
		  WHERE user_id = $1 AND is_active`, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx,
		`UPDATE user_yougile_accounts SET is_active = TRUE, updated_at = now()
		  WHERE user_id = $1 AND id = $2`, userID, accountID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NewError("USER_NOT_CONNECTED", "Такое подключение YouGile не найдено", 404)
	}
	return tx.Commit(ctx)
}

func (r *Repo) UpsertYougileAccount(ctx context.Context, acc *domain.YougileAccount) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO user_yougile_accounts
		       (user_id, company_id, yg_company_id, yg_user_id, yg_login,
		        key_ciphertext, key_fingerprint, last_validated_at,
		        created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())
		ON CONFLICT (user_id, yg_company_id) DO UPDATE SET
		       yg_company_id = EXCLUDED.yg_company_id,
		       yg_user_id = EXCLUDED.yg_user_id,
		       yg_login = EXCLUDED.yg_login,
		       key_ciphertext = EXCLUDED.key_ciphertext,
		       key_fingerprint = EXCLUDED.key_fingerprint,
		       last_validated_at = EXCLUDED.last_validated_at,
		       updated_at = now()
		RETURNING id`,
		acc.UserID, acc.CompanyID, acc.YgCompanyID, acc.YgUserID, acc.YgLogin,
		acc.KeyCiphertext, acc.KeyFingerprint, acc.LastValidatedAt,
	).Scan(&acc.ID)
}

// DeleteYougileAccount — отключить конкретную привязку (0 — все привязки
// пользователя). Если ушла активная, активной становится любая оставшаяся:
// иначе интеграция молча перестала бы работать при живых подключениях.
func (r *Repo) DeleteYougileAccount(ctx context.Context, userID, accountID int64) error {
	if accountID == 0 {
		_, err := r.pool.Exec(ctx,
			`DELETE FROM user_yougile_accounts WHERE user_id = $1`, userID)
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM user_yougile_accounts WHERE user_id = $1 AND id = $2`,
		userID, accountID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE user_yougile_accounts SET is_active = TRUE, updated_at = now()
		 WHERE id = (SELECT id FROM user_yougile_accounts
		              WHERE user_id = $1 ORDER BY updated_at DESC LIMIT 1)
		   AND NOT EXISTS (SELECT 1 FROM user_yougile_accounts
		                    WHERE user_id = $1 AND is_active)`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repo) GetYougileCompany(ctx context.Context, companyID int64) (*domain.YougileCompany, error) {
	var (
		c        domain.YougileCompany
		settings map[string]any
	)
	err := r.pool.QueryRow(ctx, `
		SELECT id, settings, yg_company_id, yg_company_name, yg_project_id,
		       yg_project_title, yg_board_id, yg_board_title,
		       yg_first_column_id, yg_completed_column_id,
		       yg_webhook_id, yg_webhook_secret
		  FROM companies WHERE id = $1`, companyID).
		Scan(&c.ID, &settings, &c.YgCompanyID, &c.YgCompanyName, &c.YgProjectID,
			&c.YgProjectTitle, &c.YgBoardID, &c.YgBoardTitle,
			&c.YgFirstColumnID, &c.YgCompletedColumnID,
			&c.YgWebhookID, &c.YgWebhookSecret)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// «Включено» — строго settings.uses_yougile == true (дефолт False в
	// DEFAULT_SETTINGS компании; это НЕ fail-open флаг _yougile_enabled).
	if v, ok := settings["uses_yougile"].(bool); ok {
		c.UsesYougile = v
	}
	return &c, nil
}

// allowedYougileCompanyFields — yg_*-колонки companies, которые правит
// интеграция.
var allowedYougileCompanyFields = map[string]bool{
	"yg_company_id": true, "yg_company_name": true,
	"yg_project_id": true, "yg_project_title": true,
	"yg_board_id": true, "yg_board_title": true,
	"yg_first_column_id": true, "yg_completed_column_id": true,
	"yg_webhook_id": true, "yg_webhook_secret": true,
}

func (r *Repo) UpdateYougileCompanyFields(ctx context.Context, companyID int64, fields map[string]any) error {
	return updateFields(ctx, r.pool, "companies", allowedYougileCompanyFields, companyID, fields)
}

func (r *Repo) SetCompanyUsesYougile(ctx context.Context, companyID int64, enabled bool) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE companies
		   SET settings = jsonb_set(COALESCE(settings, '{}'::jsonb),
		                            '{uses_yougile}', to_jsonb($2::boolean))
		 WHERE id = $1`, companyID, enabled)
	return err
}

func (r *Repo) TaskByYougileID(ctx context.Context, companyID int64, ygTaskID string) (*domain.Task, error) {
	t, err := scanTask(r.pool.QueryRow(ctx,
		"SELECT"+taskColumns+taskFrom+" WHERE t.company_id = $1 AND t.yougile_task_id = $2",
		companyID, ygTaskID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}
