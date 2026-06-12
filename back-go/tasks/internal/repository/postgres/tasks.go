// Package postgres — персистентность tasksvc (pgx, raw SQL по таблицам,
// схему которых ведёт Alembic во Flask): tasks, units, unit_types, stages,
// departments, comments, favorites, user_task_colors + read-only users.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

var (
	_ domain.TaskRepository       = (*Repo)(nil)
	_ domain.UnitRepository       = (*Repo)(nil)
	_ domain.UnitTypeRepository   = (*Repo)(nil)
	_ domain.DepartmentRepository = (*Repo)(nil)
	_ domain.StageRepository      = (*Repo)(nil)
	_ domain.CommentRepository    = (*Repo)(nil)
	_ domain.StatsRepository      = (*Repo)(nil)
)

const taskColumns = `
	t.id, t.name, t.created_at, t.author_id, t.responsible_user_id, t.link_yougile,
	t.received_at, t.department_id, t.stage_id, t.deadline, t.is_archived,
	t.archived_at, t.company_id,
	t.yougile_task_id, t.yougile_id_short, t.yougile_project_id,
	t.yougile_board_id, t.yougile_column_id,
	a.id, a.fio, a.avatar_path,
	r.id, r.fio, r.avatar_path,
	d.id, d.name,
	s.id, s.name, s.color, s."order"`

const taskFrom = `
	FROM tasks t
	JOIN users a ON a.id = t.author_id
	LEFT JOIN users r ON r.id = t.responsible_user_id
	JOIN departments d ON d.id = t.department_id
	LEFT JOIN stages s ON s.id = t.stage_id`

func scanTask(row pgx.Row) (*domain.Task, error) {
	var (
		t                       domain.Task
		aID                     int64
		aFIO                    string
		aAvatar                 *string
		rID                     *int64
		rFIO, rAvatar           *string
		dID                     int64
		dName                   string
		sID                     *int64
		sName, sColor           *string
		sOrder                  *int
	)
	err := row.Scan(
		&t.ID, &t.Name, &t.CreatedAt, &t.AuthorID, &t.ResponsibleUserID, &t.LinkYougile,
		&t.ReceivedAt, &t.DepartmentID, &t.StageID, &t.Deadline, &t.IsArchived,
		&t.ArchivedAt, &t.CompanyID,
		&t.YougileTaskID, &t.YougileIDShort, &t.YougileProjectID,
		&t.YougileBoardID, &t.YougileColumnID,
		&aID, &aFIO, &aAvatar,
		&rID, &rFIO, &rAvatar,
		&dID, &dName,
		&sID, &sName, &sColor, &sOrder,
	)
	if err != nil {
		return nil, err
	}
	t.Author = &domain.UserRef{ID: aID, FIO: aFIO, AvatarPath: aAvatar}
	if rID != nil {
		t.Responsible = &domain.UserRef{ID: *rID, AvatarPath: rAvatar}
		if rFIO != nil {
			t.Responsible.FIO = *rFIO
		}
	}
	t.Department = &domain.DeptRef{ID: dID, Name: dName}
	if sID != nil {
		t.Stage = &domain.StageRef{ID: *sID}
		if sName != nil {
			t.Stage.Name = *sName
		}
		if sColor != nil {
			t.Stage.Color = *sColor
		}
		if sOrder != nil {
			t.Stage.Order = *sOrder
		}
	}
	return &t, nil
}

func (r *Repo) GetTask(ctx context.Context, id int64) (*domain.Task, error) {
	t, err := scanTask(r.pool.QueryRow(ctx,
		"SELECT"+taskColumns+taskFrom+" WHERE t.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

// ListTasks — фильтры/сортировки/пагинация как task_repo.get_list во Flask.
func (r *Repo) ListTasks(ctx context.Context, f domain.TaskListFilter) ([]*domain.Task, int, error) {
	// Пустая семантическая выдача — сразу пустой список.
	if f.OrderedSet && len(f.OrderedIDs) == 0 {
		return []*domain.Task{}, 0, nil
	}

	var (
		where []string
		args  []any
	)
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if f.CompanyID != nil {
		where = append(where, "t.company_id = "+arg(*f.CompanyID))
	}
	switch f.Tab {
	case "active":
		where = append(where, "t.is_archived = FALSE")
	case "favorites":
		where = append(where, "t.is_archived = FALSE",
			"EXISTS (SELECT 1 FROM favorites f WHERE f.task_id = t.id AND f.user_id = "+arg(f.CurrentUserID)+")")
	case "archive":
		where = append(where, "t.is_archived = TRUE")
	}
	if f.Search != "" && !f.OrderedSet {
		where = append(where, "lower(t.name) LIKE "+arg("%"+strings.ToLower(strings.TrimSpace(f.Search))+"%"))
	}
	if f.OrderedSet {
		where = append(where, "t.id = ANY("+arg(f.OrderedIDs)+")")
	}
	if f.DeptID != nil {
		where = append(where, "t.department_id = "+arg(*f.DeptID))
	}
	if f.StageID != nil {
		where = append(where, "t.stage_id = "+arg(*f.StageID))
	}
	if f.ResponsibleUserID != nil {
		where = append(where, "t.responsible_user_id = "+arg(*f.ResponsibleUserID))
	}
	if f.ReceivedFrom != nil {
		where = append(where, "t.received_at >= "+arg(*f.ReceivedFrom))
	}
	if f.ReceivedTo != nil {
		where = append(where, "t.received_at <= "+arg(*f.ReceivedTo))
	}
	if f.AuthorID != nil {
		where = append(where, "t.author_id = "+arg(*f.AuthorID))
	}
	switch f.HasUnits {
	case "none":
		where = append(where, "NOT EXISTS (SELECT 1 FROM units u WHERE u.task_id = t.id)")
	case "mine":
		where = append(where,
			"EXISTS (SELECT 1 FROM units u WHERE u.task_id = t.id AND u.user_id = "+arg(f.CurrentUserID)+")")
	}

	cond := ""
	if len(where) > 0 {
		cond = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM tasks t"+strings.Replace(taskFrom, "FROM tasks t", "", 1)+cond,
		args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Сортировка. Семантический режим — по позиции в ordered_ids.
	var order string
	switch {
	case f.OrderedSet:
		order = " ORDER BY array_position(" + arg(f.OrderedIDs) + ", t.id)"
	case f.Sort == "created_at":
		order = " ORDER BY t.created_at DESC"
	case f.Sort == "received_at":
		order = " ORDER BY t.received_at DESC"
	case f.Sort == "deadline":
		order = " ORDER BY t.deadline ASC NULLS LAST"
	default: // last_activity
		order = ` ORDER BY (SELECT MAX(u.datetime_start) FROM units u WHERE u.task_id = t.id) DESC NULLS LAST,
			t.created_at DESC`
	}

	offset := (f.Page - 1) * f.PerPage
	sql := "SELECT" + taskColumns + taskFrom + cond + order +
		" OFFSET " + arg(offset) + " LIMIT " + arg(f.PerPage)
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []*domain.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, t)
	}
	return items, total, rows.Err()
}

func (r *Repo) CreateTask(ctx context.Context, t *domain.Task) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO tasks (name, author_id, department_id, company_id, link_yougile,
		                   deadline, responsible_user_id, stage_id,
		                   received_at, created_at, is_archived)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, now()), now(), FALSE)
		RETURNING id`,
		t.Name, t.AuthorID, t.DepartmentID, t.CompanyID, t.LinkYougile,
		t.Deadline, t.ResponsibleUserID, t.StageID, nilTime(t.ReceivedAt),
	).Scan(&t.ID)
}

// allowedTaskFields — колонки, которые сервис может менять точечно.
var allowedTaskFields = map[string]bool{
	"name": true, "link_yougile": true, "department_id": true,
	"received_at": true, "deadline": true, "responsible_user_id": true,
	"stage_id": true, "is_archived": true, "archived_at": true,
}

func (r *Repo) UpdateTaskFields(ctx context.Context, id int64, fields map[string]any) error {
	return updateFields(ctx, r.pool, "tasks", allowedTaskFields, id, fields)
}

func (r *Repo) DeleteTask(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	return err
}

func (r *Repo) HasActiveUnit(ctx context.Context, taskID int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM units WHERE task_id = $1 AND datetime_end IS NULL)`,
		taskID).Scan(&ok)
	return ok, err
}

func (r *Repo) HasAnyUnits(ctx context.Context, taskID int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM units WHERE task_id = $1)`, taskID).Scan(&ok)
	return ok, err
}

func (r *Repo) IsFavorite(ctx context.Context, taskID, userID int64) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM favorites WHERE task_id = $1 AND user_id = $2)`,
		taskID, userID).Scan(&ok)
	return ok, err
}

func (r *Repo) ToggleFavorite(ctx context.Context, taskID, userID int64) (bool, error) {
	res, err := r.pool.Exec(ctx,
		`DELETE FROM favorites WHERE task_id = $1 AND user_id = $2`, taskID, userID)
	if err != nil {
		return false, err
	}
	if res.RowsAffected() > 0 {
		return false, nil
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO favorites (task_id, user_id) VALUES ($1, $2)`, taskID, userID)
	return true, err
}

func (r *Repo) ActiveUsers(ctx context.Context, taskID int64) ([]domain.UserRef, error) {
	return r.userRefs(ctx, `
		SELECT us.id, us.fio, us.avatar_path
		  FROM users us
		  JOIN units u ON u.user_id = us.id
		 WHERE u.task_id = $1 AND u.datetime_end IS NULL`, taskID)
}

func (r *Repo) Contributors(ctx context.Context, taskID int64) ([]domain.UserRef, error) {
	return r.userRefs(ctx, `
		SELECT DISTINCT us.id, us.fio, us.avatar_path
		  FROM users us
		  JOIN units u ON u.user_id = us.id
		 WHERE u.task_id = $1
		 ORDER BY us.fio ASC`, taskID)
}

func (r *Repo) userRefs(ctx context.Context, sql string, args ...any) ([]domain.UserRef, error) {
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.UserRef{}
	for rows.Next() {
		var u domain.UserRef
		if err := rows.Scan(&u.ID, &u.FIO, &u.AvatarPath); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repo) UserColor(ctx context.Context, taskID, userID int64) (*string, error) {
	var color *string
	err := r.pool.QueryRow(ctx,
		`SELECT color FROM user_task_colors WHERE task_id = $1 AND user_id = $2`,
		taskID, userID).Scan(&color)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return color, err
}

func (r *Repo) SetUserColor(ctx context.Context, taskID, userID int64, color *string) error {
	if color == nil {
		_, err := r.pool.Exec(ctx,
			`DELETE FROM user_task_colors WHERE task_id = $1 AND user_id = $2`,
			taskID, userID)
		return err
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_task_colors (task_id, user_id, color)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, task_id) DO UPDATE SET color = EXCLUDED.color`,
		taskID, userID, *color)
	return err
}

// Enrichment — батч-обогащение списка задач (как 4 батч-запроса во Flask).
func (r *Repo) Enrichment(ctx context.Context, taskIDs []int64, userID int64) (*domain.TaskEnrichment, error) {
	out := &domain.TaskEnrichment{
		ActiveUsers: map[int64][]domain.UserRef{},
		UserColors:  map[int64]string{},
		FavoriteIDs: map[int64]bool{},
		WithUnits:   map[int64]bool{},
	}
	if len(taskIDs) == 0 {
		return out, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT u.task_id, us.id, us.fio, us.avatar_path
		  FROM units u
		  JOIN users us ON us.id = u.user_id
		 WHERE u.task_id = ANY($1) AND u.datetime_end IS NULL`, taskIDs)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var taskID int64
		var u domain.UserRef
		if err := rows.Scan(&taskID, &u.ID, &u.FIO, &u.AvatarPath); err != nil {
			rows.Close()
			return nil, err
		}
		out.ActiveUsers[taskID] = append(out.ActiveUsers[taskID], u)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	colorRows, err := r.pool.Query(ctx, `
		SELECT task_id, color FROM user_task_colors
		 WHERE user_id = $1 AND task_id = ANY($2)`, userID, taskIDs)
	if err != nil {
		return nil, err
	}
	for colorRows.Next() {
		var taskID int64
		var color string
		if err := colorRows.Scan(&taskID, &color); err != nil {
			colorRows.Close()
			return nil, err
		}
		out.UserColors[taskID] = color
	}
	colorRows.Close()
	if err := colorRows.Err(); err != nil {
		return nil, err
	}

	favRows, err := r.pool.Query(ctx, `
		SELECT task_id FROM favorites WHERE user_id = $1 AND task_id = ANY($2)`,
		userID, taskIDs)
	if err != nil {
		return nil, err
	}
	for favRows.Next() {
		var taskID int64
		if err := favRows.Scan(&taskID); err != nil {
			favRows.Close()
			return nil, err
		}
		out.FavoriteIDs[taskID] = true
	}
	favRows.Close()
	if err := favRows.Err(); err != nil {
		return nil, err
	}

	unitRows, err := r.pool.Query(ctx, `
		SELECT DISTINCT task_id FROM units WHERE task_id = ANY($1)`, taskIDs)
	if err != nil {
		return nil, err
	}
	for unitRows.Next() {
		var taskID int64
		if err := unitRows.Scan(&taskID); err != nil {
			unitRows.Close()
			return nil, err
		}
		out.WithUnits[taskID] = true
	}
	unitRows.Close()
	return out, unitRows.Err()
}

// ── Общие хелперы ────────────────────────────────────────────────

// updateFields — точечный UPDATE с детерминированным порядком колонок.
func updateFields(ctx context.Context, pool *pgxpool.Pool, table string,
	allowed map[string]bool, id int64, fields map[string]any) error {

	if len(fields) == 0 {
		return nil
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		if !allowed[k] {
			return fmt.Errorf("update %s: недопустимое поле %q", table, k)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	set := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)+1)
	for i, k := range keys {
		set = append(set, fmt.Sprintf("%q = $%d", k, i+1))
		args = append(args, fields[k])
	}
	args = append(args, id)
	_, err := pool.Exec(ctx,
		fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d", table, strings.Join(set, ", "), len(args)),
		args...)
	return err
}
