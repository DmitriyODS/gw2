package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

// ── Типы юнитов ──────────────────────────────────────────────────

func scanUnitType(row pgx.Row) (*domain.UnitType, error) {
	var ut domain.UnitType
	err := row.Scan(&ut.ID, &ut.Name, &ut.CompanyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ut, nil
}

func (r *Repo) ListUnitTypes(ctx context.Context, companyID int64) ([]*domain.UnitType, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, company_id FROM unit_types WHERE company_id = $1 ORDER BY name`,
		companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.UnitType{}
	for rows.Next() {
		ut, err := scanUnitType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ut)
	}
	return out, rows.Err()
}

func (r *Repo) GetUnitType(ctx context.Context, id int64) (*domain.UnitType, error) {
	return scanUnitType(r.pool.QueryRow(ctx,
		`SELECT id, name, company_id FROM unit_types WHERE id = $1`, id))
}

func (r *Repo) GetUnitTypeByName(ctx context.Context, name string, companyID int64) (*domain.UnitType, error) {
	return scanUnitType(r.pool.QueryRow(ctx,
		`SELECT id, name, company_id FROM unit_types WHERE name = $1 AND company_id = $2`,
		name, companyID))
}

func (r *Repo) CreateUnitType(ctx context.Context, ut *domain.UnitType) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO unit_types (name, company_id) VALUES ($1, $2) RETURNING id`,
		ut.Name, ut.CompanyID).Scan(&ut.ID)
}

func (r *Repo) UpdateUnitTypeName(ctx context.Context, id int64, name string) error {
	_, err := r.pool.Exec(ctx, `UPDATE unit_types SET name = $2 WHERE id = $1`, id, name)
	return err
}

// DeleteUnitType — юниты этого типа уходят каскадом (FK ON DELETE CASCADE).
func (r *Repo) DeleteUnitType(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM unit_types WHERE id = $1`, id)
	return err
}

// ── Отделы ───────────────────────────────────────────────────────

func scanDepartment(row pgx.Row) (*domain.Department, error) {
	var d domain.Department
	err := row.Scan(&d.ID, &d.Name, &d.CompanyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repo) ListDepartments(ctx context.Context, companyID int64) ([]*domain.Department, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, company_id FROM departments WHERE company_id = $1 ORDER BY name`,
		companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Department{}
	for rows.Next() {
		d, err := scanDepartment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repo) GetDepartment(ctx context.Context, id int64) (*domain.Department, error) {
	return scanDepartment(r.pool.QueryRow(ctx,
		`SELECT id, name, company_id FROM departments WHERE id = $1`, id))
}

func (r *Repo) GetDepartmentByName(ctx context.Context, name string, companyID int64) (*domain.Department, error) {
	return scanDepartment(r.pool.QueryRow(ctx,
		`SELECT id, name, company_id FROM departments WHERE name = $1 AND company_id = $2`,
		name, companyID))
}

func (r *Repo) CreateDepartment(ctx context.Context, d *domain.Department) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO departments (name, company_id) VALUES ($1, $2) RETURNING id`,
		d.Name, d.CompanyID).Scan(&d.ID)
}

func (r *Repo) UpdateDepartmentName(ctx context.Context, id int64, name string) error {
	_, err := r.pool.Exec(ctx, `UPDATE departments SET name = $2 WHERE id = $1`, id, name)
	return err
}

func (r *Repo) DeleteDepartment(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM departments WHERE id = $1`, id)
	return err
}

// ── Этапы ────────────────────────────────────────────────────────

func scanStage(row pgx.Row) (*domain.Stage, error) {
	var s domain.Stage
	err := row.Scan(&s.ID, &s.CompanyID, &s.Name, &s.Color, &s.Order)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

const stageColumns = `id, company_id, name, color, "order"`

func (r *Repo) ListStages(ctx context.Context, companyID int64) ([]*domain.Stage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+stageColumns+` FROM stages WHERE company_id = $1 ORDER BY "order", id`,
		companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Stage{}
	for rows.Next() {
		s, err := scanStage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repo) GetStage(ctx context.Context, id int64) (*domain.Stage, error) {
	return scanStage(r.pool.QueryRow(ctx,
		`SELECT `+stageColumns+` FROM stages WHERE id = $1`, id))
}

func (r *Repo) GetStageByName(ctx context.Context, name string, companyID int64) (*domain.Stage, error) {
	return scanStage(r.pool.QueryRow(ctx,
		`SELECT `+stageColumns+` FROM stages WHERE name = $1 AND company_id = $2`,
		name, companyID))
}

func (r *Repo) NextStageOrder(ctx context.Context, companyID int64) (int, error) {
	var max *int
	if err := r.pool.QueryRow(ctx,
		`SELECT MAX("order") FROM stages WHERE company_id = $1`, companyID).Scan(&max); err != nil {
		return 0, err
	}
	if max == nil {
		return 1, nil
	}
	return *max + 1, nil
}

func (r *Repo) CreateStage(ctx context.Context, s *domain.Stage) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO stages (name, color, company_id, "order") VALUES ($1, $2, $3, $4) RETURNING id`,
		s.Name, s.Color, s.CompanyID, s.Order).Scan(&s.ID)
}

var allowedStageFields = map[string]bool{"name": true, "color": true}

func (r *Repo) UpdateStageFields(ctx context.Context, id int64, fields map[string]any) error {
	return updateFields(ctx, r.pool, "stages", allowedStageFields, id, fields)
}

func (r *Repo) DeleteStage(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM stages WHERE id = $1`, id)
	return err
}

// ReorderStages — порядок = позиция в ordered_ids; id чужих компаний
// игнорируются (как stage_repo.reorder).
func (r *Repo) ReorderStages(ctx context.Context, companyID int64, orderedIDs []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	for idx, id := range orderedIDs {
		if _, err := tx.Exec(ctx,
			`UPDATE stages SET "order" = $1 WHERE id = $2 AND company_id = $3`,
			idx+1, id, companyID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ── Комментарии ──────────────────────────────────────────────────

const commentColumns = `
	c.id, c.task_id, c.author_id, c.text, c.created_at, c.updated_at, c.deleted_at,
	a.id, a.fio, a.avatar_path`

const commentFrom = `
	FROM comments c
	JOIN users a ON a.id = c.author_id`

func scanComment(row pgx.Row) (*domain.Comment, error) {
	var (
		c      domain.Comment
		aID    int64
		aFIO   string
		aAv    *string
	)
	err := row.Scan(&c.ID, &c.TaskID, &c.AuthorID, &c.Text,
		&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
		&aID, &aFIO, &aAv)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.Author = &domain.UserRef{ID: aID, FIO: aFIO, AvatarPath: aAv}
	return &c, nil
}

func (r *Repo) GetComment(ctx context.Context, id int64) (*domain.Comment, error) {
	return scanComment(r.pool.QueryRow(ctx,
		"SELECT"+commentColumns+commentFrom+" WHERE c.id = $1", id))
}

func (r *Repo) ListComments(ctx context.Context, taskID int64) ([]*domain.Comment, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT"+commentColumns+commentFrom+
			" WHERE c.task_id = $1 AND c.deleted_at IS NULL ORDER BY c.created_at ASC",
		taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Comment{}
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repo) CreateComment(ctx context.Context, c *domain.Comment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO comments (task_id, author_id, text, created_at)
		VALUES ($1, $2, $3, now())
		RETURNING id, created_at`,
		c.TaskID, c.AuthorID, c.Text).Scan(&c.ID, &c.CreatedAt)
}

func (r *Repo) UpdateCommentText(ctx context.Context, id int64, text string, updatedAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE comments SET text = $2, updated_at = $3 WHERE id = $1`, id, text, updatedAt)
	return err
}

func (r *Repo) SoftDeleteComment(ctx context.Context, id int64, deletedAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE comments SET deleted_at = $2 WHERE id = $1`, id, deletedAt)
	return err
}

func (r *Repo) CountNewComments(ctx context.Context, taskID, userID int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM comments c
		LEFT JOIN task_comment_seen s ON s.task_id = c.task_id AND s.user_id = $2
		WHERE c.task_id = $1
		  AND c.deleted_at IS NULL
		  AND c.author_id <> $2
		  AND c.created_at > COALESCE(s.last_seen_at, '-infinity'::timestamptz)`,
		taskID, userID).Scan(&n)
	return n, err
}

func (r *Repo) MarkCommentsSeen(ctx context.Context, taskID, userID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO task_comment_seen (user_id, task_id, last_seen_at)
		VALUES ($2, $1, now())
		ON CONFLICT (user_id, task_id) DO UPDATE SET last_seen_at = now()`,
		taskID, userID)
	return err
}

// ── Упоминания (@логин) в комментариях ──

func (r *Repo) ResolveMentions(ctx context.Context, companyID int64, logins []string) (map[string]int64, error) {
	out := map[string]int64{}
	if len(logins) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT lower(u.login), u.id
		  FROM users u
		  JOIN user_companies uc ON uc.user_id = u.id
		 WHERE uc.company_id = $1 AND lower(u.login) = ANY($2)`,
		companyID, logins)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var login string
		var id int64
		if err := rows.Scan(&login, &id); err != nil {
			return nil, err
		}
		out[login] = id
	}
	return out, rows.Err()
}

func (r *Repo) CreateMentions(ctx context.Context, taskID, commentID int64, userIDs []int64) error {
	if len(userIDs) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO task_mentions (task_id, comment_id, user_id)
		SELECT $1, $2, uid FROM unnest($3::bigint[]) AS uid`,
		taskID, commentID, userIDs)
	return err
}

func (r *Repo) MentionCounts(ctx context.Context, taskIDs []int64, userID int64) (map[int64]int, error) {
	out := map[int64]int{}
	if len(taskIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT task_id, count(*) FROM task_mentions
		 WHERE user_id = $1 AND task_id = ANY($2) AND seen_at IS NULL
		 GROUP BY task_id`, userID, taskIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var taskID int64
		var n int
		if err := rows.Scan(&taskID, &n); err != nil {
			return nil, err
		}
		out[taskID] = n
	}
	return out, rows.Err()
}

func (r *Repo) MarkMentionsSeen(ctx context.Context, taskID, userID int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE task_mentions SET seen_at = now()
		 WHERE task_id = $1 AND user_id = $2 AND seen_at IS NULL`, taskID, userID)
	return err
}
