package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/reminder/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.ReminderRepository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// columns — общий набор полей выборки (порядок совпадает со scanReminder).
const columns = `id, owner_id, title, note, remind_at, timezone,
	repeat_kind, repeat_interval, repeat_days, repeat_until,
	link_kind, link_parent_id, link_record_id, link_title, link_lead_min,
	active, last_fired_at, created_at, updated_at`

// scanner — общий интерфейс pgx.Row/pgx.Rows для одного разбора строки.
type scanner interface{ Scan(dest ...any) error }

func scanReminder(row scanner) (*domain.Reminder, error) {
	var r domain.Reminder
	err := row.Scan(&r.ID, &r.OwnerID, &r.Title, &r.Note, &r.RemindAt, &r.Timezone,
		&r.Repeat.Kind, &r.Repeat.Interval, &r.Repeat.Days, &r.Repeat.Until,
		&r.Link.Kind, &r.Link.ParentID, &r.Link.RecordID, &r.Link.Title, &r.Link.LeadMinutes,
		&r.Active, &r.LastFiredAt, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if r.Repeat.Days == nil {
		r.Repeat.Days = []int{}
	}
	return &r, nil
}

func collect(rows pgx.Rows) ([]*domain.Reminder, error) {
	defer rows.Close()
	out := []*domain.Reminder{}
	for rows.Next() {
		r, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (r *Repo) List(ctx context.Context, ownerID int64, scope domain.ListScope) ([]*domain.Reminder, error) {
	q := `SELECT ` + columns + ` FROM reminders WHERE owner_id = $1`
	switch scope {
	case domain.ScopeDone:
		q += ` AND NOT active ORDER BY remind_at DESC, id DESC`
	case domain.ScopeAll:
		q += ` ORDER BY active DESC, remind_at, id`
	default:
		q += ` AND active ORDER BY remind_at, id`
	}
	rows, err := r.pool.Query(ctx, q, ownerID)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}

func (r *Repo) Upcoming(ctx context.Context, ownerID int64, until time.Time, limit int) ([]*domain.Reminder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+columns+` FROM reminders
		 WHERE owner_id = $1 AND active AND remind_at <= $2
		 ORDER BY remind_at, id LIMIT $3`, ownerID, until, limit)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}

func (r *Repo) ByLink(ctx context.Context, ownerID int64, kind string, recordID int64) ([]*domain.Reminder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+columns+` FROM reminders
		 WHERE owner_id = $1 AND link_kind = $2 AND link_record_id = $3
		 ORDER BY remind_at, id`, ownerID, kind, recordID)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}

func (r *Repo) Get(ctx context.Context, id int64) (*domain.Reminder, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+columns+` FROM reminders WHERE id = $1`, id)
	rem, err := scanReminder(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rem, nil
}

func (r *Repo) Create(ctx context.Context, rem *domain.Reminder) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO reminders (owner_id, title, note, remind_at, timezone,
			repeat_kind, repeat_interval, repeat_days, repeat_until,
			link_kind, link_parent_id, link_record_id, link_title, link_lead_min, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at`,
		rem.OwnerID, rem.Title, rem.Note, rem.RemindAt, rem.Timezone,
		rem.Repeat.Kind, rem.Repeat.Interval, rem.Repeat.Days, rem.Repeat.Until,
		rem.Link.Kind, rem.Link.ParentID, rem.Link.RecordID, rem.Link.Title, rem.Link.LeadMinutes,
		rem.Active).
		Scan(&rem.ID, &rem.CreatedAt, &rem.UpdatedAt)
}

func (r *Repo) Update(ctx context.Context, rem *domain.Reminder) error {
	return r.pool.QueryRow(ctx, `
		UPDATE reminders SET title = $2, note = $3, remind_at = $4, timezone = $5,
			repeat_kind = $6, repeat_interval = $7, repeat_days = $8, repeat_until = $9,
			link_kind = $10, link_parent_id = $11, link_record_id = $12, link_title = $13,
			link_lead_min = $14, active = $15, last_fired_at = $16, updated_at = now()
		 WHERE id = $1 RETURNING updated_at`,
		rem.ID, rem.Title, rem.Note, rem.RemindAt, rem.Timezone,
		rem.Repeat.Kind, rem.Repeat.Interval, rem.Repeat.Days, rem.Repeat.Until,
		rem.Link.Kind, rem.Link.ParentID, rem.Link.RecordID, rem.Link.Title, rem.Link.LeadMinutes,
		rem.Active, rem.LastFiredAt).
		Scan(&rem.UpdatedAt)
}

func (r *Repo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM reminders WHERE id = $1`, id)
	return err
}

// ClaimDue — атомарный забор наступивших сроков: строки помечаются отработавшими
// (active = FALSE, fired_seq += 1) тем же запросом, которым выбираются, под
// SKIP LOCKED. Отсюда инвариант «одно срабатывание — одна доставка» даже при
// нескольких инстансах сервиса; следующий срок повтора проставит сервис своим
// Update (правило повтора — доменное знание, в SQL его не тащим).
func (r *Repo) ClaimDue(ctx context.Context, now time.Time, limit int) ([]*domain.Reminder, error) {
	rows, err := r.pool.Query(ctx, `
		UPDATE reminders SET active = FALSE, fired_seq = fired_seq + 1, updated_at = now()
		 WHERE id IN (
			SELECT id FROM reminders
			 WHERE active AND remind_at <= $1
			 ORDER BY remind_at
			 LIMIT $2
			 FOR UPDATE SKIP LOCKED
		 )
		RETURNING `+columns, now, limit)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}
