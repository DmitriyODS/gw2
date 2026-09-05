package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

const itemCols = `i.id, i.schedule_id, i.weekday, i.start_min, i.end_min, i.title,
	i.short, i.category_id, i.weeks, i.repeat_every, i.repeat_from, i.repeat_until,
	i.data, i.created_at, i.updated_at`

/* Номера недель лежат в smallint[]: недель в цикле максимум четыре, и int2 —
   их естественный размер. Домен работает с []int, поэтому границу типов
   переходим здесь, а не размазываем int16 по бизнес-логике. */

type itemRow struct {
	item  domain.Item
	weeks []int16
}

func (r *itemRow) targets() []any {
	i := &r.item
	return []any{&i.ID, &i.ScheduleID, &i.Weekday, &i.StartMin, &i.EndMin, &i.Title,
		&i.Short, &i.CategoryID, &r.weeks, &i.RepeatEvery, &i.RepeatFrom, &i.RepeatUntil,
		&i.Data, &i.CreatedAt, &i.UpdatedAt}
}

// build — доменное занятие из прочитанной строки.
func (r *itemRow) build() *domain.Item {
	out := r.item
	out.Weeks = make([]int, 0, len(r.weeks))
	for _, w := range r.weeks {
		out.Weeks = append(out.Weeks, int(w))
	}
	if out.Data == nil {
		out.Data = map[string]any{}
	}
	return &out
}

func weeksParam(weeks []int) []int16 {
	out := make([]int16, 0, len(weeks))
	for _, w := range weeks {
		out = append(out, int16(w))
	}
	return out
}

func (r *Repo) ListItems(ctx context.Context, scheduleID int64) ([]*domain.Item, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+itemCols+` FROM schedule_items i
		  WHERE i.schedule_id = $1
		  ORDER BY i.weekday, i.start_min, i.id`, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Item{}
	for rows.Next() {
		var row itemRow
		if err := rows.Scan(row.targets()...); err != nil {
			return nil, err
		}
		out = append(out, row.build())
	}
	return out, rows.Err()
}

func (r *Repo) GetItem(ctx context.Context, scheduleID, id int64) (*domain.Item, error) {
	var row itemRow
	err := r.pool.QueryRow(ctx,
		`SELECT `+itemCols+` FROM schedule_items i
		  WHERE i.id = $1 AND i.schedule_id = $2`, id, scheduleID).Scan(row.targets()...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row.build(), nil
}

func (r *Repo) CreateItem(ctx context.Context, i *domain.Item, searchText string) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO schedule_items (schedule_id, weekday, start_min, end_min, title, short,
		            category_id, weeks, repeat_every, repeat_from, repeat_until, data, search_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at`,
		i.ScheduleID, i.Weekday, i.StartMin, i.EndMin, i.Title, i.Short, i.CategoryID,
		weeksParam(i.Weeks), i.RepeatEvery, i.RepeatFrom, i.RepeatUntil, i.Data, searchText).
		Scan(&i.ID, &i.CreatedAt, &i.UpdatedAt)
}

func (r *Repo) UpdateItem(ctx context.Context, i *domain.Item, searchText string) error {
	err := r.pool.QueryRow(ctx, `
		UPDATE schedule_items
		   SET weekday = $3, start_min = $4, end_min = $5, title = $6, short = $7,
		       category_id = $8, weeks = $9, repeat_every = $10, repeat_from = $11,
		       repeat_until = $12, data = $13, search_text = $14, updated_at = now()
		 WHERE id = $1 AND schedule_id = $2
		RETURNING updated_at`,
		i.ID, i.ScheduleID, i.Weekday, i.StartMin, i.EndMin, i.Title, i.Short, i.CategoryID,
		weeksParam(i.Weeks), i.RepeatEvery, i.RepeatFrom, i.RepeatUntil, i.Data, searchText).
		Scan(&i.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrItemNotFound
	}
	return err
}

func (r *Repo) DeleteItem(ctx context.Context, scheduleID, id int64) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM schedule_items WHERE id = $1 AND schedule_id = $2`, id, scheduleID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrItemNotFound
	}
	return nil
}

// ReplaceItems — заменить занятия расписания целиком (импорт JSON): одной
// транзакцией, чтобы расписание не осталось наполовину пустым.
func (r *Repo) ReplaceItems(ctx context.Context, scheduleID int64, items []*domain.Item,
	searchText func(*domain.Item) string) error {

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM schedule_items WHERE schedule_id = $1`, scheduleID); err != nil {
		return err
	}
	for _, i := range items {
		i.ScheduleID = scheduleID
		if err := tx.QueryRow(ctx, `
			INSERT INTO schedule_items (schedule_id, weekday, start_min, end_min, title, short,
			            category_id, weeks, repeat_every, repeat_from, repeat_until, data, search_text)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			RETURNING id, created_at, updated_at`,
			scheduleID, i.Weekday, i.StartMin, i.EndMin, i.Title, i.Short, i.CategoryID,
			weeksParam(i.Weeks), i.RepeatEvery, i.RepeatFrom, i.RepeatUntil, i.Data, searchText(i)).
			Scan(&i.ID, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// NormalizeItemWeeks — подрезать номера недель под укороченный цикл.
// Занятие, у которого после подрезки не осталось ни одной недели, становится
// еженедельным (пустой набор): молча исчезнуть со шкалы оно не должно — это
// выглядело бы как потеря данных.
func (r *Repo) NormalizeItemWeeks(ctx context.Context, scheduleID int64, cycleWeeks int) (int, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE schedule_items
		   SET weeks = COALESCE((SELECT array_agg(w ORDER BY w)
		                           FROM unnest(weeks) AS w
		                          WHERE w BETWEEN 1 AND $2), '{}'::smallint[]),
		       updated_at = now()
		 WHERE schedule_id = $1
		   AND repeat_every IS NULL
		   AND EXISTS (SELECT 1 FROM unnest(weeks) AS w WHERE w > $2)`,
		scheduleID, cycleWeeks)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// accessJoin — занятия всех расписаний, доступных пользователю: своих и
// открытых ему адресно. Одним запросом — плитка и поиск идут сразу по всем.
const accessJoin = `
	  FROM schedule_items i
	  JOIN schedules s ON s.id = i.schedule_id
	  LEFT JOIN schedule_categories c ON c.id = i.category_id
	 WHERE (s.owner_id = $1 OR ` + sharedCondition + `)`

// ItemsForDay — занятия на день недели по всем доступным расписаниям. Какая
// сейчас неделя цикла, решает сервис: правило повтора живёт в домене.
func (r *Repo) ItemsForDay(ctx context.Context, userID int64, companyIDs []int64, weekday int) ([]*domain.DayItem, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+itemCols+`, `+scheduleCols+`, COALESCE(c.color, '')`+accessJoin+`
		   AND i.weekday = $3
		 ORDER BY i.start_min, i.end_min`, userID, companyIDs, weekday)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.DayItem{}
	for rows.Next() {
		var row itemRow
		var s domain.Schedule
		var color string
		targets := append(row.targets(), scheduleTargets(&s)...)
		targets = append(targets, &color)
		if err := rows.Scan(targets...); err != nil {
			return nil, err
		}
		sched := s
		out = append(out, &domain.DayItem{Item: row.build(), Schedule: &sched, Color: color})
	}
	return out, rows.Err()
}

// SearchItems — глобальный поиск Hola по занятиям доступных расписаний.
func (r *Repo) SearchItems(ctx context.Context, userID int64, companyIDs []int64,
	query string, limit int) ([]*domain.ItemScope, error) {

	if companyIDs == nil {
		companyIDs = []int64{}
	}
	rows, err := r.pool.Query(ctx, `
		SELECT i.schedule_id, s.name, i.id, i.title, i.weekday, i.start_min, i.end_min,
		       COALESCE(c.name, '')`+accessJoin+`
		   AND (i.title ILIKE '%' || $3 || '%' OR i.search_text ILIKE '%' || $3 || '%')
		 ORDER BY s.position, i.weekday, i.start_min
		 LIMIT $4`, userID, companyIDs, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.ItemScope{}
	for rows.Next() {
		var h domain.ItemScope
		if err := rows.Scan(&h.ScheduleID, &h.ScheduleName, &h.ItemID, &h.Title,
			&h.Weekday, &h.StartMin, &h.EndMin, &h.Category); err != nil {
			return nil, err
		}
		out = append(out, &h)
	}
	return out, rows.Err()
}
