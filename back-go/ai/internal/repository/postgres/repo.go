// Package postgres — персистентность aisvc (pgx, raw SQL по таблицам, схему
// которых ведёт Alembic во Flask): AI-поля companies, task_embeddings
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
	err := r.pool.QueryRow(ctx, `
		SELECT id, ai_enabled, ai_api_key_enc, ai_key_hint, ai_model_chat, ai_model_embedding
		  FROM companies
		 WHERE id = $1`, companyID).
		Scan(&c.ID, &c.Enabled, &c.APIKeyEnc, &c.KeyHint, &c.ModelChat, &c.ModelEmbedding)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repo) UpdateCompanyAI(ctx context.Context, c *domain.CompanyAI) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE companies
		   SET ai_enabled = $2,
		       ai_api_key_enc = $3,
		       ai_key_hint = $4,
		       ai_model_chat = $5,
		       ai_model_embedding = $6
		 WHERE id = $1`,
		c.ID, c.Enabled, c.APIKeyEnc, c.KeyHint, c.ModelChat, c.ModelEmbedding)
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
