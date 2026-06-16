// Package domain — модели и порты микросервиса мессенджера.
//
// Таблицами conversations/messages/message_attachments в рантайме владеет
// этот сервис; схему ведёт migrate-контейнер (goose). Чужие
// таблицы (users, companies, pets, tasks, calls) читаются read-only.
package domain

import "time"

// Сторона диалога: hidden_for_*/pinned_at_* раскладываются по сторонам
// нормализованной пары user_a_id < user_b_id. Пустая сторона ("") в выборках
// означает «без фильтра по скрытым» (соло-чаты и support-inbox).
const (
	SideA = "a"
	SideB = "b"
)

// Виды сообщений (messages.kind).
const (
	KindText     = "text"
	KindCall     = "call"
	KindTask     = "task"
	KindDevReply = "system_dev_reply"
)

// Conversation — диалог. Пара user_a<user_b либо соло-чат (dev/pet,
// user_b NULL, владелец — user_a).
type Conversation struct {
	ID            int64
	UserAID       int64
	UserBID       *int64
	CompanyID     *int64  // NULL — переписка людей без общей компании
	CompanyName   *string // подгружается листингами (JOIN companies)
	IsDevChat     bool
	IsPetChat     bool
	CreatedAt     time.Time
	LastMessageAt *time.Time
	HiddenForA    bool
	HiddenForB    bool
	PinnedAtA     *time.Time
	PinnedAtB     *time.Time
}

// IsSolo — чат без «второй стороны»: dev-чат или чат с Грувиком.
func (c *Conversation) IsSolo() bool { return c.IsDevChat || c.IsPetChat }

// OtherUserID — собеседник; nil для соло-чатов.
func (c *Conversation) OtherUserID(userID int64) *int64 {
	if c.IsSolo() {
		return nil
	}
	if c.UserAID == userID {
		return c.UserBID
	}
	a := c.UserAID
	return &a
}

// Side — 'a', если userID == user_a_id, иначе 'b'. Для соло-чатов всегда 'a'.
func (c *Conversation) Side(userID int64) string {
	if c.IsSolo() || c.UserAID == userID {
		return SideA
	}
	return SideB
}

func (c *Conversation) PinnedAtFor(userID int64) *time.Time {
	if c.IsSolo() {
		return nil
	}
	if c.Side(userID) == SideA {
		return c.PinnedAtA
	}
	return c.PinnedAtB
}

// Attachment — файл-вложение. message_id NULL до привязки к сообщению.
type Attachment struct {
	ID         int64
	MessageID  *int64
	UploaderID int64
	FilePath   string
	FileName   string
	MimeType   string
	SizeBytes  int64
	CreatedAt  time.Time
}

// ReplyPreview — выжимка цитируемого сообщения (без рекурсии).
type ReplyPreview struct {
	ID             int64
	SenderID       *int64
	SenderFIO      *string
	Text           *string
	HasAttachments bool
	Kind           string
}

// UserRef — ссылка на пользователя для метки «Переслано от …».
type UserRef struct {
	ID  int64
	FIO string
}

// CallInfo — снапшот звонка для плашки kind='call' (read-only из calls).
type CallInfo struct {
	ID             int64
	Kind           string
	Media          string
	Status         string
	StartedAt      time.Time
	EndedAt        *time.Time
	InitiatorID    int64
	ConversationID *int64 // не сериализуется; нужен GetCallMessage
}

// TaskPreview — превью прикреплённой задачи (read-only из tasks).
type TaskPreview struct {
	ID             int64
	Name           string
	IsArchived     bool
	Color          *string
	ResponsibleFIO *string
	Deadline       *time.Time
	CompanyID      int64 // не сериализуется; проверка TASK_WRONG_COMPANY
}

// Message — сообщение со всем, что сериализует снапшот REST-ответа
// (аналог _msg_load_options() во Flask-репозитории).
type Message struct {
	ID                  int64
	ConversationID      int64
	SenderID            *int64
	IsBot               bool
	Text                *string
	CreatedAt           time.Time
	ReadAt              *time.Time
	HiddenForA          bool
	HiddenForB          bool
	ReplyToID           *int64
	ForwardedFromUserID *int64
	Kind                string
	CallID              *int64
	TaskID              *int64
	PinnedAt            *time.Time
	PinnedByID          *int64

	Attachments   []Attachment
	ReplyTo       *ReplyPreview
	ForwardedFrom *UserRef
	Call          *CallInfo
	Task          *TaskPreview

	// Контекст диалога для is_from_support (conversation.is_dev_chat,
	// владелец = conversation.user_a_id).
	ConvIsDevChat bool
	ConvOwnerID   int64
}

// IsFromSupport — сообщение в dev-чате не от владельца (фронт подписывает
// «Техподдержка»). Бот-автоответ (sender NULL) — тоже от поддержки.
func (m *Message) IsFromSupport() bool {
	if !m.ConvIsDevChat {
		return false
	}
	return m.SenderID == nil || *m.SenderID != m.ConvOwnerID
}

// NewMessage — параметры создания сообщения (create_message во Flask).
type NewMessage struct {
	ConversationID      int64
	SenderID            *int64
	Text                *string
	AttachmentIDs       []int64
	ReplyToID           *int64
	ForwardedFromUserID *int64
	Kind                string
	TaskID              *int64
	CallID              *int64
	IsBot               bool
}

// User — пользователь платформы в объёме мессенджера (read-only).
//
// Идентичность (id, fio, login, avatar, контакты, активность) грузится из
// users; роль и принадлежность к компании развязаны с users — RoleLevel и
// CompanyID заполняются ИЗ ТОКЕНА в authSource (актуальны только для self),
// в БД их нет.
type User struct {
	ID            int64
	FIO           string
	Login         string
	RoleLevel     int     // из токена (active company), не из users
	CompanyID     *int64  // из токена (active company), не из users
	Phone         *string
	Email         *string
	AvatarPath    *string
	IsActive      bool
	IsSuperAdmin  bool
	CompanyActive bool
	LastSeenAt    *time.Time
}
