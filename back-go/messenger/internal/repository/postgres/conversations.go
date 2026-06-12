package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

// convCols — колонки диалога (+ имя компании) для scanConversation.
const convCols = `c.id, c.user_a_id, c.user_b_id, c.company_id, co.name,
	c.is_dev_chat, c.is_pet_chat, c.created_at, c.last_message_at,
	c.hidden_for_a, c.hidden_for_b, c.pinned_at_a, c.pinned_at_b`

const convFrom = ` FROM conversations c LEFT JOIN companies co ON co.id = c.company_id `

func scanConversation(row pgx.Row) (*domain.Conversation, error) {
	var c domain.Conversation
	err := row.Scan(&c.ID, &c.UserAID, &c.UserBID, &c.CompanyID, &c.CompanyName,
		&c.IsDevChat, &c.IsPetChat, &c.CreatedAt, &c.LastMessageAt,
		&c.HiddenForA, &c.HiddenForB, &c.PinnedAtA, &c.PinnedAtB)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repo) GetConversation(ctx context.Context, id int64) (*domain.Conversation, error) {
	return scanConversation(r.q(ctx).QueryRow(ctx,
		`SELECT `+convCols+convFrom+`WHERE c.id = $1`, id))
}

func (r *Repo) GetPair(ctx context.Context, a, b int64) (*domain.Conversation, error) {
	return scanConversation(r.q(ctx).QueryRow(ctx,
		`SELECT `+convCols+convFrom+`WHERE c.user_a_id = $1 AND c.user_b_id = $2`, a, b))
}

func (r *Repo) CreatePair(ctx context.Context, a, b, companyID int64) (*domain.Conversation, error) {
	row := r.q(ctx).QueryRow(ctx, `
		INSERT INTO conversations (user_a_id, user_b_id, company_id, is_dev_chat, is_pet_chat,
			created_at, hidden_for_a, hidden_for_b)
		VALUES ($1, $2, $3, FALSE, FALSE, now(), FALSE, FALSE)
		RETURNING id`, a, b, companyID)
	var id int64
	if err := row.Scan(&id); err != nil {
		// Гонка по уникальной паре — диалог уже создан параллельно.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return r.GetPair(ctx, a, b)
		}
		return nil, err
	}
	return r.GetConversation(ctx, id)
}

func soloFlag(pet bool) string {
	if pet {
		return "c.is_pet_chat"
	}
	return "c.is_dev_chat"
}

func (r *Repo) GetSolo(ctx context.Context, userID int64, pet bool) (*domain.Conversation, error) {
	return scanConversation(r.q(ctx).QueryRow(ctx,
		`SELECT `+convCols+convFrom+`WHERE `+soloFlag(pet)+` = TRUE AND c.user_a_id = $1`, userID))
}

func (r *Repo) CreateSolo(ctx context.Context, userID, companyID int64, pet bool) (*domain.Conversation, error) {
	row := r.q(ctx).QueryRow(ctx, `
		INSERT INTO conversations (user_a_id, user_b_id, company_id, is_dev_chat, is_pet_chat,
			created_at, hidden_for_a, hidden_for_b)
		VALUES ($1, NULL, $2, $3, $4, now(), FALSE, FALSE)
		RETURNING id`, userID, companyID, !pet, pet)
	var id int64
	if err := row.Scan(&id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return r.GetSolo(ctx, userID, pet)
		}
		return nil, err
	}
	return r.GetConversation(ctx, id)
}

func (r *Repo) listConversations(ctx context.Context, sql string, args ...any) ([]*domain.Conversation, error) {
	rows, err := r.q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Conversation
	for rows.Next() {
		c, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repo) ListPairConversations(ctx context.Context, userID int64) ([]*domain.Conversation, error) {
	return r.listConversations(ctx, `
		SELECT `+convCols+convFrom+`
		WHERE c.is_dev_chat = FALSE AND c.is_pet_chat = FALSE
		  AND ((c.user_a_id = $1 AND c.hidden_for_a = FALSE)
		    OR (c.user_b_id = $1 AND c.hidden_for_b = FALSE))
		ORDER BY c.last_message_at DESC NULLS LAST, c.created_at DESC`, userID)
}

func (r *Repo) ListDevChats(ctx context.Context) ([]*domain.Conversation, error) {
	return r.listConversations(ctx, `
		SELECT `+convCols+convFrom+`
		WHERE c.is_dev_chat = TRUE
		ORDER BY c.last_message_at DESC NULLS LAST, c.created_at DESC`)
}

// HideConversation — скрыть диалог и все его сообщения на стороне side;
// true — скрыт обеими сторонами (вызывающий удаляет физически).
func (r *Repo) HideConversation(ctx context.Context, id int64, side string) (bool, error) {
	var both bool
	err := r.RunInTx(ctx, func(ctx context.Context) error {
		row := r.q(ctx).QueryRow(ctx,
			`UPDATE conversations SET `+hiddenCol(side)+` = TRUE WHERE id = $1
			 RETURNING hidden_for_a AND hidden_for_b`, id)
		if err := row.Scan(&both); err != nil {
			return err
		}
		// Скрываем и сообщения: при «возврате» диалога собеседник не должен
		// видеть переписку, которую другая сторона стёрла.
		_, err := r.q(ctx).Exec(ctx,
			`UPDATE messages SET `+hiddenCol(side)+` = TRUE WHERE conversation_id = $1`, id)
		return err
	})
	return both, err
}

// DeleteConversation — сообщения и вложения каскадно уходят по FK.
func (r *Repo) DeleteConversation(ctx context.Context, id int64) error {
	_, err := r.q(ctx).Exec(ctx, `DELETE FROM conversations WHERE id = $1`, id)
	return err
}

func (r *Repo) SetConversationPin(ctx context.Context, id int64, side string, pinned bool) error {
	value := "NULL"
	if pinned {
		value = "now()"
	}
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE conversations SET `+pinCol(side)+` = `+value+` WHERE id = $1`, id)
	return err
}
