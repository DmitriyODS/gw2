// Package endpoint — go-kit endpoints поверх сервисного слоя.
//
// Каждый use-case обёрнут в endpoint.Endpoint: транспорты (gRPC, Fiber)
// декодируют свои запросы, зовут endpoint и кодируют ответ обратно.
// Бизнес-ошибки (*domain.Error) пролетают через error-канал endpoint'а и
// мапятся транспортом: gRPC — в поле Error ответа, HTTP — в статус + JSON.
package endpoint

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/service"
)

// Endpoints — все use-case'ы сервиса мессенджера.
type Endpoints struct {
	ListConversations     endpoint.Endpoint
	OpenConversation      endpoint.Endpoint
	ListMessages          endpoint.Endpoint
	SendMessage           endpoint.Endpoint
	ForwardMessage        endpoint.Endpoint
	MarkRead              endpoint.Endpoint
	UploadAttachment      endpoint.Endpoint
	DeleteMessage         endpoint.Endpoint
	EditMessage           endpoint.Endpoint
	DeleteConversation    endpoint.Endpoint
	ToggleConversationPin endpoint.Endpoint
	ToggleMessagePin      endpoint.Endpoint
	ToggleMessageReaction endpoint.Endpoint
	ListPinnedMessages    endpoint.Endpoint
	OpenDevChat           endpoint.Endpoint
	SupportInbox          endpoint.Endpoint
	TotalUnread           endpoint.Endpoint

	EnsureDialog      endpoint.Endpoint
	CreateCallMessage endpoint.Endpoint
	GetCallMessage    endpoint.Endpoint
	CreatePostMessage endpoint.Endpoint
}

// ── Транспорт-независимые запросы/ответы ─────────────────────────

type ListMessagesRequest struct {
	ConversationID int64
	UserID         int64
	BeforeID       *int64
	AfterID        *int64
	Limit          int
}

type SendMessageRequest struct {
	ConversationID int64
	SenderID       int64
	Body           dto.MessageCreate
}

type ForwardRequest struct {
	SenderID        int64
	MessageID       int64
	ConversationIDs []int64
	UserIDs         []int64
}

type ConvUserRequest struct {
	ConversationID int64
	UserID         int64
}

type MsgUserRequest struct {
	MessageID int64
	UserID    int64
}

type ScopedDeleteRequest struct {
	ID     int64 // message_id или conversation_id
	UserID int64
	Scope  string
}

type EditMessageRequest struct {
	MessageID int64
	UserID    int64
	Text      string
}

// SoloChatRequest — открыть/создать соло-чат (pet/dev). CompanyID — активная
// компания из токена (в users её нет: идентичность развязана с компаниями).
type SoloChatRequest struct {
	UserID    int64
	CompanyID *int64
}

// ListConversationsRequest — список диалогов. CompanyID — активная компания
// сессии из токена: нужна, чтобы автосоздать личный чат техподдержки члена
// компании (в users активной компании нет).
type ListConversationsRequest struct {
	UserID    int64
	CompanyID *int64
}

type UploadRequest struct {
	UploaderID int64
	FileName   string
	MimeType   string
	Data       []byte
}

type OpenConversationRequest struct {
	MeID        int64
	OtherUserID int64
}

type PairRequest struct {
	UserAID int64
	UserBID int64
}

type CreateCallMessageRequest struct {
	ConversationID int64
	SenderID       int64
	CallID         int64
}

type CreatePostMessageRequest struct {
	ConversationID int64
	SenderID       int64
	PostID         int64
	Title          string
	Excerpt        string
	CoverURL       string
}

type MessagePinResponse struct {
	Message *dto.Message
	Pinned  bool
}

type ReactionRequest struct {
	MessageID int64
	UserID    int64
	Emoji     string
}

type ReactionResponse struct {
	Message *dto.Message
	Added   bool
}

// CallMessageResponse — снапшот системного сообщения + адресаты message:new.
// Форма общая для плашек звонка (CreateCallMessage/GetCallMessage) и
// пересланного поста (CreatePostMessage) — оба возвращают одно и то же.
type CallMessageResponse struct {
	ConversationID int64
	Message        *dto.Message
	NotifyUserIDs  []int64
}

func New(svc service.MessengerService) Endpoints {
	return Endpoints{
		ListConversations: func(ctx context.Context, request any) (any, error) {
			req := request.(ListConversationsRequest)
			return svc.ListConversations(ctx, req.UserID, req.CompanyID)
		},
		OpenConversation: func(ctx context.Context, request any) (any, error) {
			req := request.(OpenConversationRequest)
			return svc.OpenConversation(ctx, req.MeID, req.OtherUserID)
		},
		ListMessages: func(ctx context.Context, request any) (any, error) {
			req := request.(ListMessagesRequest)
			return svc.ListMessages(ctx, req.ConversationID, req.UserID, req.BeforeID, req.AfterID, req.Limit)
		},
		SendMessage: func(ctx context.Context, request any) (any, error) {
			req := request.(SendMessageRequest)
			return svc.SendMessage(ctx, req.ConversationID, req.SenderID, req.Body)
		},
		ForwardMessage: func(ctx context.Context, request any) (any, error) {
			req := request.(ForwardRequest)
			return svc.ForwardMessage(ctx, req.SenderID, req.MessageID, req.ConversationIDs, req.UserIDs)
		},
		MarkRead: func(ctx context.Context, request any) (any, error) {
			req := request.(ConvUserRequest)
			return svc.MarkRead(ctx, req.ConversationID, req.UserID)
		},
		UploadAttachment: func(ctx context.Context, request any) (any, error) {
			req := request.(UploadRequest)
			return svc.UploadAttachment(ctx, req.UploaderID, req.FileName, req.MimeType, req.Data)
		},
		DeleteMessage: func(ctx context.Context, request any) (any, error) {
			req := request.(ScopedDeleteRequest)
			return svc.DeleteMessage(ctx, req.ID, req.UserID, req.Scope)
		},
		EditMessage: func(ctx context.Context, request any) (any, error) {
			req := request.(EditMessageRequest)
			return svc.EditMessage(ctx, req.MessageID, req.UserID, req.Text)
		},
		DeleteConversation: func(ctx context.Context, request any) (any, error) {
			req := request.(ScopedDeleteRequest)
			return svc.DeleteConversation(ctx, req.ID, req.UserID, req.Scope)
		},
		ToggleConversationPin: func(ctx context.Context, request any) (any, error) {
			req := request.(ConvUserRequest)
			return svc.ToggleConversationPin(ctx, req.ConversationID, req.UserID)
		},
		ToggleMessagePin: func(ctx context.Context, request any) (any, error) {
			req := request.(MsgUserRequest)
			msg, pinned, err := svc.ToggleMessagePin(ctx, req.MessageID, req.UserID)
			if err != nil {
				return nil, err
			}
			return MessagePinResponse{Message: msg, Pinned: pinned}, nil
		},
		ToggleMessageReaction: func(ctx context.Context, request any) (any, error) {
			req := request.(ReactionRequest)
			msg, added, err := svc.ToggleMessageReaction(ctx, req.MessageID, req.UserID, req.Emoji)
			if err != nil {
				return nil, err
			}
			return ReactionResponse{Message: msg, Added: added}, nil
		},
		ListPinnedMessages: func(ctx context.Context, request any) (any, error) {
			req := request.(ConvUserRequest)
			return svc.ListPinnedMessages(ctx, req.ConversationID, req.UserID)
		},
		OpenDevChat: func(ctx context.Context, request any) (any, error) {
			req := request.(SoloChatRequest)
			return svc.OpenDevChat(ctx, req.UserID, req.CompanyID)
		},
		SupportInbox: func(ctx context.Context, request any) (any, error) {
			return svc.SupportInbox(ctx, request.(int64))
		},
		TotalUnread: func(ctx context.Context, request any) (any, error) {
			return svc.TotalUnread(ctx, request.(int64))
		},

		EnsureDialog: func(ctx context.Context, request any) (any, error) {
			req := request.(PairRequest)
			return svc.EnsureDialog(ctx, req.UserAID, req.UserBID)
		},
		CreateCallMessage: func(ctx context.Context, request any) (any, error) {
			req := request.(CreateCallMessageRequest)
			msg, notify, err := svc.CreateCallMessage(ctx, req.ConversationID, req.SenderID, req.CallID)
			if err != nil {
				return nil, err
			}
			return CallMessageResponse{Message: msg, NotifyUserIDs: notify}, nil
		},
		GetCallMessage: func(ctx context.Context, request any) (any, error) {
			convID, msg, notify, err := svc.GetCallMessage(ctx, request.(int64))
			if err != nil {
				return nil, err
			}
			return CallMessageResponse{ConversationID: convID, Message: msg, NotifyUserIDs: notify}, nil
		},
		CreatePostMessage: func(ctx context.Context, request any) (any, error) {
			req := request.(CreatePostMessageRequest)
			msg, notify, err := svc.CreatePostMessage(ctx, req.ConversationID, req.SenderID, req.PostID,
				req.Title, req.Excerpt, req.CoverURL)
			if err != nil {
				return nil, err
			}
			return CallMessageResponse{ConversationID: req.ConversationID, Message: msg, NotifyUserIDs: notify}, nil
		},
	}
}
