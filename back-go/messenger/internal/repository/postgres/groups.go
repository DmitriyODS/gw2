package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

// memberCols — колонки участника группы (без пользователя).
const memberCols = `m.conversation_id, m.user_id, m.role, m.joined_at,
	m.last_read_message_id, m.last_read_at, m.pinned_at, m.hidden_at, m.muted,
	m.can_manage_members, m.can_edit_info, m.can_pin_messages`

func scanMember(row pgx.Row, dst ...any) (*domain.Member, error) {
	var m domain.Member
	base := []any{&m.ConversationID, &m.UserID, &m.Role, &m.JoinedAt,
		&m.LastReadMessageID, &m.LastReadAt, &m.PinnedAt, &m.HiddenAt, &m.Muted,
		&m.CanManageMembers, &m.CanEditInfo, &m.CanPinMessages}
	if err := row.Scan(append(base, dst...)...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *Repo) CreateGroup(ctx context.Context, title string, avatarPath *string,
	creatorID int64, memberIDs []int64) (*domain.Conversation, error) {

	var conv *domain.Conversation
	err := r.RunInTx(ctx, func(ctx context.Context) error {
		var id int64
		if err := r.q(ctx).QueryRow(ctx, `
			INSERT INTO conversations (is_group, title, avatar_path, created_by,
				is_dev_chat, created_at)
			VALUES (TRUE, $1, $2, $3, FALSE, now())
			RETURNING id`, title, avatarPath, creatorID).Scan(&id); err != nil {
			return err
		}
		// Владелец + участники (дедуп, создатель гарантированно owner).
		seen := map[int64]bool{creatorID: true}
		if _, err := r.q(ctx).Exec(ctx, `
			INSERT INTO conversation_members (conversation_id, user_id, role)
			VALUES ($1, $2, 'owner')`, id, creatorID); err != nil {
			return err
		}
		for _, uid := range memberIDs {
			if uid == 0 || seen[uid] {
				continue
			}
			seen[uid] = true
			if _, err := r.q(ctx).Exec(ctx, `
				INSERT INTO conversation_members (conversation_id, user_id, role)
				VALUES ($1, $2, 'member')
				ON CONFLICT (conversation_id, user_id) DO NOTHING`, id, uid); err != nil {
				return err
			}
		}
		c, err := r.GetConversation(ctx, id)
		conv = c
		return err
	})
	return conv, err
}

func (r *Repo) ListGroupConversations(ctx context.Context, userID int64) ([]*domain.Conversation, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT `+convCols+`,
			m.role, m.muted, m.pinned_at, m.last_read_message_id,
			(SELECT count(*) FROM conversation_members mm WHERE mm.conversation_id = c.id)
		FROM conversations c
		LEFT JOIN companies co ON co.id = c.company_id
		JOIN conversation_members m ON m.conversation_id = c.id AND m.user_id = $1
		WHERE c.is_group = TRUE AND m.hidden_at IS NULL
		ORDER BY c.last_message_at DESC NULLS LAST, c.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Conversation
	for rows.Next() {
		var c domain.Conversation
		var aID *int64
		if err := rows.Scan(&c.ID, &aID, &c.UserBID, &c.CompanyID, &c.CompanyName,
			&c.IsDevChat, &c.CreatedAt, &c.LastMessageAt,
			&c.HiddenForA, &c.HiddenForB, &c.PinnedAtA, &c.PinnedAtB,
			&c.IsGroup, &c.Title, &c.AvatarPath, &c.CreatedBy, &c.InviteCode,
			&c.MyRole, &c.MyMuted, &c.MyPinnedAt, &c.MyLastReadID, &c.MemberCount); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (r *Repo) RenameGroup(ctx context.Context, convID int64, title string) error {
	_, err := r.q(ctx).Exec(ctx, `UPDATE conversations SET title = $2 WHERE id = $1`, convID, title)
	return err
}

func (r *Repo) SetGroupAvatar(ctx context.Context, convID int64, avatarPath *string) error {
	_, err := r.q(ctx).Exec(ctx, `UPDATE conversations SET avatar_path = $2 WHERE id = $1`, convID, avatarPath)
	return err
}

func (r *Repo) SetInviteCode(ctx context.Context, convID int64, code *string) error {
	_, err := r.q(ctx).Exec(ctx, `UPDATE conversations SET invite_code = $2 WHERE id = $1`, convID, code)
	return err
}

func (r *Repo) FindByInviteCode(ctx context.Context, code string) (*domain.Conversation, error) {
	return scanConversation(r.q(ctx).QueryRow(ctx,
		`SELECT `+convCols+convFrom+`WHERE c.invite_code = $1 AND c.is_group = TRUE`, code))
}

func (r *Repo) AddMember(ctx context.Context, convID, userID int64, role string) error {
	_, err := r.q(ctx).Exec(ctx, `
		INSERT INTO conversation_members (conversation_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (conversation_id, user_id) DO UPDATE SET hidden_at = NULL`,
		convID, userID, role)
	return err
}

func (r *Repo) RemoveMember(ctx context.Context, convID, userID int64) error {
	_, err := r.q(ctx).Exec(ctx,
		`DELETE FROM conversation_members WHERE conversation_id = $1 AND user_id = $2`, convID, userID)
	return err
}

func (r *Repo) GetMember(ctx context.Context, convID, userID int64) (*domain.Member, error) {
	return scanMember(r.q(ctx).QueryRow(ctx,
		`SELECT `+memberCols+` FROM conversation_members m
		 WHERE m.conversation_id = $1 AND m.user_id = $2`, convID, userID))
}

func (r *Repo) ListMembers(ctx context.Context, convID int64) ([]*domain.Member, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT `+memberCols+`, `+userCols+`
		FROM conversation_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.conversation_id = $1
		ORDER BY (m.role = 'owner') DESC, (m.role = 'admin') DESC, m.joined_at ASC`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Member
	for rows.Next() {
		var m domain.Member
		var u domain.User
		if err := rows.Scan(&m.ConversationID, &m.UserID, &m.Role, &m.JoinedAt,
			&m.LastReadMessageID, &m.LastReadAt, &m.PinnedAt, &m.HiddenAt, &m.Muted,
			&m.CanManageMembers, &m.CanEditInfo, &m.CanPinMessages,
			&u.ID, &u.FIO, &u.Login, &u.AvatarPath, &u.Phone, &u.Email,
			&u.IsActive, &u.IsSuperAdmin, &u.LastSeenAt, &u.StatusEmoji, &u.StatusText); err != nil {
			return nil, err
		}
		m.User = &u
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (r *Repo) ListMemberMutes(ctx context.Context, convID int64) (map[int64]bool, error) {
	rows, err := r.q(ctx).Query(ctx,
		`SELECT user_id, muted FROM conversation_members WHERE conversation_id = $1`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64]bool{}
	for rows.Next() {
		var uid int64
		var muted bool
		if err := rows.Scan(&uid, &muted); err != nil {
			return nil, err
		}
		out[uid] = muted
	}
	return out, rows.Err()
}

func (r *Repo) UpdateMemberRole(ctx context.Context, convID, userID int64, role string) error {
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE conversation_members SET role = $3 WHERE conversation_id = $1 AND user_id = $2`,
		convID, userID, role)
	return err
}

func (r *Repo) UpdateMemberRights(ctx context.Context, convID, userID int64, manageMembers, editInfo, pinMessages bool) error {
	_, err := r.q(ctx).Exec(ctx, `
		UPDATE conversation_members
		SET can_manage_members = $3, can_edit_info = $4, can_pin_messages = $5
		WHERE conversation_id = $1 AND user_id = $2`,
		convID, userID, manageMembers, editInfo, pinMessages)
	return err
}

func (r *Repo) SetMemberMute(ctx context.Context, convID, userID int64, muted bool) error {
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE conversation_members SET muted = $3 WHERE conversation_id = $1 AND user_id = $2`,
		convID, userID, muted)
	return err
}

func (r *Repo) SetMemberPin(ctx context.Context, convID, userID int64, pinned bool) error {
	value := "NULL"
	if pinned {
		value = "now()"
	}
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE conversation_members SET pinned_at = `+value+
			` WHERE conversation_id = $1 AND user_id = $2`, convID, userID)
	return err
}

func (r *Repo) HideConversationMember(ctx context.Context, convID, userID int64, hidden bool) error {
	value := "NULL"
	if hidden {
		value = "now()"
	}
	_, err := r.q(ctx).Exec(ctx,
		`UPDATE conversation_members SET hidden_at = `+value+
			` WHERE conversation_id = $1 AND user_id = $2`, convID, userID)
	return err
}

func (r *Repo) SetMemberRead(ctx context.Context, convID, userID, lastMessageID int64) (bool, error) {
	tag, err := r.q(ctx).Exec(ctx, `
		UPDATE conversation_members
		SET last_read_message_id = $3, last_read_at = now()
		WHERE conversation_id = $1 AND user_id = $2
		  AND $3 > COALESCE(last_read_message_id, 0)`, convID, userID, lastMessageID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repo) ReadersOf(ctx context.Context, convID, messageID, authorID int64) ([]*domain.Member, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT `+memberCols+`, `+userCols+`
		FROM conversation_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.conversation_id = $1 AND m.user_id <> $3
		  AND m.last_read_message_id >= $2
		ORDER BY m.last_read_at DESC NULLS LAST`, convID, messageID, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Member
	for rows.Next() {
		var m domain.Member
		var u domain.User
		if err := rows.Scan(&m.ConversationID, &m.UserID, &m.Role, &m.JoinedAt,
			&m.LastReadMessageID, &m.LastReadAt, &m.PinnedAt, &m.HiddenAt, &m.Muted,
			&m.CanManageMembers, &m.CanEditInfo, &m.CanPinMessages,
			&u.ID, &u.FIO, &u.Login, &u.AvatarPath, &u.Phone, &u.Email,
			&u.IsActive, &u.IsSuperAdmin, &u.LastSeenAt, &u.StatusEmoji, &u.StatusText); err != nil {
			return nil, err
		}
		m.User = &u
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (r *Repo) CountGroupUnread(ctx context.Context, convIDs []int64, userID int64) (map[int64]int, error) {
	out := map[int64]int{}
	if len(convIDs) == 0 {
		return out, nil
	}
	rows, err := r.q(ctx).Query(ctx, `
		SELECT m.conversation_id, count(msg.id)
		FROM conversation_members m
		JOIN messages msg ON msg.conversation_id = m.conversation_id
		  AND msg.id > COALESCE(m.last_read_message_id, 0)
		  AND msg.sender_id IS DISTINCT FROM $2
		WHERE m.user_id = $2 AND m.conversation_id = ANY($1)
		GROUP BY m.conversation_id`, convIDs, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int64
		var n int
		if err := rows.Scan(&cid, &n); err != nil {
			return nil, err
		}
		out[cid] = n
	}
	return out, rows.Err()
}

func (r *Repo) TotalGroupUnread(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.q(ctx).QueryRow(ctx, `
		SELECT count(msg.id)
		FROM conversation_members m
		JOIN messages msg ON msg.conversation_id = m.conversation_id
		  AND msg.id > COALESCE(m.last_read_message_id, 0)
		  AND msg.sender_id IS DISTINCT FROM $1
		WHERE m.user_id = $1 AND m.hidden_at IS NULL`, userID).Scan(&n)
	return n, err
}
