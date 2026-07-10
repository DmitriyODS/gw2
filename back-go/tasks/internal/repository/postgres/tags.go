package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

// Теги задач: справочник компании (tags) + связка many-to-many (task_tags).

func (r *Repo) ListTags(ctx context.Context, companyID int64) ([]*domain.Tag, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, name, color FROM tags
		 WHERE company_id = $1 ORDER BY lower(name)`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Tag{}
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.CompanyID, &t.Name, &t.Color); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

func (r *Repo) GetTag(ctx context.Context, id int64) (*domain.Tag, error) {
	var t domain.Tag
	err := r.pool.QueryRow(ctx,
		`SELECT id, company_id, name, color FROM tags WHERE id = $1`, id).
		Scan(&t.ID, &t.CompanyID, &t.Name, &t.Color)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repo) GetTagByName(ctx context.Context, name string, companyID int64) (*domain.Tag, error) {
	var t domain.Tag
	err := r.pool.QueryRow(ctx, `
		SELECT id, company_id, name, color FROM tags
		 WHERE company_id = $1 AND lower(name) = lower($2)`, companyID, name).
		Scan(&t.ID, &t.CompanyID, &t.Name, &t.Color)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repo) CreateTag(ctx context.Context, t *domain.Tag) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO tags (company_id, name, color) VALUES ($1, $2, $3)
		RETURNING id`, t.CompanyID, t.Name, t.Color).Scan(&t.ID)
}

var allowedTagFields = map[string]bool{"name": true, "color": true}

func (r *Repo) UpdateTagFields(ctx context.Context, id int64, fields map[string]any) error {
	return updateFields(ctx, r.pool, "tags", allowedTagFields, id, fields)
}

func (r *Repo) DeleteTag(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tags WHERE id = $1`, id)
	return err
}

// SetTaskTags — полная замена набора тегов задачи одной транзакцией.
func (r *Repo) SetTaskTags(ctx context.Context, taskID int64, tagIDs []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // после Commit — no-op

	if _, err := tx.Exec(ctx, `DELETE FROM task_tags WHERE task_id = $1`, taskID); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, taskID, tagID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repo) TagsByTasks(ctx context.Context, taskIDs []int64) (map[int64][]domain.TagRef, error) {
	out := map[int64][]domain.TagRef{}
	if len(taskIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT tt.task_id, tg.id, tg.name, tg.color
		  FROM task_tags tt
		  JOIN tags tg ON tg.id = tt.tag_id
		 WHERE tt.task_id = ANY($1)
		 ORDER BY lower(tg.name)`, taskIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var taskID int64
		var t domain.TagRef
		if err := rows.Scan(&taskID, &t.ID, &t.Name, &t.Color); err != nil {
			return nil, err
		}
		out[taskID] = append(out[taskID], t)
	}
	return out, rows.Err()
}
