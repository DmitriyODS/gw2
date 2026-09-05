package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

/* Структура расписания: справочник категорий и набор дополнительных полей.
   И то и другое принадлежит КОНКРЕТНОМУ расписанию — общего каталога на
   человека нет. */

const categoryCols = `id, schedule_id, name, color, position, created_at`

func scanCategories(rows pgx.Rows) ([]*domain.Category, error) {
	defer rows.Close()
	out := []*domain.Category{}
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.ScheduleID, &c.Name, &c.Color, &c.Position, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *Repo) ListCategories(ctx context.Context, scheduleID int64) ([]*domain.Category, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+categoryCols+` FROM schedule_categories
		  WHERE schedule_id = $1 ORDER BY position, id`, scheduleID)
	if err != nil {
		return nil, err
	}
	return scanCategories(rows)
}

// CategoriesBySchedules — категории сразу для списка расписаний: список слева
// показывает цвета, и досчитывать их по одному запросу на расписание нельзя.
func (r *Repo) CategoriesBySchedules(ctx context.Context, ids []int64) (map[int64][]*domain.Category, error) {
	out := map[int64][]*domain.Category{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+categoryCols+` FROM schedule_categories
		  WHERE schedule_id = ANY($1) ORDER BY schedule_id, position, id`, ids)
	if err != nil {
		return nil, err
	}
	list, err := scanCategories(rows)
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		out[c.ScheduleID] = append(out[c.ScheduleID], c)
	}
	return out, nil
}

func (r *Repo) NextCategoryPosition(ctx context.Context, scheduleID int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(max(position), 0) + 1 FROM schedule_categories WHERE schedule_id = $1`,
		scheduleID).Scan(&pos)
	return pos, err
}

func (r *Repo) CreateCategory(ctx context.Context, c *domain.Category) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO schedule_categories (schedule_id, name, color, position)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		c.ScheduleID, c.Name, c.Color, c.Position).Scan(&c.ID, &c.CreatedAt)
}

// UpdateCategory — правка своей категории: schedule_id в условии, поэтому чужой
// id просто не найдётся.
func (r *Repo) UpdateCategory(ctx context.Context, c *domain.Category) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE schedule_categories SET name = $3, color = $4, position = $5
		 WHERE id = $1 AND schedule_id = $2`,
		c.ID, c.ScheduleID, c.Name, c.Color, c.Position)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

// DeleteCategory — занятия категорию теряют (ON DELETE SET NULL), но остаются:
// пропажа занятия из-за удалённой категории была бы потерей данных.
func (r *Repo) DeleteCategory(ctx context.Context, scheduleID, id int64) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM schedule_categories WHERE id = $1 AND schedule_id = $2`, id, scheduleID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

const fieldCols = `id, schedule_id, label, type, config, position, col_span,
	row_span, show_on_block, show_in_card, created_at`

func scanFields(rows pgx.Rows) ([]*domain.Field, error) {
	defer rows.Close()
	out := []*domain.Field{}
	for rows.Next() {
		var f domain.Field
		if err := rows.Scan(&f.ID, &f.ScheduleID, &f.Label, &f.Type, &f.Config, &f.Position,
			&f.ColSpan, &f.RowSpan, &f.ShowOnBlock, &f.ShowInCard, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &f)
	}
	return out, rows.Err()
}

func (r *Repo) ListFields(ctx context.Context, scheduleID int64) ([]*domain.Field, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+fieldCols+` FROM schedule_fields
		  WHERE schedule_id = $1 ORDER BY position, id`, scheduleID)
	if err != nil {
		return nil, err
	}
	return scanFields(rows)
}

func (r *Repo) FieldsBySchedules(ctx context.Context, ids []int64) (map[int64][]*domain.Field, error) {
	out := map[int64][]*domain.Field{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+fieldCols+` FROM schedule_fields
		  WHERE schedule_id = ANY($1) ORDER BY schedule_id, position, id`, ids)
	if err != nil {
		return nil, err
	}
	list, err := scanFields(rows)
	if err != nil {
		return nil, err
	}
	for _, f := range list {
		out[f.ScheduleID] = append(out[f.ScheduleID], f)
	}
	return out, nil
}

// ReplaceFields — полная замена набора полей одной транзакцией: поле с
// известным id обновляется, новое вставляется, пропавшее удаляется. Id полей
// переживают правку структуры, потому что по ним лежат ЗНАЧЕНИЯ в занятиях:
// пересоздание набора обнулило бы заполненные карточки.
func (r *Repo) ReplaceFields(ctx context.Context, scheduleID int64, fields []*domain.Field) ([]int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	keep := make([]int64, 0, len(fields))
	for pos, f := range fields {
		f.ScheduleID = scheduleID
		f.Position = pos + 1
		if f.ID > 0 {
			tag, err := tx.Exec(ctx, `
				UPDATE schedule_fields
				   SET label = $3, type = $4, config = $5, position = $6, col_span = $7,
				       row_span = $8, show_on_block = $9, show_in_card = $10
				 WHERE id = $1 AND schedule_id = $2`,
				f.ID, scheduleID, f.Label, f.Type, f.Config, f.Position, f.ColSpan,
				f.RowSpan, f.ShowOnBlock, f.ShowInCard)
			if err != nil {
				return nil, err
			}
			if tag.RowsAffected() > 0 {
				keep = append(keep, f.ID)
				continue
			}
			f.ID = 0 // чужой или уже удалённый id — заводим поле заново
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO schedule_fields (schedule_id, label, type, config, position,
			                             col_span, row_span, show_on_block, show_in_card)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id, created_at`,
			scheduleID, f.Label, f.Type, f.Config, f.Position, f.ColSpan, f.RowSpan,
			f.ShowOnBlock, f.ShowInCard).Scan(&f.ID, &f.CreatedAt); err != nil {
			return nil, err
		}
		keep = append(keep, f.ID)
	}

	rows, err := tx.Query(ctx,
		`DELETE FROM schedule_fields
		  WHERE schedule_id = $1 AND NOT (id = ANY($2))
		RETURNING id`, scheduleID, keep)
	if err != nil {
		return nil, err
	}
	removed := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		removed = append(removed, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return removed, tx.Commit(ctx)
}

// ReplaceCategories — заменить справочник целиком (импорт расписания). Одной
// транзакцией: расписание не должно остаться с половиной категорий, на которые
// уже не ссылается ни одно занятие.
func (r *Repo) ReplaceCategories(ctx context.Context, scheduleID int64, cats []*domain.Category) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM schedule_categories WHERE schedule_id = $1`, scheduleID); err != nil {
		return err
	}
	for pos, c := range cats {
		c.ScheduleID = scheduleID
		c.Position = pos + 1
		if err := tx.QueryRow(ctx, `
			INSERT INTO schedule_categories (schedule_id, name, color, position)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at`,
			scheduleID, c.Name, c.Color, c.Position).Scan(&c.ID, &c.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
