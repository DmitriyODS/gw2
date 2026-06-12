// Package dto — transfer-объекты HTTP/событийного контракта. Формы JSON
// байт-в-байт совместимы с прежними marshmallow-схемами Flask
// (schemas/message.py, schemas/user.py) — фронт не меняется.
package dto

import (
	"time"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

// JSONTime — формат marshmallow (ISO8601 с явным смещением +00:00, не "Z").
type JSONTime time.Time

func (t JSONTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).UTC().Format("2006-01-02T15:04:05.999999-07:00") + `"`), nil
}

func jsonTimePtr(t *time.Time) *JSONTime {
	if t == nil {
		return nil
	}
	jt := JSONTime(*t)
	return &jt
}

// JSONTimePtr — экспортированный вариант для сборки DTO в сервисном слое.
func JSONTimePtr(t *time.Time) *JSONTime { return jsonTimePtr(t) }

// ── Пользователи (UserDirectorySchema) ───────────────────────────

type RoleRef struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// DirectoryUser — публичный профиль (без внутренних полей).
type DirectoryUser struct {
	ID         int64     `json:"id"`
	FIO        string    `json:"fio"`
	Login      string    `json:"login"`
	Post       *string   `json:"post"`
	Role       RoleRef   `json:"role"`
	CompanyID  *int64    `json:"company_id"`
	Phone      *string   `json:"phone"`
	Email      *string   `json:"email"`
	AvatarPath *string   `json:"avatar_path"`
	LastSeenAt *JSONTime `json:"last_seen_at"`
}

func NewDirectoryUser(u *domain.User) *DirectoryUser {
	if u == nil {
		return nil
	}
	return &DirectoryUser{
		ID:         u.ID,
		FIO:        u.FIO,
		Login:      u.Login,
		Post:       u.Post,
		Role:       RoleRef{ID: u.RoleID, Name: u.RoleName, Level: u.RoleLevel},
		CompanyID:  u.CompanyID,
		Phone:      u.Phone,
		Email:      u.Email,
		AvatarPath: u.AvatarPath,
		LastSeenAt: jsonTimePtr(u.LastSeenAt),
	}
}

// ── Сообщения (MessageSchema и вложенные) ────────────────────────

type Attachment struct {
	ID        int64  `json:"id"`
	FileName  string `json:"file_name"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	URL       string `json:"url"`
}

func NewAttachment(a *domain.Attachment) *Attachment {
	return &Attachment{
		ID:        a.ID,
		FileName:  a.FileName,
		MimeType:  a.MimeType,
		SizeBytes: a.SizeBytes,
		URL:       "/uploads/" + a.FilePath,
	}
}

type ReplyPreview struct {
	ID             int64   `json:"id"`
	SenderID       *int64  `json:"sender_id"`
	SenderFIO      *string `json:"sender_fio"`
	Text           *string `json:"text"`
	HasAttachments bool    `json:"has_attachments"`
	Kind           string  `json:"kind"`
}

type TaskPreview struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	IsArchived     bool      `json:"is_archived"`
	Color          *string   `json:"color"`
	ResponsibleFIO *string   `json:"responsible_fio"`
	Deadline       *JSONTime `json:"deadline"`
}

type CallInfo struct {
	ID          int64     `json:"id"`
	Kind        string    `json:"kind"`
	Media       string    `json:"media"`
	Status      string    `json:"status"`
	StartedAt   JSONTime  `json:"started_at"`
	EndedAt     *JSONTime `json:"ended_at"`
	InitiatorID int64     `json:"initiator_id"`
	DurationSec *int64    `json:"duration_sec"`
}

type ForwardedFrom struct {
	ID  int64  `json:"id"`
	FIO string `json:"fio"`
}

type Message struct {
	ID             int64          `json:"id"`
	ConversationID int64          `json:"conversation_id"`
	SenderID       *int64         `json:"sender_id"`
	IsBot          bool           `json:"is_bot"`
	Text           *string        `json:"text"`
	CreatedAt      JSONTime       `json:"created_at"`
	ReadAt         *JSONTime      `json:"read_at"`
	Attachments    []Attachment   `json:"attachments"`
	ReplyTo        *ReplyPreview  `json:"reply_to"`
	ForwardedFrom  *ForwardedFrom `json:"forwarded_from"`
	Kind           string         `json:"kind"`
	Call           *CallInfo      `json:"call"`
	Task           *TaskPreview   `json:"task"`
	PinnedAt       *JSONTime      `json:"pinned_at"`
	PinnedByID     *int64         `json:"pinned_by_id"`
	IsFromSupport  bool           `json:"is_from_support"`
}

// truncateRunes — как срез строки в Python (по символам, не байтам).
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func NewMessage(m *domain.Message) *Message {
	if m == nil {
		return nil
	}
	out := &Message{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		IsBot:          m.IsBot,
		Text:           m.Text,
		CreatedAt:      JSONTime(m.CreatedAt),
		ReadAt:         jsonTimePtr(m.ReadAt),
		Attachments:    make([]Attachment, 0, len(m.Attachments)),
		Kind:           m.Kind,
		PinnedAt:       jsonTimePtr(m.PinnedAt),
		PinnedByID:     m.PinnedByID,
		IsFromSupport:  m.IsFromSupport(),
	}
	for i := range m.Attachments {
		out.Attachments = append(out.Attachments, *NewAttachment(&m.Attachments[i]))
	}
	if r := m.ReplyTo; r != nil {
		rp := &ReplyPreview{
			ID:             r.ID,
			SenderID:       r.SenderID,
			SenderFIO:      r.SenderFIO,
			HasAttachments: r.HasAttachments,
			Kind:           r.Kind,
		}
		if r.Text != nil && *r.Text != "" {
			t := truncateRunes(*r.Text, 140)
			rp.Text = &t
		}
		out.ReplyTo = rp
	}
	if f := m.ForwardedFrom; f != nil {
		out.ForwardedFrom = &ForwardedFrom{ID: f.ID, FIO: f.FIO}
	}
	if c := m.Call; c != nil {
		ci := &CallInfo{
			ID:          c.ID,
			Kind:        c.Kind,
			Media:       c.Media,
			Status:      c.Status,
			StartedAt:   JSONTime(c.StartedAt),
			EndedAt:     jsonTimePtr(c.EndedAt),
			InitiatorID: c.InitiatorID,
		}
		if c.EndedAt != nil {
			d := int64(c.EndedAt.Sub(c.StartedAt).Seconds())
			ci.DurationSec = &d
		}
		out.Call = ci
	}
	if t := m.Task; t != nil {
		out.Task = &TaskPreview{
			ID:             t.ID,
			Name:           t.Name,
			IsArchived:     t.IsArchived,
			Color:          t.Color,
			ResponsibleFIO: t.ResponsibleFIO,
			Deadline:       jsonTimePtr(t.Deadline),
		}
	}
	return out
}

func NewMessages(ms []*domain.Message) []*Message {
	out := make([]*Message, 0, len(ms))
	for _, m := range ms {
		out = append(out, NewMessage(m))
	}
	return out
}

// ── Диалоги ──────────────────────────────────────────────────────

// Conversation — форма ConversationSchema.
type Conversation struct {
	ID            int64     `json:"id"`
	UserAID       int64     `json:"user_a_id"`
	UserBID       *int64    `json:"user_b_id"`
	CreatedAt     JSONTime  `json:"created_at"`
	LastMessageAt *JSONTime `json:"last_message_at"`
	IsDevChat     bool      `json:"is_dev_chat"`
	IsPetChat     bool      `json:"is_pet_chat"`
	PetName       *string   `json:"pet_name"`
	CompanyID     int64     `json:"company_id"`
}

func NewConversation(c *domain.Conversation, petName *string) *Conversation {
	return &Conversation{
		ID:            c.ID,
		UserAID:       c.UserAID,
		UserBID:       c.UserBID,
		CreatedAt:     JSONTime(c.CreatedAt),
		LastMessageAt: jsonTimePtr(c.LastMessageAt),
		IsDevChat:     c.IsDevChat,
		IsPetChat:     c.IsPetChat,
		PetName:       petName,
		CompanyID:     c.CompanyID,
	}
}

// ConversationWithOther — ответ POST /conversations: ConversationSchema +
// профиль собеседника.
type ConversationWithOther struct {
	Conversation
	OtherUser *DirectoryUser `json:"other_user"`
}

// ConversationListItem — форма ConversationListItemSchema.
type ConversationListItem struct {
	ID            int64          `json:"id"`
	OtherUser     *DirectoryUser `json:"other_user"`
	LastMessage   *Message       `json:"last_message"`
	UnreadCount   int            `json:"unread_count"`
	LastMessageAt *JSONTime      `json:"last_message_at"`
	IsPinned      bool           `json:"is_pinned"`
	PinnedAt      *JSONTime      `json:"pinned_at"`
	IsDevChat     bool           `json:"is_dev_chat"`
	IsPetChat     bool           `json:"is_pet_chat"`
	PetName       *string        `json:"pet_name"`
	CompanyID     int64          `json:"company_id"`
	CompanyName   *string        `json:"company_name"`
	OwnerUser     *DirectoryUser `json:"owner_user"`
}

// ── Запросы REST ─────────────────────────────────────────────────

// MessageCreate — тело POST /conversations/<id>/messages.
type MessageCreate struct {
	Text          *string `json:"text"`
	AttachmentIDs []int64 `json:"attachment_ids"`
	ReplyToID     *int64  `json:"reply_to_id"`
	TaskID        *int64  `json:"task_id"`
}

// ForwardRequest — тело POST /forward.
type ForwardRequest struct {
	MessageID       *int64  `json:"message_id"`
	ConversationIDs []int64 `json:"conversation_ids"`
	UserIDs         []int64 `json:"user_ids"`
}

// ForwardResult — элемент ответа {"forwarded": [...]}.
type ForwardResult struct {
	ConversationID int64    `json:"conversation_id"`
	Message        *Message `json:"message"`
}

// ── Payload'ы сокет-событий (эмитятся Flask-мостом вербатим) ─────

// MessageNewEvent — message:new.
type MessageNewEvent struct {
	ConversationID int64    `json:"conversation_id"`
	Message        *Message `json:"message"`
	FromUserID     *int64   `json:"from_user_id"`
}

// MessageReadEvent — message:read.
type MessageReadEvent struct {
	ConversationID int64 `json:"conversation_id"`
	ReaderID       int64 `json:"reader_id"`
}

// MessageDeletedEvent — message:deleted.
type MessageDeletedEvent struct {
	ConversationID int64 `json:"conversation_id"`
	MessageID      int64 `json:"message_id"`
}

// MessagePinEvent — message:pin.
type MessagePinEvent struct {
	ConversationID int64    `json:"conversation_id"`
	MessageID      int64    `json:"message_id"`
	Pinned         bool     `json:"pinned"`
	Message        *Message `json:"message"`
}

// ConversationDeletedEvent — conversation:deleted.
type ConversationDeletedEvent struct {
	ConversationID int64 `json:"conversation_id"`
}

// ConversationPinEvent — conversation:pin.
type ConversationPinEvent struct {
	ConversationID int64 `json:"conversation_id"`
	IsPinned       bool  `json:"is_pinned"`
}
