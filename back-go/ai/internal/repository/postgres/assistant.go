package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
)

// AssistantRepo — персистентность диалога ИИ-ассистента
// (ai_assistant_conversations/ai_assistant_messages, миграция 00021).
type AssistantRepo struct {
	pool *pgxpool.Pool
}

var _ domain.AssistantRepository = (*AssistantRepo)(nil)

func NewAssistantRepo(pool *pgxpool.Pool) *AssistantRepo {
	return &AssistantRepo{pool: pool}
}

func (r *AssistantRepo) GetOrCreateConversation(ctx context.Context, userID, companyID int64) (*domain.AssistantConversation, error) {
	var c domain.AssistantConversation
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, company_id, created_at
		  FROM ai_assistant_conversations
		 WHERE user_id = $1 AND company_id = $2`, userID, companyID).
		Scan(&c.ID, &c.UserID, &c.CompanyID, &c.CreatedAt)
	if err == nil {
		return &c, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	// ON CONFLICT — на случай гонки параллельных первых сообщений.
	err = r.pool.QueryRow(ctx, `
		INSERT INTO ai_assistant_conversations (user_id, company_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, company_id) DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING id, user_id, company_id, created_at`, userID, companyID).
		Scan(&c.ID, &c.UserID, &c.CompanyID, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *AssistantRepo) RecentMessages(ctx context.Context, conversationID int64, limit int) ([]domain.AssistantMessage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, conversation_id, role, text, sources, created_at
		  FROM (
		      SELECT id, conversation_id, role, text, sources, created_at
		        FROM ai_assistant_messages
		       WHERE conversation_id = $1
		       ORDER BY created_at DESC, id DESC
		       LIMIT $2
		  ) recent
		 ORDER BY created_at ASC, id ASC`, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssistantMessages(rows)
}

// History — с голосом владельца (my_feedback). LEFT JOIN без фильтра по
// user_id корректен: UpsertFeedback пишет голоса только владельца диалога,
// других строк по этим сообщениям не бывает (UNIQUE message_id+user_id).
func (r *AssistantRepo) History(ctx context.Context, conversationID int64, limit int, before *time.Time) ([]domain.AssistantMessage, error) {
	var rows pgx.Rows
	var err error
	if before != nil {
		rows, err = r.pool.Query(ctx, `
			SELECT m.id, m.conversation_id, m.role, m.text, m.sources, m.created_at, f.verdict
			  FROM ai_assistant_messages m
			  LEFT JOIN ai_assistant_feedback f ON f.message_id = m.id
			 WHERE m.conversation_id = $1 AND m.created_at < $2
			 ORDER BY m.created_at DESC, m.id DESC
			 LIMIT $3`, conversationID, *before, limit)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT m.id, m.conversation_id, m.role, m.text, m.sources, m.created_at, f.verdict
			  FROM ai_assistant_messages m
			  LEFT JOIN ai_assistant_feedback f ON f.message_id = m.id
			 WHERE m.conversation_id = $1
			 ORDER BY m.created_at DESC, m.id DESC
			 LIMIT $2`, conversationID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AssistantMessage{}
	for rows.Next() {
		var m domain.AssistantMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Text, &m.Sources, &m.CreatedAt, &m.MyFeedback); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *AssistantRepo) AppendMessage(ctx context.Context, conversationID int64, role, text string, sources *string) (*domain.AssistantMessage, error) {
	var m domain.AssistantMessage
	err := r.pool.QueryRow(ctx, `
		INSERT INTO ai_assistant_messages (conversation_id, role, text, sources)
		VALUES ($1, $2, $3, $4)
		RETURNING id, conversation_id, role, text, sources, created_at`,
		conversationID, role, text, sources).
		Scan(&m.ID, &m.ConversationID, &m.Role, &m.Text, &m.Sources, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// UpsertFeedback — INSERT … SELECT с проверкой принадлежности в одном
// запросе: сообщение должно быть ответом ассистента в диалоге именно этой
// пары (userID, companyID) — чужой message_id не проходит фильтр SELECT и
// возвращает false без ошибки.
func (r *AssistantRepo) UpsertFeedback(ctx context.Context, messageID, userID, companyID int64, verdict string, reason *string) (bool, error) {
	// Явные касты обязательны: $2 стоит и в SELECT-списке, и в WHERE —
	// без каста Postgres не может однозначно вывести тип параметра
	// («inconsistent types deduced», SQLSTATE 42P08).
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO ai_assistant_feedback (message_id, user_id, verdict, reason)
		SELECT m.id, $2::bigint, $4::text, $5::text
		  FROM ai_assistant_messages m
		  JOIN ai_assistant_conversations c ON c.id = m.conversation_id
		 WHERE m.id = $1 AND m.role = 'assistant'
		   AND c.user_id = $2::bigint AND c.company_id = $3
		ON CONFLICT (message_id, user_id)
		DO UPDATE SET verdict = EXCLUDED.verdict, reason = EXCLUDED.reason, created_at = now()`,
		messageID, userID, companyID, verdict, reason)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func scanAssistantMessages(rows pgx.Rows) ([]domain.AssistantMessage, error) {
	out := []domain.AssistantMessage{}
	for rows.Next() {
		var m domain.AssistantMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Text, &m.Sources, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
