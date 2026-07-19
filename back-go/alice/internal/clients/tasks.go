// Package clients — gRPC-клиенты alicesvc к сервисам-владельцам
// (tasksvc/diarysvc/notesvc/aisvc). Бизнес-ошибки приходят полем Error
// в ответе и конвертируются в domain.Error с исходным кодом.
package clients

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/alice/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/taskspb"
)

const callTimeout = 8 * time.Second

func dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

// pbErr — Error из ответа → domain.Error. Геттеры protobuf nil-безопасны,
// поэтому типизированный nil-указатель за интерфейсом отдаёт пустой код —
// это «ошибки нет».
func pbErr(e interface {
	GetCode() string
	GetMessage() string
	GetHttpStatus() int32
}) error {
	if e == nil || (e.GetCode() == "" && e.GetMessage() == "") {
		return nil
	}
	return domain.NewError(e.GetCode(), e.GetMessage(), int(e.GetHttpStatus()))
}

type Tasks struct {
	conn *grpc.ClientConn
	stub taskspb.TasksServiceClient
}

var _ domain.TasksClient = (*Tasks)(nil)

func NewTasks(addr string) (*Tasks, error) {
	conn, err := dial(addr)
	if err != nil {
		return nil, err
	}
	return &Tasks{conn: conn, stub: taskspb.NewTasksServiceClient(conn)}, nil
}

func (c *Tasks) Close() { _ = c.conn.Close() }

func (c *Tasks) SearchTasks(ctx context.Context, companyID int64, query string, limit int) ([]domain.TaskRef, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.SearchTasks(ctx, &taskspb.SearchTasksRequest{
		CompanyId: companyID, Query: query, Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil {
		return nil, pbErr(resp.GetError())
	}
	out := make([]domain.TaskRef, 0, len(resp.GetTasks()))
	for _, t := range resp.GetTasks() {
		out = append(out, domain.TaskRef{ID: t.GetId(), Name: t.GetName()})
	}
	return out, nil
}

func (c *Tasks) CreateTask(ctx context.Context, companyID, userID int64, name string, departmentID int64) (*domain.TaskRef, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.CreateTask(ctx, &taskspb.CreateTaskRequest{
		CompanyId: companyID, UserId: userID, Name: name, DepartmentId: departmentID,
	})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil {
		return nil, pbErr(resp.GetError())
	}
	return &domain.TaskRef{ID: resp.GetId(), Name: resp.GetName()}, nil
}

func (c *Tasks) CloseTask(ctx context.Context, companyID, userID, taskID int64) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.CloseTask(ctx, &taskspb.CloseTaskRequest{
		CompanyId: companyID, UserId: userID, TaskId: taskID,
	})
	if err != nil {
		return "", err
	}
	if resp.GetError() != nil {
		return "", pbErr(resp.GetError())
	}
	return resp.GetName(), nil
}

func (c *Tasks) ListOpenTasks(ctx context.Context, companyID, userID int64, limit int) ([]domain.TaskRef, int, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.ListOpenTasks(ctx, &taskspb.ListOpenTasksRequest{
		CompanyId: companyID, UserId: userID, Limit: int32(limit),
	})
	if err != nil {
		return nil, 0, err
	}
	if resp.GetError() != nil {
		return nil, 0, pbErr(resp.GetError())
	}
	out := make([]domain.TaskRef, 0, len(resp.GetTasks()))
	for _, t := range resp.GetTasks() {
		out = append(out, domain.TaskRef{ID: t.GetId(), Name: t.GetName()})
	}
	return out, int(resp.GetTotal()), nil
}

func (c *Tasks) ListDepartments(ctx context.Context, companyID int64) ([]domain.CatalogItem, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.ListAllDepartments(ctx, &taskspb.ListAllDepartmentsRequest{CompanyId: companyID})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil {
		return nil, pbErr(resp.GetError())
	}
	out := make([]domain.CatalogItem, 0, len(resp.GetDepartments()))
	for _, d := range resp.GetDepartments() {
		out = append(out, domain.CatalogItem{ID: d.GetId(), Name: d.GetName()})
	}
	return out, nil
}

func (c *Tasks) ListUnitTypes(ctx context.Context, companyID int64) ([]domain.CatalogItem, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.ListUnitTypes(ctx, &taskspb.ListUnitTypesRequest{CompanyId: companyID})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil {
		return nil, pbErr(resp.GetError())
	}
	out := make([]domain.CatalogItem, 0, len(resp.GetUnitTypes()))
	for _, t := range resp.GetUnitTypes() {
		out = append(out, domain.CatalogItem{ID: t.GetId(), Name: t.GetName()})
	}
	return out, nil
}

func (c *Tasks) StartUnit(ctx context.Context, companyID, userID, taskID, unitTypeID int64) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.StartUnit(ctx, &taskspb.StartUnitRequest{
		CompanyId: companyID, UserId: userID, TaskId: taskID, UnitTypeId: unitTypeID,
	})
	if err != nil {
		return "", err
	}
	if resp.GetError() != nil {
		return "", pbErr(resp.GetError())
	}
	return resp.GetTaskName(), nil
}

func (c *Tasks) StopActiveUnit(ctx context.Context, userID int64) (*domain.StoppedUnit, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.StopActiveUnit(ctx, &taskspb.StopActiveUnitRequest{UserId: userID})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil {
		return nil, pbErr(resp.GetError())
	}
	return &domain.StoppedUnit{
		UnitName: resp.GetUnitName(), TaskName: resp.GetTaskName(), Minutes: int(resp.GetMinutes()),
	}, nil
}

func (c *Tasks) GetActiveUnit(ctx context.Context, userID int64) (*domain.ActiveUnit, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	resp, err := c.stub.GetActiveUnit(ctx, &taskspb.GetActiveUnitRequest{UserId: userID})
	if err != nil {
		return nil, err
	}
	if resp.GetError() != nil {
		return nil, pbErr(resp.GetError())
	}
	if !resp.GetActive() {
		return nil, nil
	}
	return &domain.ActiveUnit{
		UnitID: resp.GetUnitId(), TaskID: resp.GetTaskId(),
		UnitName: resp.GetUnitName(), TaskName: resp.GetTaskName(), Minutes: int(resp.GetMinutes()),
	}, nil
}
