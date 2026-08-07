// Package postgres — персистентность aisvc (pgx, raw SQL по таблицам, схему
// которых ведёт migrate-контейнер goose): AI-поля companies, task_embeddings
// (pgvector) + read-only лукапы tasks/departments/users.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.Repository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ── Компании (AI-срез) ───────────────────────────────────────────

func (r *Repo) GetCompanyAI(ctx context.Context, companyID int64) (*domain.CompanyAI, error) {
	var c domain.CompanyAI
	var owner *int64
	err := r.pool.QueryRow(ctx, `
		SELECT id, ai_enabled, ai_shared, ai_feat_search, ai_feat_tv_fact, created_by
		  FROM companies
		 WHERE id = $1`, companyID).
		Scan(&c.ID, &c.Enabled, &c.Shared, &c.FeatSearch, &c.FeatTVFact, &owner)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if owner != nil {
		c.OwnerID = *owner
	}
	return &c, nil
}

// GetUserAI — личные ИИ-настройки; nil без ошибки, если человек ещё не
// подключал свой ключ (строки в user_ai_settings просто нет).
func (r *Repo) GetUserAI(ctx context.Context, userID int64) (*domain.UserAI, error) {
	var u domain.UserAI
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, enabled, api_key_enc, key_hint, model_chat,
		       api_base_url, feat_assistant, feat_notes
		  FROM user_ai_settings
		 WHERE user_id = $1`, userID).
		Scan(&u.UserID, &u.Enabled, &u.APIKeyEnc, &u.KeyHint, &u.ModelChat,
			&u.APIBaseURL, &u.FeatAssistant, &u.FeatNotes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repo) UpsertUserAI(ctx context.Context, u *domain.UserAI) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_ai_settings
		    (user_id, enabled, api_key_enc, key_hint, model_chat, api_base_url, feat_assistant, feat_notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE
		   SET enabled = EXCLUDED.enabled,
		       api_key_enc = EXCLUDED.api_key_enc,
		       key_hint = EXCLUDED.key_hint,
		       model_chat = EXCLUDED.model_chat,
		       api_base_url = EXCLUDED.api_base_url,
		       feat_assistant = EXCLUDED.feat_assistant,
		       feat_notes = EXCLUDED.feat_notes,
		       updated_at = now()`,
		u.UserID, u.Enabled, u.APIKeyEnc, u.KeyHint, u.ModelChat,
		u.APIBaseURL, u.FeatAssistant, u.FeatNotes)
	return err
}

// MembershipLevel — уровень роли пользователя в компании; 0 — не член.
func (r *Repo) MembershipLevel(ctx context.Context, userID, companyID int64) (int, error) {
	var level int
	err := r.pool.QueryRow(ctx, `
		SELECT r.level
		  FROM user_companies uc
		  JOIN roles r ON r.id = uc.role_id
		 WHERE uc.user_id = $1 AND uc.company_id = $2`, userID, companyID).Scan(&level)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return level, nil
}

func (r *Repo) UpdateCompanyAI(ctx context.Context, c *domain.CompanyAI) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE companies
		   SET ai_enabled = $2,
		       ai_shared = $3,
		       ai_feat_search = $4,
		       ai_feat_tv_fact = $5
		 WHERE id = $1`,
		c.ID, c.Enabled, c.Shared, c.FeatSearch, c.FeatTVFact)
	return err
}

// ── Подсчёты индексации ──────────────────────────────────────────

func (r *Repo) CountTasks(ctx context.Context, companyID int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tasks WHERE company_id = $1`, companyID).Scan(&n)
	return n, err
}

func (r *Repo) CountEmbeddings(ctx context.Context, companyID int64, model string) (int, error) {
	sql := `SELECT COUNT(*) FROM task_embeddings WHERE company_id = $1`
	args := []any{companyID}
	if model != "" {
		sql += ` AND model = $2`
		args = append(args, model)
	}
	var n int
	err := r.pool.QueryRow(ctx, sql, args...).Scan(&n)
	return n, err
}

func (r *Repo) FindUnindexedTaskIDs(ctx context.Context, companyID int64, model string) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id
		  FROM tasks t
		  LEFT JOIN task_embeddings e ON e.task_id = t.id
		 WHERE t.company_id = $1
		   AND (e.task_id IS NULL OR e.model <> $2)`, companyID, model)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ── Задачи (read-only, текст для эмбеддинга) ─────────────────────

const taskTextCols = `t.id, t.company_id, t.name, d.name, u.fio`

const taskTextFrom = `
	FROM tasks t
	LEFT JOIN departments d ON d.id = t.department_id
	LEFT JOIN users u ON u.id = t.responsible_user_id `

func scanTaskText(row pgx.Row) (*domain.TaskText, error) {
	var t domain.TaskText
	err := row.Scan(&t.ID, &t.CompanyID, &t.Name, &t.DepartmentName, &t.ResponsibleFIO)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *Repo) GetTaskText(ctx context.Context, taskID int64) (*domain.TaskText, error) {
	return scanTaskText(r.pool.QueryRow(ctx,
		`SELECT `+taskTextCols+taskTextFrom+`WHERE t.id = $1`, taskID))
}

func (r *Repo) ListTaskTexts(ctx context.Context, ids []int64) ([]*domain.TaskText, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+taskTextCols+taskTextFrom+`WHERE t.id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.TaskText
	for rows.Next() {
		t, err := scanTaskText(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ── Эмбеддинги (pgvector) ────────────────────────────────────────

// vecToStr — pgvector принимает строку вида "[0.1,0.2,...]"; форматирование
// %.6f — как _vec_to_str во Flask.
func vecToStr(v []float32) string {
	var b strings.Builder
	b.Grow(len(v)*10 + 2)
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%.6f", x)
	}
	b.WriteByte(']')
	return b.String()
}

func (r *Repo) UpsertEmbedding(ctx context.Context, taskID, companyID int64, vector []float32, model string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO task_embeddings (task_id, company_id, embedding, model, updated_at)
		VALUES ($1, $2, CAST($3 AS vector), $4, $5)
		ON CONFLICT (task_id) DO UPDATE
		  SET company_id = EXCLUDED.company_id,
		      embedding  = EXCLUDED.embedding,
		      model      = EXCLUDED.model,
		      updated_at = EXCLUDED.updated_at`,
		taskID, companyID, vecToStr(vector), model, time.Now().UTC())
	return err
}

func (r *Repo) SearchEmbeddings(ctx context.Context, companyID int64, vector []float32, model string, limit int) ([]domain.SearchHit, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT task_id,
		       1 - (embedding <=> CAST($1 AS vector)) AS score
		  FROM task_embeddings
		 WHERE company_id = $2
		   AND model = $3
		 ORDER BY embedding <=> CAST($1 AS vector)
		 LIMIT $4`,
		vecToStr(vector), companyID, model, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.SearchHit
	for rows.Next() {
		var h domain.SearchHit
		if err := rows.Scan(&h.TaskID, &h.Score); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ── Платформенный ИИ (ai_platform_settings + ai_models) ──────────

// GetPlatformAI — единственная строка настроек; её заводит миграция, поэтому
// «нет строки» трактуем как выключенный ИИ, а не как ошибку.
func (r *Repo) GetPlatformAI(ctx context.Context) (*domain.PlatformAI, error) {
	var p domain.PlatformAI
	err := r.pool.QueryRow(ctx, `
		SELECT enabled, api_key_enc, key_hint, base_url, model_chat, model_embedding, model_support
		  FROM ai_platform_settings WHERE id = 1`).
		Scan(&p.Enabled, &p.APIKeyEnc, &p.KeyHint, &p.BaseURL, &p.ModelChat, &p.ModelEmbedding, &p.ModelSupport)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &domain.PlatformAI{}, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repo) UpdatePlatformAI(ctx context.Context, p *domain.PlatformAI) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ai_platform_settings
		    (id, enabled, api_key_enc, key_hint, base_url, model_chat, model_embedding, model_support, updated_at)
		VALUES (1, $1, $2, $3, $4, $5, $6, $7, now())
		ON CONFLICT (id) DO UPDATE
		   SET enabled = EXCLUDED.enabled,
		       api_key_enc = EXCLUDED.api_key_enc,
		       key_hint = EXCLUDED.key_hint,
		       base_url = EXCLUDED.base_url,
		       model_chat = EXCLUDED.model_chat,
		       model_embedding = EXCLUDED.model_embedding,
		       model_support = EXCLUDED.model_support,
		       updated_at = now()`,
		p.Enabled, p.APIKeyEnc, p.KeyHint, p.BaseURL, p.ModelChat, p.ModelEmbedding, p.ModelSupport)
	return err
}

func (r *Repo) ListModels(ctx context.Context) ([]*domain.AIModel, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT code, title, kind, price_per_mtok, selectable, is_active, sort
		  FROM ai_models ORDER BY sort, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.AIModel{}
	for rows.Next() {
		var m domain.AIModel
		if err := rows.Scan(&m.Code, &m.Title, &m.Kind, &m.PricePerMTok,
			&m.Selectable, &m.IsActive, &m.Sort); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// UpsertModel — правка каталога супер-админом (цена задаёт стоимость
// обращения в токенах доступа).
func (r *Repo) UpsertModel(ctx context.Context, m *domain.AIModel) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ai_models (code, title, kind, price_per_mtok, selectable, is_active, sort, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())
		ON CONFLICT (code) DO UPDATE
		   SET title = EXCLUDED.title, kind = EXCLUDED.kind,
		       price_per_mtok = EXCLUDED.price_per_mtok, selectable = EXCLUDED.selectable,
		       is_active = EXCLUDED.is_active, sort = EXCLUDED.sort, updated_at = now()`,
		m.Code, m.Title, m.Kind, m.PricePerMTok, m.Selectable, m.IsActive, m.Sort)
	return err
}

// CompanyOwner — создатель компании: его токены тратят ИИ-возможности компании.
func (r *Repo) CompanyOwner(ctx context.Context, companyID int64) (int64, error) {
	var owner *int64
	if err := r.pool.QueryRow(ctx, `SELECT created_by FROM companies WHERE id = $1`, companyID).
		Scan(&owner); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	if owner == nil {
		return 0, nil
	}
	return *owner, nil
}
