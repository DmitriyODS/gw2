// Package clients — gRPC-клиенты authsvc к другим микросервисам.
// Межсервисное общение — только gRPC.
package clients

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/mailpb"
)

// Mail — клиент mailsvc: отправка брендированных писем (подтверждение email).
type Mail struct {
	conn *grpc.ClientConn
	stub mailpb.MailServiceClient
	log  *slog.Logger
}

var _ domain.MailClient = (*Mail)(nil)

func NewMail(addr string, log *slog.Logger) (*Mail, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Mail{conn: conn, stub: mailpb.NewMailServiceClient(conn), log: log}, nil
}

func (c *Mail) Close() { _ = c.conn.Close() }

func (c *Mail) send(ctx context.Context, to, toName, template string, params map[string]string) error {
	resp, err := c.stub.Send(ctx, &mailpb.SendRequest{
		To: to, ToName: toName, Template: template, Params: params,
	})
	if err != nil {
		return err
	}
	if resp.GetError() != nil {
		return fmt.Errorf("%s: %s", resp.GetError().GetCode(), resp.GetError().GetMessage())
	}
	return nil
}

func (c *Mail) SendVerification(ctx context.Context, to, fio, code, link string) error {
	return c.send(ctx, to, fio, "verify_email", map[string]string{"fio": fio, "code": code, "link": link})
}

func (c *Mail) SendPasswordReset(ctx context.Context, to, fio, link string) error {
	return c.send(ctx, to, fio, "reset_password", map[string]string{"fio": fio, "link": link})
}

func (c *Mail) SendCompanyInvite(ctx context.Context, to, companyName, roleName, link string) error {
	return c.send(ctx, to, "", "company_invite", map[string]string{"company": companyName, "role": roleName, "link": link})
}
