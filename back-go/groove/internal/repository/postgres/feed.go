// Package postgres — репозитории groovesvc поверх общей PostgreSQL платформы
// (схему всех таблиц ведёт Alembic на стороне Flask; raw SQL через pgx).
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

type FeedRepo struct {
	pool *pgxpool.Pool
}

var _ domain.FeedRepo = (*FeedRepo)(nil)

func NewFeedRepo(pool *pgxpool.Pool) *FeedRepo {
	return &FeedRepo{pool: pool}
}

func scanPayload(raw []byte) map[string]any {
	payload := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &payload)
	}
	return payload
}

// scanUserRef — поля id/fio/avatar_path из LEFT JOIN users (всё nullable).
func userRef(id *int64, fio, avatar *string) *domain.UserRef {
	if id == nil {
		return nil
	}
	ref := &domain.UserRef{ID: *id, AvatarPath: avatar}
	if fio != nil {
		ref.FIO = *fio
	}
	return ref
}

func (r *FeedRepo) CreateEvent(ctx context.Context, companyID int64, userID *int64,
	kind string, payload map[string]any) (*domain.FeedEvent, error) {

	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	event := &domain.FeedEvent{
		CompanyID: companyID, UserID: userID, Kind: kind, Payload: payload,
	}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO feed_events (company_id, user_id, kind, payload, created_at)
		VALUES ($1, $2, $3, $4, now())
		RETURNING id, created_at`,
		companyID, userID, kind, raw,
	).Scan(&event.ID, &event.CreatedAt)
	if err != nil {
		return nil, err
	}
	if userID != nil {
		var fio *string
		var avatar *string
		err = r.pool.QueryRow(ctx,
			`SELECT fio, avatar_path FROM users WHERE id = $1`, *userID,
		).Scan(&fio, &avatar)
		if err == nil {
			event.User = userRef(userID, fio, avatar)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	return event, nil
}

const eventCols = `e.id, e.company_id, e.user_id, e.kind, e.payload, e.created_at,
	u.id, u.fio, u.avatar_path`

func scanEvent(row pgx.Row) (*domain.FeedEvent, error) {
	var e domain.FeedEvent
	var raw []byte
	var uid *int64
	var fio, avatar *string
	err := row.Scan(&e.ID, &e.CompanyID, &e.UserID, &e.Kind, &raw, &e.CreatedAt,
		&uid, &fio, &avatar)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	e.Payload = scanPayload(raw)
	e.User = userRef(uid, fio, avatar)
	return &e, nil
}

func (r *FeedRepo) GetEvent(ctx context.Context, id int64) (*domain.FeedEvent, error) {
	return scanEvent(r.pool.QueryRow(ctx, `
		SELECT `+eventCols+`
		FROM feed_events e LEFT JOIN users u ON u.id = e.user_id
		WHERE e.id = $1`, id))
}

func (r *FeedRepo) ListEvents(ctx context.Context, companyID, beforeID int64,
	limit int) ([]*domain.FeedEvent, error) {

	q := `SELECT ` + eventCols + `
		FROM feed_events e LEFT JOIN users u ON u.id = e.user_id
		WHERE e.company_id = $1`
	args := []any{companyID}
	if beforeID > 0 {
		q += ` AND e.id < $2 ORDER BY e.id DESC LIMIT $3`
		args = append(args, beforeID, limit)
	} else {
		q += ` ORDER BY e.id DESC LIMIT $2`
		args = append(args, limit)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.FeedEvent
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ───────────────────────────── реакции ─────────────────────────────

func (r *FeedRepo) ToggleReaction(ctx context.Context, eventID, userID int64,
	emoji string) (bool, error) {

	tag, err := r.pool.Exec(ctx, `
		DELETE FROM feed_reactions
		WHERE event_id = $1 AND user_id = $2 AND emoji = $3`,
		eventID, userID, emoji)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() > 0 {
		return false, nil
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO feed_reactions (event_id, user_id, emoji, created_at)
		VALUES ($1, $2, $3, now())`,
		eventID, userID, emoji)
	return true, err
}

func (r *FeedRepo) ReactionCounts(ctx context.Context, eventIDs []int64) (map[int64]map[string]int, error) {
	result := map[int64]map[string]int{}
	if len(eventIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT event_id, emoji, count(id) FROM feed_reactions
		WHERE event_id = ANY($1) GROUP BY event_id, emoji`, eventIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var eventID int64
		var emoji string
		var count int
		if err := rows.Scan(&eventID, &emoji, &count); err != nil {
			return nil, err
		}
		if result[eventID] == nil {
			result[eventID] = map[string]int{}
		}
		result[eventID][emoji] = count
	}
	return result, rows.Err()
}

func (r *FeedRepo) MyReactions(ctx context.Context, eventIDs []int64, userID int64) (map[int64][]string, error) {
	result := map[int64][]string{}
	if len(eventIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT event_id, emoji FROM feed_reactions
		WHERE event_id = ANY($1) AND user_id = $2`, eventIDs, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var eventID int64
		var emoji string
		if err := rows.Scan(&eventID, &emoji); err != nil {
			return nil, err
		}
		result[eventID] = append(result[eventID], emoji)
	}
	return result, rows.Err()
}

func (r *FeedRepo) ReactionCountFor(ctx context.Context, eventID int64, emoji string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(id) FROM feed_reactions WHERE event_id = $1 AND emoji = $2`,
		eventID, emoji).Scan(&count)
	return count, err
}

// ─────────────────────────── комментарии ───────────────────────────

func (r *FeedRepo) CommentCounts(ctx context.Context, eventIDs []int64) (map[int64]int, error) {
	result := map[int64]int{}
	if len(eventIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT event_id, count(id) FROM feed_comments
		WHERE event_id = ANY($1) GROUP BY event_id`, eventIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var eventID int64
		var count int
		if err := rows.Scan(&eventID, &count); err != nil {
			return nil, err
		}
		result[eventID] = count
	}
	return result, rows.Err()
}

const commentCols = `c.id, c.event_id, c.author_id, c.is_bot, c.reply_to_id,
	c.text, c.created_at, u.id, u.fio, u.avatar_path`

func scanComment(row pgx.Row) (*domain.FeedComment, error) {
	var c domain.FeedComment
	var uid *int64
	var fio, avatar *string
	err := row.Scan(&c.ID, &c.EventID, &c.AuthorID, &c.IsBot, &c.ReplyToID,
		&c.Text, &c.CreatedAt, &uid, &fio, &avatar)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	c.Author = userRef(uid, fio, avatar)
	return &c, nil
}

func (r *FeedRepo) ListComments(ctx context.Context, eventID int64) ([]*domain.FeedComment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+commentCols+`
		FROM feed_comments c LEFT JOIN users u ON u.id = c.author_id
		WHERE c.event_id = $1 ORDER BY c.created_at ASC`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.FeedComment
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *FeedRepo) CreateComment(ctx context.Context, eventID int64, authorID *int64,
	text string, replyToID *int64, isBot bool) (*domain.FeedComment, error) {

	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO feed_comments (event_id, author_id, is_bot, reply_to_id, text, created_at)
		VALUES ($1, $2, $3, $4, $5, now())
		RETURNING id`,
		eventID, authorID, isBot, replyToID, text).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetComment(ctx, id)
}

func (r *FeedRepo) GetComment(ctx context.Context, id int64) (*domain.FeedComment, error) {
	return scanComment(r.pool.QueryRow(ctx, `
		SELECT `+commentCols+`
		FROM feed_comments c LEFT JOIN users u ON u.id = c.author_id
		WHERE c.id = $1`, id))
}

func (r *FeedRepo) DeleteComment(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM feed_comments WHERE id = $1`, id)
	return err
}

// ─────────────────────── wrapped «Моя неделя» ──────────────────────

func (r *FeedRepo) CountUserEvents(ctx context.Context, companyID, userID int64,
	kind string, since time.Time) (int, error) {

	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(id) FROM feed_events
		WHERE company_id = $1 AND user_id = $2 AND kind = $3 AND created_at >= $4`,
		companyID, userID, kind, since).Scan(&count)
	return count, err
}

func (r *FeedRepo) ReactionsReceived(ctx context.Context, userID int64, since time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(fr.id)
		FROM feed_reactions fr JOIN feed_events fe ON fe.id = fr.event_id
		WHERE fe.user_id = $1 AND fr.user_id != $1 AND fr.created_at >= $2`,
		userID, since).Scan(&count)
	return count, err
}

func (r *FeedRepo) KudosReceived(ctx context.Context, companyID, userID int64, since time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(id) FROM feed_events
		WHERE company_id = $1 AND kind = 'kudos' AND created_at >= $2
		  AND (payload->>'to_user_id')::bigint = $3`,
		companyID, since, userID).Scan(&count)
	return count, err
}

// ─────────────────────────── live-блок ─────────────────────────────

func (r *FeedRepo) ListActiveUnits(ctx context.Context, companyID int64) ([]*domain.ActiveUnit, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT un.id, un.name, un.task_id, t.name, un.datetime_start,
		       u.id, u.fio, u.avatar_path
		FROM units un
		JOIN users u ON u.id = un.user_id
		LEFT JOIN tasks t ON t.id = un.task_id
		WHERE un.company_id = $1 AND un.datetime_end IS NULL AND u.is_active
		ORDER BY un.datetime_start ASC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.ActiveUnit
	for rows.Next() {
		var a domain.ActiveUnit
		var uid int64
		var fio string
		var avatar *string
		if err := rows.Scan(&a.ID, &a.Name, &a.TaskID, &a.TaskName, &a.StartedAt,
			&uid, &fio, &avatar); err != nil {
			return nil, err
		}
		a.User = &domain.UserRef{ID: uid, FIO: fio, AvatarPath: avatar}
		out = append(out, &a)
	}
	return out, rows.Err()
}
