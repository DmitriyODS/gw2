// Package postgres — pgx-реализация портов персистентности.
//
// Сервис работает с общей PostgreSQL платформы. Владелец схемы — goose
// (back-go/migrate): здесь только запросы, никакого DDL.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
)

type CallRepository struct {
	pool *pgxpool.Pool
}

var _ domain.CallRepository = (*CallRepository)(nil)

func NewCallRepository(pool *pgxpool.Pool) *CallRepository {
	return &CallRepository{pool: pool}
}

const callColumns = `id, initiator_id, company_id, kind, status, media,
	started_at, ended_at, conversation_id, room_name, share_code`

func scanCall(row pgx.Row) (*domain.Call, error) {
	var c domain.Call
	var roomName, shareCode *string
	err := row.Scan(&c.ID, &c.InitiatorID, &c.CompanyID, &c.Kind, &c.Status,
		&c.Media, &c.StartedAt, &c.EndedAt, &c.ConversationID, &roomName, &shareCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if roomName != nil {
		c.RoomName = *roomName
	}
	if shareCode != nil {
		c.ShareCode = *shareCode
	}
	return &c, nil
}

func (r *CallRepository) CreateCall(ctx context.Context, call *domain.Call,
	participants []*domain.Participant) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck — no-op после Commit

	err = tx.QueryRow(ctx, `
		INSERT INTO calls (initiator_id, company_id, kind, status, media,
		                   started_at, conversation_id, share_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		call.InitiatorID, call.CompanyID, call.Kind, call.Status, call.Media,
		call.StartedAt, call.ConversationID, call.ShareCode,
	).Scan(&call.ID)
	if err != nil {
		return err
	}

	call.RoomName = domain.RoomNameFor(call.ID)
	if _, err = tx.Exec(ctx,
		`UPDATE calls SET room_name = $1 WHERE id = $2`,
		call.RoomName, call.ID); err != nil {
		return err
	}

	for _, p := range participants {
		p.CallID = call.ID
		if err = insertParticipant(ctx, tx, p); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func insertParticipant(ctx context.Context, tx pgx.Tx, p *domain.Participant) error {
	return tx.QueryRow(ctx, `
		INSERT INTO call_participants (call_id, user_id, role, invited_at,
		                               joined_at, left_at, declined)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		p.CallID, p.UserID, p.Role, p.InvitedAt, p.JoinedAt, p.LeftAt, p.Declined,
	).Scan(&p.ID)
}

func (r *CallRepository) GetCall(ctx context.Context, id int64) (*domain.Call, error) {
	row := r.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM calls WHERE id = $1`, callColumns), id)
	return scanCall(row)
}

func (r *CallRepository) GetCallByShareCode(ctx context.Context, code string) (*domain.Call, error) {
	row := r.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM calls WHERE share_code = $1`, callColumns), code)
	return scanCall(row)
}

func (r *CallRepository) UpdateCall(ctx context.Context, call *domain.Call) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE calls SET kind = $1, status = $2, ended_at = $3 WHERE id = $4`,
		call.Kind, call.Status, call.EndedAt, call.ID)
	return err
}

// DeleteCall — участники уходят каскадом (FK ondelete=CASCADE),
// messages.call_id обнуляется (SET NULL).
func (r *CallRepository) DeleteCall(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM calls WHERE id = $1`, id)
	return err
}

func (r *CallRepository) GetParticipant(ctx context.Context, callID, userID int64) (*domain.Participant, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT cp.id, cp.call_id, cp.user_id, cp.role, cp.invited_at,
		       cp.joined_at, cp.left_at, cp.declined, u.fio, u.avatar_path
		FROM call_participants cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.call_id = $1 AND cp.user_id = $2`, callID, userID)
	p, err := scanParticipant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func scanParticipant(row pgx.Row) (*domain.Participant, error) {
	var p domain.Participant
	if err := row.Scan(&p.ID, &p.CallID, &p.UserID, &p.Role, &p.InvitedAt,
		&p.JoinedAt, &p.LeftAt, &p.Declined, &p.FIO, &p.AvatarPath); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *CallRepository) ListParticipants(ctx context.Context, callID int64) ([]*domain.Participant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT cp.id, cp.call_id, cp.user_id, cp.role, cp.invited_at,
		       cp.joined_at, cp.left_at, cp.declined, u.fio, u.avatar_path
		FROM call_participants cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.call_id = $1
		ORDER BY cp.id`, callID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Participant
	for rows.Next() {
		p, err := scanParticipant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *CallRepository) CreateParticipant(ctx context.Context, p *domain.Participant) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO call_participants (call_id, user_id, role, invited_at,
		                               joined_at, left_at, declined)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		p.CallID, p.UserID, p.Role, p.InvitedAt, p.JoinedAt, p.LeftAt, p.Declined,
	).Scan(&p.ID)
}

func (r *CallRepository) UpdateParticipant(ctx context.Context, p *domain.Participant) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE call_participants
		SET invited_at = $1, joined_at = $2, left_at = $3, declined = $4
		WHERE id = $5`,
		p.InvitedAt, p.JoinedAt, p.LeftAt, p.Declined, p.ID)
	return err
}

func (r *CallRepository) CloseOpenParticipants(ctx context.Context, callID int64, leftAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE call_participants SET left_at = $1
		WHERE call_id = $2 AND left_at IS NULL`, leftAt, callID)
	return err
}

func (r *CallRepository) ListUnfinishedCalls(ctx context.Context) ([]*domain.Call, error) {
	rows, err := r.pool.Query(ctx, fmt.Sprintf(
		`SELECT %s FROM calls WHERE status IN ('ringing', 'active')`, callColumns))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectCalls(rows)
}

func (r *CallRepository) ListHistoryForUser(ctx context.Context, userID int64, limit int) ([]*domain.Call, error) {
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT DISTINCT %s FROM calls c
		JOIN call_participants cp ON cp.call_id = c.id
		WHERE cp.user_id = $1
		ORDER BY c.started_at DESC
		LIMIT $2`, prefixed("c", callColumns)), userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectCalls(rows)
}

func collectCalls(rows pgx.Rows) ([]*domain.Call, error) {
	var out []*domain.Call
	for rows.Next() {
		c, err := scanCall(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
