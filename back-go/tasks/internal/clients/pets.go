// Package clients — gRPC-клиенты tasksvc к другим микросервисам
// (petsvc — хуки геймификации, aisvc — семантический поиск/реиндекс).
// Межсервисное общение — только gRPC.
package clients

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/petspb"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

const petsTimeout = 5 * time.Second

// Pets — fire-and-forget хуки геймификации: зовутся ПОСЛЕ коммита, ошибки
// только в лог — геймификация не роняет основной флоу задач/юнитов.
type Pets struct {
	conn *grpc.ClientConn
	stub petspb.PetsServiceClient
	log  *slog.Logger
}

var _ domain.PetsHooks = (*Pets)(nil)

func NewPets(addr string, log *slog.Logger) (*Pets, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Pets{conn: conn, stub: petspb.NewPetsServiceClient(conn), log: log}, nil
}

func (c *Pets) Close() { _ = c.conn.Close() }

func (c *Pets) fire(method string, call func(ctx context.Context) error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), petsTimeout)
		defer cancel()
		if err := call(ctx); err != nil {
			c.log.Warn("pets.hook_failed", "method", method, "error", err)
		}
	}()
}

func (c *Pets) OnUnitStarted(u *domain.Unit, taskName string) {
	req := &petspb.UnitStartedRequest{
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

func (c *Pets) OnUnitStopped(u *domain.Unit, taskName string) {
	// Минуты — как во Flask on_unit_stopped: (end - start) / 60, int.
	minutes := int32(0)
	if u.DatetimeEnd != nil {
		minutes = int32(u.DatetimeEnd.Sub(u.DatetimeStart).Minutes())
	}
	req := &petspb.UnitStoppedRequest{
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

func (c *Pets) OnTaskClosed(t *domain.Task, actorID int64) {
	// Герой закрытия: актор → ответственный → автор (как hero_id во Flask).
	heroID := actorID
	if heroID == 0 && t.ResponsibleUserID != nil {
		heroID = *t.ResponsibleUserID
	}
	if heroID == 0 {
		heroID = t.AuthorID
	}
	req := &petspb.TaskClosedRequest{
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
