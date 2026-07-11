// Package clients — исходящие gRPC-клиенты msgsvc.
package clients

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/aipb"
)

// SupportAI — клиент aisvc.SupportChat (ИИ техподдержки dev-чата). Fail-open
// на стороне вызывающего: любая ошибка (сервис недоступен, ключ не настроен)
// откатывает поддержку на канированный автоответ.
type SupportAI struct {
	conn *grpc.ClientConn
	stub aipb.AiServiceClient
	log  *slog.Logger
}

var _ domain.SupportAI = (*SupportAI)(nil)

func NewSupportAI(addr string, log *slog.Logger) (*SupportAI, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &SupportAI{conn: conn, stub: aipb.NewAiServiceClient(conn), log: log}, nil
}

func (c *SupportAI) Close() { _ = c.conn.Close() }

func (c *SupportAI) SupportReply(ctx context.Context, messagesJSON string) (string, error) {
	resp, err := c.stub.SupportChat(ctx, &aipb.SupportChatRequest{MessagesJson: messagesJSON})
	if err != nil {
		return "", err
	}
	if e := resp.GetError(); e != nil {
		return "", domain.NewError(e.GetCode(), e.GetMessage(), int(e.GetHttpStatus()))
	}
	return resp.GetContent(), nil
}
