package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.Repository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const topicCols = `id, company_id, name, color, icon, created_by, created_at`

func scanTopic(row pgx.Row) (*domain.Topic, error) {
	var t domain.Topic
	err := row.Scan(&t.ID, &t.CompanyID, &t.Name, &t.Color, &t.Icon, &t.CreatedBy, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repo) ListTopics(ctx context.Context, companyID int64) ([]*domain.Topic, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+topicCols+` FROM portal_topics WHERE company_id = $1 ORDER BY name`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Topic{}
	for rows.Next() {
		t, err := scanTopic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repo) GetTopic(ctx context.Context, id int64) (*domain.Topic, error) {
	return scanTopic(r.pool.QueryRow(ctx, `SELECT `+topicCols+` FROM portal_topics WHERE id = $1`, id))
}

func (r *Repo) CreateTopic(ctx context.Context, t *domain.Topic) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO portal_topics (company_id, name, color, icon, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		t.CompanyID, t.Name, t.Color, t.Icon, t.CreatedBy,
	).Scan(&t.ID, &t.CreatedAt)
}

func (r *Repo) UpdateTopic(ctx context.Context, id int64, name string, color, icon *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE portal_topics SET name = $2, color = $3, icon = $4 WHERE id = $1`,
		id, name, color, icon)
	return err
}

func (r *Repo) DeleteTopic(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM portal_topics WHERE id = $1`, id)
	return err
}
