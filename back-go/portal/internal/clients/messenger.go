// Package clients — gRPC-клиенты portalsvc к другим микросервисам.
// Межсервисное общение — только gRPC.
package clients

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/messengerpb"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

// Messenger — клиент msgsvc: пересылка поста как плашки kind='post' в
// диалоге. Бизнес-ошибка приходит полем error в ответе (транспорт всегда OK).
type Messenger struct {
	conn *grpc.ClientConn
	stub messengerpb.MessengerServiceClient
	log  *slog.Logger
}

var _ domain.MessengerClient = (*Messenger)(nil)

func NewMessenger(addr string, log *slog.Logger) (*Messenger, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Messenger{conn: conn, stub: messengerpb.NewMessengerServiceClient(conn), log: log}, nil
}

func (c *Messenger) Close() { _ = c.conn.Close() }

func bizErr(code, message string) error {
	return fmt.Errorf("%s: %s", code, message)
}

// EnsureDialog — найти/создать парный диалог (пересылка по user_ids, когда
// диалога ещё нет).
func (c *Messenger) EnsureDialog(ctx context.Context, userAID, userBID int64) (int64, error) {
	resp, err := c.stub.EnsureDialog(ctx, &messengerpb.EnsureDialogRequest{
		UserAId: userAID, UserBId: userBID,
	})
	if err != nil {
		return 0, err
	}
	if resp.GetError() != nil {
		return 0, bizErr(resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetConversationId(), nil
}

// CreatePostMessage — системная плашка пересланного поста в диалоге.
// Возвращает готовый JSON-снапшот сообщения (форма REST msgsvc) и адресатов
// message:new.
func (c *Messenger) CreatePostMessage(ctx context.Context, conversationID, senderID, postID int64, preview domain.PostPreview) (string, []int64, error) {
	resp, err := c.stub.CreatePostMessage(ctx, &messengerpb.CreatePostMessageRequest{
		ConversationId: conversationID, SenderId: senderID, PostId: postID,
		Title: preview.Title, Excerpt: preview.Excerpt, CoverUrl: preview.CoverURL,
	})
	if err != nil {
		return "", nil, err
	}
	if resp.GetError() != nil {
		return "", nil, bizErr(resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetMessageJson(), resp.GetNotifyUserIds(), nil
}
