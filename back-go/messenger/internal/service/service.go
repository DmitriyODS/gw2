// Package service — бизнес-логика мессенджера. Портировано из
// back/app/services/messenger_service.py (и messenger-части message_repo.py)
// без изменения правил; сюда же переехал автоответ техподдержки.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

// MaxAttachmentSize — лимит одного вложения (MESSENGER_ATTACHMENT_MAX).
const MaxAttachmentSize = 500 * 1024 * 1024

// SupportAutoReplyText — автоответ техподдержки на первое за сутки обращение.
const SupportAutoReplyText = "Здравствуйте! Спасибо за обращение! " +
	"Ваше сообщение было направлено нашим разработчикам."

// SupportAutoReplyAfter — «сутки тишины», после которых бот отвечает снова.
const SupportAutoReplyAfter = 24 * time.Hour

// SupportHumanLullPeriod — окно «человек-поддержка в диалоге»: если
// супер-админ отвечал в dev-чате свежее, ИИ-бот молчит и не влезает.
const SupportHumanLullPeriod = 15 * time.Minute

// supportHistoryLimit — сколько последних сообщений dev-чата уходит ИИ
// как контекст диалога.
const supportHistoryLimit = 16

// supportReplyTimeout — потолок фонового ИИ-ответа поддержки (LLM-вызов).
const supportReplyTimeout = 60 * time.Second

// MessengerService — все use-case'ы сервиса (REST + gRPC).
type MessengerService interface {
	ListConversations(ctx context.Context, userID int64, companyID *int64) ([]*dto.ConversationListItem, error)
	OpenConversation(ctx context.Context, meID, otherUserID int64) (*dto.ConversationWithOther, error)
	ListMessages(ctx context.Context, convID, userID int64, beforeID, afterID *int64, limit int) ([]*dto.Message, error)
	SendMessage(ctx context.Context, convID, senderID int64, req dto.MessageCreate) (*dto.Message, error)
	ForwardMessage(ctx context.Context, senderID, messageID int64, conversationIDs, userIDs []int64) ([]dto.ForwardResult, error)
	MarkRead(ctx context.Context, convID, userID int64) (int, error)
	UploadAttachment(ctx context.Context, uploaderID int64, fileName, mimeType string, data []byte) (*dto.Attachment, error)
	DeleteMessage(ctx context.Context, messageID, userID int64, scope string) (bool, error)
	EditMessage(ctx context.Context, messageID, userID int64, text string) (*dto.Message, error)
	DeleteConversation(ctx context.Context, convID, userID int64, scope string) (bool, error)
	ToggleConversationPin(ctx context.Context, convID, userID int64) (bool, error)
	ToggleMessagePin(ctx context.Context, messageID, userID int64) (*dto.Message, bool, error)
	ToggleMessageReaction(ctx context.Context, messageID, userID int64, emoji string) (*dto.Message, bool, error)
	ListPinnedMessages(ctx context.Context, convID, userID int64) ([]*dto.Message, error)
	OpenDevChat(ctx context.Context, userID int64, companyID *int64) (*dto.Conversation, error)
	SupportInbox(ctx context.Context, userID int64) ([]*dto.ConversationListItem, error)
	TotalUnread(ctx context.Context, userID int64) (int, error)

	// Оформление чатов (личное, синк между устройствами).
	GetChatBackgrounds(ctx context.Context, userID int64) (*dto.ChatBackgroundsResponse, error)
	SetChatBackground(ctx context.Context, userID int64, convID *int64, recipe json.RawMessage) error
	DeleteChatBackground(ctx context.Context, userID int64, convID *int64) error

	// Папки чатов (личная навигация, синк между устройствами).
	ListFolders(ctx context.Context, userID int64) ([]*dto.Folder, error)
	CreateFolder(ctx context.Context, userID int64, in dto.FolderInput) (*dto.Folder, error)
	UpdateFolder(ctx context.Context, userID, folderID int64, in dto.FolderInput) (*dto.Folder, error)
	DeleteFolder(ctx context.Context, userID, folderID int64) error
	ReorderFolders(ctx context.Context, userID int64, orderedIDs []int64) error
	AddFolderItem(ctx context.Context, userID, folderID, convID int64) error
	RemoveFolderItem(ctx context.Context, userID, folderID, convID int64) error

	// Группы.
	CreateGroup(ctx context.Context, creatorID int64, title string, avatarAttID *int64, memberIDs []int64) (*dto.Conversation, error)
	GetGroup(ctx context.Context, convID, userID int64) (*dto.Conversation, error)
	AddGroupMembers(ctx context.Context, convID, actorID int64, userIDs []int64) error
	RemoveGroupMember(ctx context.Context, convID, actorID, memberID int64) error
	LeaveGroup(ctx context.Context, convID, userID int64) error
	RenameGroup(ctx context.Context, convID, actorID int64, title string) error
	SetGroupAvatar(ctx context.Context, convID, actorID int64, avatarAttID *int64) error
	SetMemberRole(ctx context.Context, convID, actorID, memberID int64, role string) error
	SetMemberRights(ctx context.Context, convID, actorID, memberID int64, manageMembers, editInfo, pinMessages bool) error
	TransferOwnership(ctx context.Context, convID, actorID, newOwnerID int64) error
	SetGroupMute(ctx context.Context, convID, userID int64, muted bool) (bool, error)
	GroupInviteLink(ctx context.Context, convID, actorID int64) (string, error)
	RevokeGroupInviteLink(ctx context.Context, convID, actorID int64) error
	GroupInvitePreview(ctx context.Context, code string) (*dto.Conversation, error)
	JoinGroupByCode(ctx context.Context, code string, userID int64) (*dto.Conversation, error)
	ReadBy(ctx context.Context, messageID, userID int64) ([]*dto.DirectoryUser, error)

	// gRPC (callsvc, portalsvc).
	EnsureDialog(ctx context.Context, userAID, userBID int64) (int64, error)
	CreateCallMessage(ctx context.Context, convID, senderID, callID int64) (*dto.Message, []int64, error)
	GetCallMessage(ctx context.Context, callID int64) (int64, *dto.Message, []int64, error)
	CreatePostMessage(ctx context.Context, convID, senderID, postID int64, title, excerpt, coverURL string) (*dto.Message, []int64, error)
}

type Service struct {
	repo  domain.Repository
	users domain.UserReader
	files domain.FileStore
	pub   domain.EventPublisher
	// ai — ИИ техподдержки dev-чата (gRPC aisvc); nil — только канированный
	// автоответ. Любая ошибка ИИ тоже откатывается на него (fail-open).
	ai  domain.SupportAI
	log *slog.Logger
}

var _ MessengerService = (*Service)(nil)

func New(repo domain.Repository, users domain.UserReader, files domain.FileStore,
	pub domain.EventPublisher, ai domain.SupportAI, log *slog.Logger) *Service {
	return &Service{repo: repo, users: users, files: files, pub: pub, ai: ai, log: log}
}

// ── Общие ошибки ─────────────────────────────────────────────────

func errConvNotFound() *domain.Error {
	return domain.NewError("CONV_NOT_FOUND", "Диалог не найден", 404)
}

func errMsgNotFound() *domain.Error {
	return domain.NewError("MSG_NOT_FOUND", "Сообщение не найдено", 404)
}

func errNoAccess() *domain.Error {
	return domain.NewError("FORBIDDEN", "Нет доступа к диалогу", 403)
}

// ── Общие хелперы ────────────────────────────────────────────────

func room(userID int64) string { return fmt.Sprintf("user_%d", userID) }

func rooms(ids ...int64) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, room(id))
	}
	return out
}

// audience — комнаты-получатели событий диалога (все, кто его видит):
// группа — все участники, dev-чат — владелец+супер-админы, пара — обе стороны.
func (s *Service) audience(ctx context.Context, conv *domain.Conversation) ([]int64, error) {
	if conv.IsGroup {
		mutes, err := s.repo.ListMemberMutes(ctx, conv.ID)
		if err != nil {
			return nil, err
		}
		ids := make([]int64, 0, len(mutes))
		for id := range mutes {
			ids = append(ids, id)
		}
		return ids, nil
	}
	if conv.IsDevChat {
		return s.users.DevChatUserIDs(ctx, conv.UserAID)
	}
	ids := []int64{conv.UserAID}
	if conv.UserBID != nil {
		ids = append(ids, *conv.UserBID)
	}
	return ids, nil
}

// ensureMember — доступ к диалогу: p2p — только участники; dev-чат —
// владелец (user_a) + любой супер-админ; группа — участник conversation_members.
func (s *Service) ensureMember(ctx context.Context, conv *domain.Conversation, userID int64) error {
	if conv.IsGroup {
		m, err := s.repo.GetMember(ctx, conv.ID, userID)
		if err != nil {
			return err
		}
		if m == nil {
			return errNoAccess()
		}
		return nil
	}
	if conv.IsDevChat {
		if conv.UserAID == userID {
			return nil // владелец
		}
		user, err := s.users.GetUser(ctx, userID)
		if err != nil {
			return err
		}
		if user == nil {
			return errNoAccess()
		}
		if user.IsSuperAdmin {
			return nil // супер-админ (техподдержка)
		}
		return errNoAccess()
	}
	if conv.UserAID == userID || (conv.UserBID != nil && *conv.UserBID == userID) {
		return nil
	}
	return errNoAccess()
}

// conversationForUser — диалог + проверка доступа (get_conversation_for_user).
func (s *Service) conversationForUser(ctx context.Context, convID, userID int64) (*domain.Conversation, error) {
	conv, err := s.repo.GetConversation(ctx, convID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, errConvNotFound()
	}
	if err := s.ensureMember(ctx, conv, userID); err != nil {
		return nil, err
	}
	return conv, nil
}

// ensureConversation — найти/создать парный диалог с проверками
// (_ensure_conversation во Flask: self/hidden/multi-tenancy).
func (s *Service) ensureConversation(ctx context.Context, currentUserID, otherUserID int64) (*domain.Conversation, error) {
	if currentUserID == otherUserID {
		return nil, domain.NewError("SELF_CONVERSATION", "Нельзя написать самому себе", 400)
	}
	me, err := s.users.GetUser(ctx, currentUserID)
	if err != nil {
		return nil, err
	}
	other, err := s.users.GetUser(ctx, otherUserID)
	if err != nil {
		return nil, err
	}
	if other == nil || !other.IsActive {
		return nil, domain.NewError("USER_NOT_FOUND", "Собеседник не найден", 404)
	}
	// Переписка между сотрудниками разных компаний (и людьми без общей
	// компании) разрешена: company-барьер для чата снят.

	a, b := currentUserID, otherUserID
	lower, higher := me, other
	if a > b {
		a, b = b, a
		lower, higher = other, me
	}
	conv, err := s.repo.GetPair(ctx, a, b)
	if err != nil || conv != nil {
		return conv, err
	}
	// company_id диалога теперь опционален: берём активную компанию любого из
	// участников (сначала меньший id), nil — если общей/любой компании нет.
	var companyID *int64
	switch {
	case lower != nil && lower.CompanyID != nil:
		companyID = lower.CompanyID
	case higher != nil && higher.CompanyID != nil:
		companyID = higher.CompanyID
	}
	return s.repo.CreatePair(ctx, a, b, companyID)
}

// notifyIDs — оба участника парного диалога (для gRPC notify_user_ids).
func pairNotifyIDs(conv *domain.Conversation) []int64 {
	out := []int64{conv.UserAID}
	if conv.UserBID != nil {
		out = append(out, *conv.UserBID)
	}
	return out
}
