// Package clients — gRPC-клиенты petsvc к другим микросервисам.
// Межсервисное общение — только gRPC.
package clients

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/portalpb"
)

const portalTimeout = 5 * time.Second

// Portal — fire-and-forget системные посты корпоративного портала
// (поздравление с эволюцией питомца): зовётся после фиксации эволюции,
// ошибки только в лог — недоступный portalsvc гейм-механику не роняет.
// Дедуп повторных постов (ретраи хука) — на стороне portalsvc.
type Portal struct {
	conn *grpc.ClientConn
	stub portalpb.PortalServiceClient
	log  *slog.Logger
}

var _ domain.PortalClient = (*Portal)(nil)

func NewPortal(addr string, log *slog.Logger) (*Portal, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Portal{conn: conn, stub: portalpb.NewPortalServiceClient(conn), log: log}, nil
}

func (c *Portal) Close() { _ = c.conn.Close() }

func (c *Portal) CreateSystemPost(companyID, authorUserID int64, systemKind, title, body string) {
	req := &portalpb.CreateSystemPostRequest{
		CompanyId:    companyID,
		AuthorUserId: authorUserID,
		SystemKind:   systemKind,
		Title:        title,
		Body:         body,
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.log.Warn("portal.system_post_panic", "kind", systemKind, "panic", r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), portalTimeout)
		defer cancel()
		resp, err := c.stub.CreateSystemPost(ctx, req)
		if err != nil {
			c.log.Warn("portal.system_post_failed", "kind", systemKind,
				"company_id", companyID, "error", err)
			return
		}
		if e := resp.GetError(); e != nil {
			c.log.Warn("portal.system_post_rejected", "kind", systemKind,
				"company_id", companyID, "code", e.GetCode(), "message", e.GetMessage())
		}
	}()
}
