// Package clients — gRPC-клиенты msgsvc к другим микросервисам.
package clients

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/groovepb"
)

const grooveHookTimeout = 10 * time.Second

// Groove — уведомления groovesvc о pet-чате. Fire-and-forget: вызов уходит
// в фоне со своим дедлайном, ошибки только логируются — отправка сообщения
// хозяина никогда не страдает из-за недоступности геймификации.
type Groove struct {
	conn *grpc.ClientConn
	stub groovepb.GrooveServiceClient
	log  *slog.Logger
}

var _ domain.GrooveNotifier = (*Groove)(nil)

func NewGroove(addr string, log *slog.Logger) (*Groove, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Groove{conn: conn, stub: groovepb.NewGrooveServiceClient(conn), log: log}, nil
}

func (c *Groove) Close() { _ = c.conn.Close() }

func (c *Groove) OnPetMessage(conversationID int64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), grooveHookTimeout)
		defer cancel()
		resp, err := c.stub.OnPetMessage(ctx, &groovepb.PetMessageRequest{
			ConversationId: conversationID,
		})
		if err != nil {
			c.log.Warn("groove_grpc.pet_message_failed",
				"conversation_id", conversationID, "error", err)
			return
		}
		if e := resp.GetError(); e != nil {
			c.log.Warn("groove_grpc.pet_message_rejected",
				"conversation_id", conversationID, "code", e.Code, "message", e.Message)
		}
	}()
}
