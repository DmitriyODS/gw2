package clients

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/messengerpb"
)

const messengerTimeout = 10 * time.Second

// Messenger — клиент msgsvc для pet-чата: история диалога как контекст
// AI-ответа и публикация бот-сообщения (msgsvc сам эмитит message:new).
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
	return &Messenger{
		conn: conn,
		stub: messengerpb.NewMessengerServiceClient(conn),
		log:  log,
	}, nil
}

func (c *Messenger) Close() { _ = c.conn.Close() }

func (c *Messenger) PostBotMessage(ctx context.Context, conversationID int64, text string) error {
	rctx, cancel := context.WithTimeout(ctx, messengerTimeout)
	defer cancel()
	resp, err := c.stub.PostBotMessage(rctx, &messengerpb.PostBotMessageRequest{
		ConversationId: conversationID,
		Text:           text,
	})
	if err != nil {
		return err
	}
	if e := resp.GetError(); e != nil {
		return domain.NewError(e.Code, e.Message, int(e.HttpStatus))
	}
	return nil
}

func (c *Messenger) ListRecentMessages(ctx context.Context, conversationID int64,
	limit int) ([]domain.ChatMessage, error) {

	rctx, cancel := context.WithTimeout(ctx, messengerTimeout)
	defer cancel()
	resp, err := c.stub.ListRecentMessages(rctx, &messengerpb.ListRecentMessagesRequest{
		ConversationId: conversationID,
		Limit:          int32(limit),
	})
	if err != nil {
		return nil, err
	}
	if e := resp.GetError(); e != nil {
		return nil, domain.NewError(e.Code, e.Message, int(e.HttpStatus))
	}
	out := make([]domain.ChatMessage, 0, len(resp.GetMessages()))
	for _, m := range resp.GetMessages() {
		out = append(out, domain.ChatMessage{IsBot: m.GetIsBot(), Text: m.GetText()})
	}
	return out, nil
}
