// Package clients — исходящие gRPC-клиенты notesvc.
package clients

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/aipb"
)

// Embedder — клиент aisvc.Embed (векторизация текста для ИИ-поиска заметок).
// Fail-open на стороне вызывающего: любая ошибка (сервис недоступен, у компании
// не включён AI) откатывает поиск на текстовый.
type Embedder struct {
	conn *grpc.ClientConn
	stub aipb.AiServiceClient
	log  *slog.Logger
}

var _ domain.Embedder = (*Embedder)(nil)

func NewEmbedder(addr string, log *slog.Logger) (*Embedder, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Embedder{conn: conn, stub: aipb.NewAiServiceClient(conn), log: log}, nil
}

func (c *Embedder) Close() { _ = c.conn.Close() }

func (c *Embedder) Enabled() bool { return c != nil }

func (c *Embedder) Embed(ctx context.Context, companyID int64, text string) ([]float32, string, error) {
	resp, err := c.stub.Embed(ctx, &aipb.EmbedRequest{CompanyId: companyID, Text: text})
	if err != nil {
		return nil, "", err
	}
	if e := resp.GetError(); e != nil {
		return nil, "", domain.NewError(e.GetCode(), e.GetMessage(), int(e.GetHttpStatus()))
	}
	return resp.GetVector(), resp.GetModel(), nil
}
