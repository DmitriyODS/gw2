package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

// msgCols/msgFrom — полный снапшот сообщения: всё, что сериализует
// MessageSchema (аналог _msg_load_options() во Flask), одним запросом.
const msgCols = `m.id, m.conversation_id, m.sender_id, m.is_bot, m.text, m.created_at,
	m.read_at, m.hidden_for_a, m.hidden_for_b, m.reply_to_id, m.forwarded_from_user_id,
	m.kind, m.call_id, m.task_id, m.pinned_at, m.pinned_by_id,
	c.is_dev_chat, c.user_a_id,
	r.id, r.sender_id, ru.fio, r.text, r.kind,
	EXISTS(SELECT 1 FROM message_attachments ra WHERE ra.message_id = r.id),
	fu.id, fu.fio,
	cl.id, cl.kind, cl.media, cl.status, cl.started_at, cl.ended_at, cl.initiator_id, cl.conversation_id,
	t.id, t.name, t.is_archived, t.color, tu.fio, t.deadline, t.company_id`

const msgFrom = `
	FROM messages m
	JOIN conversations c ON c.id = m.conversation_id
	LEFT JOIN messages r ON r.id = m.reply_to_id
	LEFT JOIN users ru ON ru.id = r.sender_id
	LEFT JOIN users fu ON fu.id = m.forwarded_from_user_id
	LEFT JOIN calls cl ON cl.id = m.call_id
	LEFT JOIN tasks t ON t.id = m.task_id
	LEFT JOIN users tu ON tu.id = t.responsible_user_id `

func scanMessage(row pgx.Row) (*domain.Message, error) {
	var (
		m domain.Message

		replyID    *int64
		replySndID *int64
		replyFIO   *string
		replyText  *string
		replyKind  *string
		replyAtt   *bool

		fwdID  *int64
		fwdFIO *string

		callID        *int64
		callKind      *string
		callMedia     *string
		callStatus    *string
		callStartedAt *time.Time
		callEndedAt   *time.Time
		callInitiator *int64
		callConvID    *int64

		taskID       *int64
		taskName     *string
		taskArchived *bool
		taskColor    *string
		taskRespFIO  *string
		taskDeadline *time.Time
		taskCompany  *int64
	)
	err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.IsBot, &m.Text, &m.CreatedAt,
		&m.ReadAt, &m.HiddenForA, &m.HiddenForB, &m.ReplyToID, &m.ForwardedFromUserID,
		&m.Kind, &m.CallID, &m.TaskID, &m.PinnedAt, &m.PinnedByID,
		&m.ConvIsDevChat, &m.ConvOwnerID,
		&replyID, &replySndID, &replyFIO, &replyText, &replyKind, &replyAtt,
		&fwdID, &fwdFIO,
		&callID, &callKind, &callMedia, &callStatus, &callStartedAt, &callEndedAt,
		&callInitiator, &callConvID,
		&taskID, &taskName, &taskArchived, &taskColor, &taskRespFIO, &taskDeadline, &taskCompany)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if replyID != nil {
		m.ReplyTo = &domain.ReplyPreview{
			ID:             *replyID,
			SenderID:       replySndID,
			SenderFIO:      replyFIO,
			Text:           replyText,
			HasAttachments: replyAtt != nil && *replyAtt,
			Kind:           deref(replyKind),
		}
	}
	if fwdID != nil {
		m.ForwardedFrom = &domain.UserRef{ID: *fwdID, FIO: deref(fwdFIO)}
	}
	if callID != nil {
		m.Call = &domain.CallInfo{
			ID:             *callID,
			Kind:           deref(callKind),
			Media:          deref(callMedia),
			Status:         deref(callStatus),
			StartedAt:      derefTime(callStartedAt),
			EndedAt:        callEndedAt,
			InitiatorID:    derefInt(callInitiator),
			ConversationID: callConvID,
		}
	}
	if taskID != nil {
		m.Task = &domain.TaskPreview{
			ID:             *taskID,
			Name:           deref(taskName),
			IsArchived:     taskArchived != nil && *taskArchived,
			Color:          taskColor,
			ResponsibleFIO: taskRespFIO,
			Deadline:       taskDeadline,
			CompanyID:      derefInt(taskCompany),
		}
	}
	return &m, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// queryMessages — выборка полных снапшотов + батч-подгрузка вложений.
func (r *Repo) queryMessages(ctx context.Context, sql string, args ...any) ([]*domain.Message, error) {
	rows, err := r.q(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.loadAttachments(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repo) loadAttachments(ctx context.Context, msgs []*domain.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	byID := make(map[int64]*domain.Message, len(msgs))
	ids := make([]int64, 0, len(msgs))
	for _, m := range msgs {
		m.Attachments = []domain.Attachment{}
		byID[m.ID] = m
		ids = append(ids, m.ID)
	}
	rows, err := r.q(ctx).Query(ctx, `
		SELECT id, message_id, uploader_id, file_path, file_name, mime_type, size_bytes, created_at
		FROM message_attachments WHERE message_id = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var a domain.Attachment
		if err := rows.Scan(&a.ID, &a.MessageID, &a.UploaderID, &a.FilePath,
			&a.FileName, &a.MimeType, &a.SizeBytes, &a.CreatedAt); err != nil {
			return err
		}
		if m := byID[*a.MessageID]; m != nil {
			m.Attachments = append(m.Attachments, a)
		}
	}
	return rows.Err()
}

func (r *Repo) GetMessage(ctx context.Context, id int64) (*domain.Message, error) {
	msgs, err := r.queryMessages(ctx, `SELECT `+msgCols+msgFrom+`WHERE m.id = $1`, id)
	if err != nil || len(msgs) == 0 {
		return nil, err
	}
	return msgs[0], nil
}

func (r *Repo) ListMessages(ctx context.Context, convID int64, side string,
	beforeID, afterID *int64, limit int) ([]*domain.Message, error) {

	where := `WHERE m.conversation_id = $1 AND m.` + hiddenCol(side) + ` = FALSE`
	args := []any{convID}
	if beforeID != nil {
		args = append(args, *beforeID)
		where += fmt.Sprintf(" AND m.id < $%d", len(args))
	}
	if afterID != nil {
		args = append(args, *afterID)
		where += fmt.Sprintf(" AND m.id > $%d", len(args))
		// При after_id — прямой порядок (старые → новые), без переворота.
		args = append(args, limit)
		return r.queryMessages(ctx, `SELECT `+msgCols+msgFrom+where+
			fmt.Sprintf(` ORDER BY m.id ASC LIMIT $%d`, len(args)), args...)
	}
	args = append(args, limit)
	msgs, err := r.queryMessages(ctx, `SELECT `+msgCols+msgFrom+where+
		fmt.Sprintf(` ORDER BY m.id DESC LIMIT $%d`, len(args)), args...)
	if err != nil {
		return nil, err
	}
	reverse(msgs)
	return msgs, nil
}

func reverse(msgs []*domain.Message) {
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
}

func (r *Repo) ListPinned(ctx context.Context, convID int64, side string) ([]*domain.Message, error) {
	return r.queryMessages(ctx, `SELECT `+msgCols+msgFrom+`
		WHERE m.conversation_id = $1 AND m.pinned_at IS NOT NULL AND m.`+hiddenCol(side)+` = FALSE
		ORDER BY m.pinned_at DESC`, convID)
}

func (r *Repo) LastVisibleMessages(ctx context.Context, convIDs []int64, side string) (map[int64]*domain.Message, error) {
	out := map[int64]*domain.Message{}
	if len(convIDs) == 0 {
		return out, nil
	}
	hidden := ""
	if side != "" {
		hidden = ` AND m.` + hiddenCol(side) + ` = FALSE`
	}
	msgs, err := r.queryMessages(ctx, `SELECT DISTINCT ON (m.conversation_id) `+msgCols+msgFrom+`
		WHERE m.conversation_id = ANY($1)`+hidden+`
		ORDER BY m.conversation_id, m.id DESC`, convIDs)
	if err != nil {
		return nil, err
	}
	for _, m := range msgs {
		out[m.ConversationID] = m
	}
	return out, nil
}

func (r *Repo) CountUnread(ctx context.Context, convIDs []int64, userID int64, side string) (map[int64]int, error) {
	out := map[int64]int{}
	if len(convIDs) == 0 {
		return out, nil
	}
	hidden := ""
	if side != "" {
		hidden = ` AND ` + hiddenCol(side) + ` = FALSE`
	}
	// Явный OR sender IS NULL: иначе трёхзначная логика SQL молча теряет
	// бот-сообщения (Грувик и автоответ техподдержки идут с sender NULL).
	rows, err := r.q(ctx).Query(ctx, `
		SELECT conversation_id, COUNT(id) FROM messages
		WHERE conversation_id = ANY($1)
		  AND (sender_id IS NULL OR sender_id != $2)
		  AND read_at IS NULL`+hidden+`
		GROUP BY conversation_id`, convIDs, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var convID int64
		var n int
		if err := rows.Scan(&convID, &n); err != nil {
			return nil, err
		}
		out[convID] = n
	}
	return out, rows.Err()
}

func (r *Repo) CountUnreadFromSenders(ctx context.Context, convIDs, senderIDs []int64) (map[int64]int, error) {
	out := map[int64]int{}
	if len(convIDs) == 0 || len(senderIDs) == 0 {
		return out, nil
	}
	rows, err := r.q(ctx).Query(ctx, `
		SELECT conversation_id, COUNT(id) FROM messages
		WHERE conversation_id = ANY($1) AND sender_id = ANY($2) AND read_at IS NULL
		GROUP BY conversation_id`, convIDs, senderIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var convID int64
		var n int
		if err := rows.Scan(&convID, &n); err != nil {
			return nil, err
		}
		out[convID] = n
	}
	return out, rows.Err()
}

func (r *Repo) TotalUnread(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `
		SELECT COUNT(m.id)
		FROM messages m
		JOIN conversations c ON c.id = m.conversation_id
		WHERE ((c.user_a_id = $1 AND c.hidden_for_a = FALSE AND m.hidden_for_a = FALSE)
		    OR (c.user_b_id = $1 AND c.hidden_for_b = FALSE AND m.hidden_for_b = FALSE))
		  AND (m.sender_id IS NULL OR m.sender_id != $1)
		  AND m.read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *Repo) CreateMessage(ctx context.Context, nm domain.NewMessage) (*domain.Message, error) {
	kind := nm.Kind
	if kind == "" {
		kind = domain.KindText
	}
	var msg *domain.Message
	err := r.RunInTx(ctx, func(ctx context.Context) error {
		var id int64
		var createdAt time.Time
		err := r.q(ctx).QueryRow(ctx, `
			INSERT INTO messages (conversation_id, sender_id, is_bot, text, created_at,
				hidden_for_a, hidden_for_b, reply_to_id, forwarded_from_user_id,
				kind, call_id, task_id)
			VALUES ($1, $2, $3, $4, now(), FALSE, FALSE, $5, $6, $7, $8, $9)
			RETURNING id, created_at`,
			nm.ConversationID, nm.SenderID, nm.IsBot, nm.Text, nm.ReplyToID,
			nm.ForwardedFromUserID, kind, nm.CallID, nm.TaskID,
		).Scan(&id, &createdAt)
		if err != nil {
			return err
		}
		if len(nm.AttachmentIDs) > 0 && nm.SenderID != nil {
			if _, err := r.q(ctx).Exec(ctx, `
				UPDATE message_attachments SET message_id = $1
				WHERE id = ANY($2) AND uploader_id = $3 AND message_id IS NULL`,
				id, nm.AttachmentIDs, *nm.SenderID); err != nil {
				return err
			}
		}
		// last_message_at + «возврат» диалога обеим сторонам, если кто-то
		// его раньше скрыл у себя.
		if _, err := r.q(ctx).Exec(ctx, `
			UPDATE conversations
			SET last_message_at = $2, hidden_for_a = FALSE, hidden_for_b = FALSE
			WHERE id = $1`, nm.ConversationID, createdAt); err != nil {
			return err
		}
		msg, err = r.GetMessage(ctx, id)
		return err
	})
	return msg, err
}

func (r *Repo) MarkRead(ctx context.Context, convID, readerID int64) (int, error) {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE messages SET read_at = now()
		WHERE conversation_id = $1
		  AND (sender_id IS NULL OR sender_id != $2)
		  AND read_at IS NULL`, convID, readerID)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *Repo) HideMessage(ctx context.Context, id int64, side string) (bool, error) {
	var both bool
	err := r.q(ctx).QueryRow(ctx,
		`UPDATE messages SET `+hiddenCol(side)+` = TRUE WHERE id = $1
		 RETURNING hidden_for_a AND hidden_for_b`, id).Scan(&both)
	return both, err
}

// DeleteMessage — вложения каскадно уходят по FK; файлы на диске удаляет
// вызывающий.
func (r *Repo) DeleteMessage(ctx context.Context, id int64) error {
	_, err := r.q(ctx).Exec(ctx, `DELETE FROM messages WHERE id = $1`, id)
	return err
}

func (r *Repo) RecomputeLastMessageAt(ctx context.Context, convID int64) error {
	_, err := r.q(ctx).Exec(ctx, `
		UPDATE conversations
		SET last_message_at = (SELECT MAX(created_at) FROM messages WHERE conversation_id = $1)
		WHERE id = $1`, convID)
	return err
}

func (r *Repo) SetMessagePin(ctx context.Context, id int64, pinned bool, byID *int64) error {
	if pinned {
		_, err := r.q(ctx).Exec(ctx,
			`UPDATE messages SET pinned_at = now(), pinned_by_id = $2 WHERE id = $1`, id, byID)
		return err
	}
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE messages SET pinned_at = NULL, pinned_by_id = NULL WHERE id = $1`, id)
	return err
}

func (r *Repo) HasHumanMessageSince(ctx context.Context, convID int64, since time.Time, beforeID int64) (bool, error) {
	var exists bool
	err := r.q(ctx).QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM messages
			WHERE conversation_id = $1 AND id < $2 AND is_bot = FALSE AND created_at >= $3
		)`, convID, beforeID, since).Scan(&exists)
	return exists, err
}

func (r *Repo) ListRecent(ctx context.Context, convID int64, limit int) ([]*domain.Message, error) {
	msgs, err := r.queryMessages(ctx, `SELECT `+msgCols+msgFrom+`
		WHERE m.conversation_id = $1
		ORDER BY m.id DESC LIMIT $2`, convID, limit)
	if err != nil {
		return nil, err
	}
	reverse(msgs)
	return msgs, nil
}

func (r *Repo) FindCallMessage(ctx context.Context, callID, convID int64) (*domain.Message, error) {
	msgs, err := r.queryMessages(ctx, `SELECT `+msgCols+msgFrom+`
		WHERE m.call_id = $1 AND m.kind = 'call' AND m.conversation_id = $2
		ORDER BY m.id DESC LIMIT 1`, callID, convID)
	if err != nil || len(msgs) == 0 {
		return nil, err
	}
	return msgs[0], nil
}

func (r *Repo) ListAttachmentPathsOfConversation(ctx context.Context, convID int64) ([]string, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT a.file_path
		FROM message_attachments a
		JOIN messages m ON m.id = a.message_id
		WHERE m.conversation_id = $1`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repo) CreateAttachment(ctx context.Context, att *domain.Attachment) error {
	return r.q(ctx).QueryRow(ctx, `
		INSERT INTO message_attachments (message_id, uploader_id, file_path, file_name,
			mime_type, size_bytes, created_at)
		VALUES (NULL, $1, $2, $3, $4, $5, now())
		RETURNING id, created_at`,
		att.UploaderID, att.FilePath, att.FileName, att.MimeType, att.SizeBytes,
	).Scan(&att.ID, &att.CreatedAt)
}

func (r *Repo) GetAttachment(ctx context.Context, id int64) (*domain.Attachment, error) {
	var a domain.Attachment
	err := r.q(ctx).QueryRow(ctx, `
		SELECT id, message_id, uploader_id, file_path, file_name, mime_type, size_bytes, created_at
		FROM message_attachments WHERE id = $1`, id,
	).Scan(&a.ID, &a.MessageID, &a.UploaderID, &a.FilePath, &a.FileName,
		&a.MimeType, &a.SizeBytes, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}
