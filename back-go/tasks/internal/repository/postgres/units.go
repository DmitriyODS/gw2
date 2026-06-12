package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

func nilTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

const unitColumns = `
	u.id, u.name, u.user_id, u.unit_type_id, u.task_id, u.company_id,
	u.is_edited, u.datetime_start, u.datetime_end, u.created_at,
	us.id, us.fio, us.avatar_path,
	ut.id, ut.name`

const unitFrom = `
	FROM units u
	JOIN users us ON us.id = u.user_id
	JOIN unit_types ut ON ut.id = u.unit_type_id`

func scanUnit(row pgx.Row) (*domain.Unit, error) {
	var (
		u       domain.Unit
		usID    int64
		usFIO   string
		usAv    *string
		utID    int64
		utName  string
	)
	err := row.Scan(
		&u.ID, &u.Name, &u.UserID, &u.UnitTypeID, &u.TaskID, &u.CompanyID,
		&u.IsEdited, &u.DatetimeStart, &u.DatetimeEnd, &u.CreatedAt,
		&usID, &usFIO, &usAv,
		&utID, &utName,
	)
	if err != nil {
		return nil, err
	}
	u.User = &domain.UserRef{ID: usID, FIO: usFIO, AvatarPath: usAv}
	u.UnitType = &domain.UnitTypeRef{ID: utID, Name: utName}
	return &u, nil
}

func (r *Repo) GetUnit(ctx context.Context, id int64) (*domain.Unit, error) {
	u, err := scanUnit(r.pool.QueryRow(ctx,
		"SELECT"+unitColumns+unitFrom+" WHERE u.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *Repo) UnitsByTask(ctx context.Context, taskID int64) ([]*domain.Unit, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT"+unitColumns+unitFrom+" WHERE u.task_id = $1 ORDER BY u.datetime_start DESC",
		taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*domain.Unit{}
	for rows.Next() {
		u, err := scanUnit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repo) ActiveUnitForUser(ctx context.Context, userID int64) (*domain.Unit, error) {
	u, err := scanUnit(r.pool.QueryRow(ctx,
		"SELECT"+unitColumns+unitFrom+" WHERE u.user_id = $1 AND u.datetime_end IS NULL", userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *Repo) CreateUnit(ctx context.Context, u *domain.Unit) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO units (name, user_id, unit_type_id, task_id, company_id,
		                   is_edited, datetime_start, created_at)
		VALUES ($1, $2, $3, $4, $5, FALSE, now(), now())
		RETURNING id, datetime_start, created_at`,
		u.Name, u.UserID, u.UnitTypeID, u.TaskID, u.CompanyID,
	).Scan(&u.ID, &u.DatetimeStart, &u.CreatedAt)
}

var allowedUnitFields = map[string]bool{
	"name": true, "unit_type_id": true, "datetime_start": true, "datetime_end": true,
	"is_edited": true,
}

// UpdateUnitFields — любое редактирование (даже пустой PATCH) помечает юнит
// is_edited=TRUE, как unit_repo.update во Flask.
func (r *Repo) UpdateUnitFields(ctx context.Context, id int64, fields map[string]any) error {
	merged := map[string]any{"is_edited": true}
	for k, v := range fields {
		merged[k] = v
	}
	return updateFields(ctx, r.pool, "units", allowedUnitFields, id, merged)
}

func (r *Repo) StopUnit(ctx context.Context, id int64) (time.Time, error) {
	var end time.Time
	err := r.pool.QueryRow(ctx, `
		UPDATE units SET datetime_end = now() WHERE id = $1
		RETURNING datetime_end`, id).Scan(&end)
	return end, err
}

func (r *Repo) DeleteUnit(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM units WHERE id = $1`, id)
	return err
}
