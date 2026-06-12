// Package clients — gRPC-клиенты tasksvc к другим микросервисам
// (groovesvc — хуки геймификации, aisvc — семантический поиск/реиндекс).
// Межсервисное общение — только gRPC.
package clients

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/groovepb"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

const grooveTimeout = 5 * time.Second

// Groove — fire-and-forget хуки геймификации (порт groove_client.py):
// зовутся ПОСЛЕ коммита, ошибки только в лог — геймификация не роняет
// основной флоу задач/юнитов.
type Groove struct {
	conn *grpc.ClientConn
	stub groovepb.GrooveServiceClient
	log  *slog.Logger
}

var _ domain.GrooveHooks = (*Groove)(nil)

func NewGroove(addr string, log *slog.Logger) (*Groove, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Groove{conn: conn, stub: groovepb.NewGrooveServiceClient(conn), log: log}, nil
}

func (c *Groove) Close() { _ = c.conn.Close() }

func (c *Groove) fire(method string, call func(ctx context.Context) error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), grooveTimeout)
		defer cancel()
		if err := call(ctx); err != nil {
			c.log.Warn("groove.hook_failed", "method", method, "error", err)
		}
	}()
}

func (c *Groove) OnUnitStarted(u *domain.Unit, taskName string) {
	req := &groovepb.UnitStartedRequest{
		CompanyId: u.CompanyID,
		UserId:    u.UserID,
		UnitId:    u.ID,
		UnitName:  u.Name,
		TaskId:    u.TaskID,
		TaskName:  taskName,
	}
	c.fire("OnUnitStarted", func(ctx context.Context) error {
		_, err := c.stub.OnUnitStarted(ctx, req)
		return err
	})
}

func (c *Groove) OnUnitStopped(u *domain.Unit, taskName string) {
	// Минуты — как во Flask on_unit_stopped: (end - start) / 60, int.
	minutes := int32(0)
	if u.DatetimeEnd != nil {
		minutes = int32(u.DatetimeEnd.Sub(u.DatetimeStart).Minutes())
	}
	req := &groovepb.UnitStoppedRequest{
		CompanyId: u.CompanyID,
		UserId:    u.UserID,
		UnitId:    u.ID,
		UnitName:  u.Name,
		TaskId:    u.TaskID,
		TaskName:  taskName,
		Minutes:   minutes,
	}
	c.fire("OnUnitStopped", func(ctx context.Context) error {
		_, err := c.stub.OnUnitStopped(ctx, req)
		return err
	})
}

func (c *Groove) OnTaskClosed(t *domain.Task, actorID int64) {
	// Герой закрытия: актор → ответственный → автор (как hero_id во Flask).
	heroID := actorID
	if heroID == 0 && t.ResponsibleUserID != nil {
		heroID = *t.ResponsibleUserID
	}
	if heroID == 0 {
		heroID = t.AuthorID
	}
	req := &groovepb.TaskClosedRequest{
		CompanyId:  t.CompanyID,
		HeroUserId: heroID,
		TaskId:     t.ID,
		TaskName:   t.Name,
	}
	c.fire("OnTaskClosed", func(ctx context.Context) error {
		_, err := c.stub.OnTaskClosed(ctx, req)
		return err
	})
}
