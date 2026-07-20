// Package domain — модели и порты микросервиса мессенджера.
//
// Таблицами conversations/messages/message_attachments в рантайме владеет
// этот сервис; схему ведёт migrate-контейнер (goose). Чужие
// таблицы (users, companies, tasks, calls) читаются read-only.
package domain

import (
	"encoding/json"
	"time"
)

// ChatBackground — персональный рецепт оформления чата. ConversationID == nil —
// общий дефолт пользователя; иначе переопределение конкретного чата. Recipe —
// непрозрачный для бэкенда JSON, форму которого владеет фронт.
type ChatBackground struct {
	ConversationID *int64
	Recipe         json.RawMessage
}

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
	KindPost     = "post"
	KindDevReply = "system_dev_reply"
	// KindSystem — служебная плашка группы (создан/добавлен/вышел/переименован).
	// sender_id = инициатор, текст — готовое описание события.
	KindSystem = "system"
)

// Роли участника группы (conversation_members.role).
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// Conversation — диалог. Пара user_a<user_b, соло-чат (dev, user_b NULL,
// владелец — user_a) либо группа (is_group, user_a/user_b NULL, участники —
// в conversation_members).
type Conversation struct {
	ID            int64
	UserAID       int64
	UserBID       *int64
	CompanyID     *int64  // NULL — переписка людей без общей компании / группа
	CompanyName   *string // подгружается листингами (JOIN companies)
	IsDevChat     bool
	CreatedAt     time.Time
	LastMessageAt *time.Time
	HiddenForA    bool
	HiddenForB    bool
	PinnedAtA     *time.Time
	PinnedAtB     *time.Time

	// Группа.
	IsGroup    bool
	Title      *string
	AvatarPath *string
	CreatedBy  *int64
	InviteCode *string
	// Проекция «для зрителя» — заполняется листингом/открытием группы из
	// его строки conversation_members (актуальны только для конкретного userID).
	MemberCount  int
	MyRole       string
	MyMuted      bool
	MyPinnedAt   *time.Time
	MyLastReadID *int64
	Members      []*Member // гидрируется по требованию (карточка группы)
}

// IsSolo — чат без «второй стороны»: dev-чат техподдержки.
func (c *Conversation) IsSolo() bool { return c.IsDevChat }

// OtherUserID — собеседник; nil для соло-чатов и групп.
func (c *Conversation) OtherUserID(userID int64) *int64 {
	if c.IsSolo() || c.IsGroup {
		return nil
	}
	if c.UserAID == userID {
		return c.UserBID
	}
	a := c.UserAID
	return &a
}

// Side — 'a', если userID == user_a_id, иначе 'b'. Для соло/групп — 'a'
// (группы прочтение/скрытие ведут в conversation_members, side им не важен).
func (c *Conversation) Side(userID int64) string {
	if c.IsSolo() || c.IsGroup || c.UserAID == userID {
		return SideA
	}
	return SideB
}

func (c *Conversation) PinnedAtFor(userID int64) *time.Time {
	if c.IsSolo() {
		return nil
	}
	if c.IsGroup {
		return c.MyPinnedAt
	}
	if c.Side(userID) == SideA {
		return c.PinnedAtA
	}
	return c.PinnedAtB
}

// Member — участник группы (conversation_members).
type Member struct {
	ConversationID    int64
	UserID            int64
	Role              string
	JoinedAt          time.Time
	LastReadMessageID *int64
	LastReadAt        *time.Time
	PinnedAt          *time.Time
	HiddenAt          *time.Time
	Muted             bool
	CanManageMembers  bool
	CanEditInfo       bool
	CanPinMessages    bool
	User              *User // гидрируется для списка участников
}

// CanManage — может ли участник выполнять управляющее действие action.
// owner — всё; admin — по своим флагам; member — ничего.
func (m *Member) CanManage(action string) bool {
	if m == nil {
		return false
	}
	if m.Role == RoleOwner {
		return true
	}
	if m.Role != RoleAdmin {
		return false
	}
	switch action {
	case "members":
		return m.CanManageMembers
	case "info":
		return m.CanEditInfo
	case "pin":
		return m.CanPinMessages
	}
	return false
}

// Attachment — файл-вложение. message_id NULL до привязки к сообщению.
type Attachment struct {
	ID         int64
	MessageID  *int64
	UploaderID int64
	FilePath   string
	// ThumbPath — уменьшенное превью картинки (nil, если не картинка или сжать
	// не удалось). Отдаётся клиенту как облегчённая версия для ленты чата.
	ThumbPath *string
	FileName  string
	MimeType  string
	SizeBytes int64
	CreatedAt time.Time
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

// PostPreview — ЗАМОРОЖЕННЫЙ снапшот пересланного поста портала (в отличие от
// TaskPreview — не живой JOIN на чужую таблицу: portalsvc передаёт
// title/excerpt/cover_url в момент пересылки через gRPC CreatePostMessage,
// мессенджер хранит их как есть на самом сообщении — так он не завязывается
// на схему portalsvc).
type PostPreview struct {
	ID       int64
	Title    string
	Excerpt  string
	CoverURL *string
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
	PostID              *int64
	PinnedAt            *time.Time
	PinnedByID          *int64
	EditedAt            *time.Time

	Attachments   []Attachment
	Reactions     []Reaction
	ReplyTo       *ReplyPreview
	ForwardedFrom *UserRef
	Call          *CallInfo
	Task          *TaskPreview
	Post          *PostPreview

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

// Reaction — эмодзи-реакция пользователя на сообщение (toggle по тройке
// message_id+user_id+emoji). Группировку по эмодзи делает клиент.
type Reaction struct {
	MessageID int64
	UserID    int64
	Emoji     string
}

// MaxReactionsPerUser — максимум разных реакций одного пользователя на одном
// сообщении.
const MaxReactionsPerUser = 2

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
	PostID              *int64
	PostTitle           *string
	PostExcerpt         *string
	PostCoverURL        *string
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
	RoleLevel     int    // из токена (active company), не из users
	CompanyID     *int64 // из токена (active company), не из users
	Phone         *string
	Email         *string
	AvatarPath    *string
	IsActive      bool
	IsSuperAdmin  bool
	CompanyActive bool
	LastSeenAt    *time.Time
	StatusEmoji   *string
	StatusText    *string
}
