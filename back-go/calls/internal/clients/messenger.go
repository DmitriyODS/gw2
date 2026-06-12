// Package clients — gRPC-клиенты callsvc к другим микросервисам.
// Межсервисное общение — только gRPC.
package clients

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/calls/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/messengerpb"
)

// Messenger — клиент msgsvc: парный диалог (синхронно, до создания звонка)
// и плашка звонка kind='call' (для events-паблишера). Бизнес-ошибка приходит
// полем error в ответе (транспорт всегда OK) — конвертируется в обычную
// ошибку: для звонка диалог и плашка вторичны, вызывающие стороны их
// логируют и продолжают.
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

// CreateCallMessage — системная плашка звонка. Возвращает готовый JSON-снапшот
// сообщения (форма REST msgsvc) и адресатов message:new.
func (c *Messenger) CreateCallMessage(ctx context.Context, conversationID, senderID, callID int64) (string, []int64, error) {
	resp, err := c.stub.CreateCallMessage(ctx, &messengerpb.CreateCallMessageRequest{
		ConversationId: conversationID, SenderId: senderID, CallId: callID,
	})
	if err != nil {
		return "", nil, err
	}
	if resp.GetError() != nil {
		return "", nil, bizErr(resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetMessageJson(), resp.GetNotifyUserIds(), nil
}

// GetCallMessage — актуальный снапшот плашки (для message:updated).
// Пустой message_json без ошибки — плашки нет (group-звонок).
func (c *Messenger) GetCallMessage(ctx context.Context, callID int64) (int64, string, []int64, error) {
	resp, err := c.stub.GetCallMessage(ctx, &messengerpb.GetCallMessageRequest{CallId: callID})
	if err != nil {
		return 0, "", nil, err
	}
	if resp.GetError() != nil {
		return 0, "", nil, bizErr(resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return resp.GetConversationId(), resp.GetMessageJson(), resp.GetNotifyUserIds(), nil
}
