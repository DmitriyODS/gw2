package domain

import (
	"context"
	"time"
)

// Repository — персистентность мессенджера (PostgreSQL, общая БД платформы)
// + read-only лукапы смежных таблиц (pets, tasks, calls).
type Repository interface {
	// RunInTx — выполнить fn в одной транзакции (forward: диалоги и
	// сообщения создаются атомарно, иначе при ошибке остаётся пустой диалог).
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error

	// ── Диалоги ──────────────────────────────────────────────────
	GetConversation(ctx context.Context, id int64) (*Conversation, error)
	// GetPair — пара уже нормализована вызывающим (a < b).
	GetPair(ctx context.Context, a, b int64) (*Conversation, error)
	// CreatePair — INSERT пары; при гонке по уникальному индексу возвращает
	// существующий диалог. companyID может быть nil (нет общей компании).
	CreatePair(ctx context.Context, a, b int64, companyID *int64) (*Conversation, error)
	// GetSolo / CreateSolo — личный dev-чат владельца (техподдержка).
	GetSolo(ctx context.Context, userID int64) (*Conversation, error)
	CreateSolo(ctx context.Context, userID, companyID int64) (*Conversation, error)
	// ListPairConversations — не скрытые на стороне userID, с именем
	// компании, порядок: last_message_at DESC NULLS LAST, created_at DESC.
	ListPairConversations(ctx context.Context, userID int64) ([]*Conversation, error)
	// ListDevChats — все dev-чаты (support-inbox), с именем компании,
	// порядок как у ListPairConversations.
	ListDevChats(ctx context.Context) ([]*Conversation, error)
	// HideConversation — скрыть диалог и все его сообщения на стороне side;
	// true — теперь скрыт обеими сторонами (вызывающий удаляет физически).
	HideConversation(ctx context.Context, id int64, side string) (bool, error)
	DeleteConversation(ctx context.Context, id int64) error
	SetConversationPin(ctx context.Context, id int64, side string, pinned bool) error

	// ── Группы ───────────────────────────────────────────────────
	// CreateGroup — INSERT диалога is_group + участников (creatorID — owner,
	// остальные — member). memberIDs включает создателя (дедуп внутри).
	CreateGroup(ctx context.Context, title string, avatarPath *string, creatorID int64, memberIDs []int64) (*Conversation, error)
	// ListGroupConversations — группы, где userID участник и не скрыл
	// (hidden_at IS NULL), с проекцией зрителя (MyRole/MyMuted/MyPinnedAt/
	// MyLastReadID/MemberCount) и именем компании.
	ListGroupConversations(ctx context.Context, userID int64) ([]*Conversation, error)
	RenameGroup(ctx context.Context, convID int64, title string) error
	SetGroupAvatar(ctx context.Context, convID int64, avatarPath *string) error
	SetInviteCode(ctx context.Context, convID int64, code *string) error
	FindByInviteCode(ctx context.Context, code string) (*Conversation, error)

	// ── Участники группы ─────────────────────────────────────────
	AddMember(ctx context.Context, convID, userID int64, role string) error
	RemoveMember(ctx context.Context, convID, userID int64) error
	GetMember(ctx context.Context, convID, userID int64) (*Member, error)
	ListMembers(ctx context.Context, convID int64) ([]*Member, error)
	// ListMemberMutes — user_id → muted всех участников (для веера/пуша).
	ListMemberMutes(ctx context.Context, convID int64) (map[int64]bool, error)
	UpdateMemberRole(ctx context.Context, convID, userID int64, role string) error
	UpdateMemberRights(ctx context.Context, convID, userID int64, manageMembers, editInfo, pinMessages bool) error
	SetMemberMute(ctx context.Context, convID, userID int64, muted bool) error
	SetMemberPin(ctx context.Context, convID, userID int64, pinned bool) error
	HideConversationMember(ctx context.Context, convID, userID int64, hidden bool) error
	// SetMemberRead — поднять watermark участника до lastMessageID (только вверх);
	// true — значение изменилось (есть что рассылать).
	SetMemberRead(ctx context.Context, convID, userID, lastMessageID int64) (bool, error)
	// ReadersOf — участники, прочитавшие сообщение (last_read >= messageID),
	// кроме автора; гидрированы User (для панели «кто прочитал»).
	ReadersOf(ctx context.Context, convID, messageID, authorID int64) ([]*Member, error)
	// CountGroupUnread — непрочитанные по группам: id > last_read И sender != userID.
	CountGroupUnread(ctx context.Context, convIDs []int64, userID int64) (map[int64]int, error)
	// TotalGroupUnread — суммарно по всем не скрытым группам пользователя.
	TotalGroupUnread(ctx context.Context, userID int64) (int, error)

	// ── Сообщения ────────────────────────────────────────────────
	// GetMessage — полный снапшот (вложения, цитата, плашки call/task,
	// контекст dev-чата); nil — нет такого.
	GetMessage(ctx context.Context, id int64) (*Message, error)
	// ListMessages — сообщения диалога без скрытых на стороне side,
	// старые → новые. beforeID — пагинация в историю, afterID — только новые.
	ListMessages(ctx context.Context, convID int64, side string, beforeID, afterID *int64, limit int) ([]*Message, error)
	// ListPinned — закреплённые, не скрытые на стороне side, свежие первыми.
	ListPinned(ctx context.Context, convID int64, side string) ([]*Message, error)
	// LastVisibleMessages — последнее сообщение каждого диалога, не скрытое
	// на стороне side (side "" — без фильтра, для соло-чатов и inbox).
	LastVisibleMessages(ctx context.Context, convIDs []int64, side string) (map[int64]*Message, error)
	// CountUnread — непрочитанные по диалогам: (sender IS NULL OR sender !=
	// userID) AND read_at IS NULL — явный OR IS NULL, иначе трёхзначная
	// логика SQL молча теряет бот-сообщения. side "" — без фильтра скрытых.
	CountUnread(ctx context.Context, convIDs []int64, userID int64, side string) (map[int64]int, error)
	// CountUnreadFromSenders — непрочитанные от конкретных авторов
	// (support-inbox: только сообщения владельцев dev-чатов).
	CountUnreadFromSenders(ctx context.Context, convIDs, senderIDs []int64) (map[int64]int, error)
	// TotalUnread — по всем не скрытым диалогам пользователя, без скрытых
	// на его стороне сообщений.
	TotalUnread(ctx context.Context, userID int64) (int, error)
	// CreateMessage — INSERT + привязка вложений (uploader = sender,
	// message_id IS NULL) + у диалога last_message_at и сброс hidden_for_*.
	// Возвращает полный снапшот.
	CreateMessage(ctx context.Context, m NewMessage) (*Message, error)
	// MarkRead — read_at для всех входящих; количество обновлённых.
	MarkRead(ctx context.Context, convID, readerID int64) (int, error)
	// HideMessage — true, если сообщение теперь скрыто обеими сторонами.
	HideMessage(ctx context.Context, id int64, side string) (bool, error)
	DeleteMessage(ctx context.Context, id int64) error
	RecomputeLastMessageAt(ctx context.Context, convID int64) error
	SetMessagePin(ctx context.Context, id int64, pinned bool, byID *int64) error
	// UpdateMessageText — новый текст и edited_at = now() (редактирование).
	UpdateMessageText(ctx context.Context, id int64, text string) error
	// ToggleReaction — поставить/снять реакцию; true — реакция теперь стоит.
	ToggleReaction(ctx context.Context, messageID, userID int64, emoji string) (bool, error)
	// HasHumanMessageSince — было ли сообщение НЕ бота свежее since и старше
	// beforeID (нужен ли автоответ техподдержки).
	HasHumanMessageSince(ctx context.Context, convID int64, since time.Time, beforeID int64) (bool, error)
	// HasSupportHumanReplySince — отвечал ли ЧЕЛОВЕК поддержки (kind='dev_reply',
	// не бот) свежее since: пока разработчик в диалоге, ИИ-бот молчит.
	HasSupportHumanReplySince(ctx context.Context, convID int64, since time.Time) (bool, error)
	// FindCallMessage — свежайшая плашка kind='call' звонка в его диалоге
	// (фильтр по диалогу обязателен: пересланные плашки живут в чужих).
	FindCallMessage(ctx context.Context, callID, convID int64) (*Message, error)
	ListAttachmentPathsOfConversation(ctx context.Context, convID int64) ([]string, error)

	// ── Вложения ─────────────────────────────────────────────────
	CreateAttachment(ctx context.Context, att *Attachment) error
	GetAttachment(ctx context.Context, id int64) (*Attachment, error)

	// ── Оформление чатов ─────────────────────────────────────────
	// ListChatBackgrounds — все рецепты пользователя (дефолт + по чатам).
	ListChatBackgrounds(ctx context.Context, userID int64) ([]*ChatBackground, error)
	// UpsertChatBackground — сохранить рецепт (convID nil — общий дефолт).
	UpsertChatBackground(ctx context.Context, userID int64, convID *int64, recipe []byte) error
	// DeleteChatBackground — снять рецепт (convID nil — общий дефолт).
	DeleteChatBackground(ctx context.Context, userID int64, convID *int64) error

	// ── Папки чатов ──────────────────────────────────────────────
	// ListFolders — папки владельца (по position), с ручными привязками
	// (ConversationIDs), без N+1.
	ListFolders(ctx context.Context, ownerID int64) ([]*Folder, error)
	CountFolders(ctx context.Context, ownerID int64) (int, error)
	// CreateFolder — создаёт папку (position = в конец) и возвращает её id.
	CreateFolder(ctx context.Context, f *Folder) (int64, error)
	// UpdateFolder — обновляет поля папки владельца (title/emoji/флаги).
	UpdateFolder(ctx context.Context, ownerID int64, f *Folder) error
	DeleteFolder(ctx context.Context, ownerID, folderID int64) error
	// ReorderFolders — проставляет position по порядку orderedIDs (только свои).
	ReorderFolders(ctx context.Context, ownerID int64, orderedIDs []int64) error
	// SetFolderItems — полная замена ручных привязок папки.
	SetFolderItems(ctx context.Context, ownerID, folderID int64, convIDs []int64) error
	AddFolderItem(ctx context.Context, ownerID, folderID, convID int64) error
	RemoveFolderItem(ctx context.Context, ownerID, folderID, convID int64) error

	// ── Read-only лукапы чужих таблиц ────────────────────────────
	GetCall(ctx context.Context, id int64) (*CallInfo, error)
	GetTask(ctx context.Context, id int64) (*TaskPreview, error)
}

// UserReader — read-only доступ к пользователям платформы.
type UserReader interface {
	GetUser(ctx context.Context, id int64) (*User, error)
	// CompanyActive — активна ли выбранная (активная) компания сессии из
	// токена. nil (активной компании нет) → true.
	CompanyActive(ctx context.Context, companyID *int64) (bool, error)
	// ListUsers — профили по id (включая неактивных — как «собеседники»
	// в листинге).
	ListUsers(ctx context.Context, ids []int64) ([]*User, error)
	// DevChatUserIDs — адресаты событий dev-чата: владелец + все активные
	// супер-админы (техподдержка).
	DevChatUserIDs(ctx context.Context, ownerID int64) ([]int64, error)
}

// SupportAI — ИИ техподдержки dev-чата (gRPC aisvc SupportChat). messagesJSON —
// история диалога [{role, content}] (владелец — user, поддержка — assistant),
// системный промпт добавляет aisvc. Ключ не настроен / сбой → ошибка,
// вызывающий откатывается на канированный автоответ.
type SupportAI interface {
	SupportReply(ctx context.Context, messagesJSON string) (string, error)
}

// FileStore — файлы вложений в общем uploads-каталоге (наружу отдаёт nginx
// по /uploads/, в dev — Flask).
type FileStore interface {
	// Save — сохранить под messages/YYYY/MM/{uuid32hex}{ext}; возвращает
	// относительный путь (file_path).
	Save(data []byte, ext string) (string, error)
	// Copy — физическая копия файла (пересылка): удаление одной копии не
	// задевает другую. Возвращает путь новой копии.
	Copy(srcRelPath string) (string, error)
	// Remove — best-effort удаление; ошибки — только warn-лог.
	Remove(paths []string)
}

// EventPublisher — доставка событий Socket.IO через Flask-мост
// (Redis-канал gw2:messenger:events). Потеря события не фатальна.
type EventPublisher interface {
	// Publish — {"event": ..., "rooms": ["user_12", ...], "payload": {...}}.
	// События с префиксом "_" — служебные хуки моста, наружу не эмитятся.
	Publish(ctx context.Context, event string, rooms []string, payload any)
}
