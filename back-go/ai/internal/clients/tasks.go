// Package clients — исходящие gRPC-клиенты aisvc (сейчас — только
// tasksvc: статистика и поиск задач для инструментов ИИ-ассистента).
package clients

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DmitriyODS/gw2/back-go/ai/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/gen/taskspb"
)

const tasksTimeout = 10 * time.Second

// Tasks — клиент tasksvc для инструментов ассистента.
type Tasks struct {
	conn *grpc.ClientConn
	stub taskspb.TasksServiceClient
	log  *slog.Logger
}

var _ domain.TasksClient = (*Tasks)(nil)

func NewTasks(addr string, log *slog.Logger) (*Tasks, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Tasks{conn: conn, stub: taskspb.NewTasksServiceClient(conn), log: log}, nil
}

func (c *Tasks) Close() { _ = c.conn.Close() }

func (c *Tasks) GetStatsSummary(ctx context.Context, companyID int64, period string) (*domain.StatsSummary, error) {
	rctx, cancel := context.WithTimeout(ctx, tasksTimeout)
	defer cancel()
	resp, err := c.stub.GetStatsSummary(rctx, &taskspb.GetStatsSummaryRequest{CompanyId: companyID, Period: period})
	if err != nil {
		return nil, err
	}
	if e := resp.GetError(); e != nil {
		return nil, domain.NewError(e.Code, e.Message, int(e.HttpStatus))
	}
	return &domain.StatsSummary{
		NewCount:        int(resp.GetNewCount()),
		ClosedCount:     int(resp.GetClosedCount()),
		InProgressCount: int(resp.GetInProgressCount()),
		DebtCount:       int(resp.GetDebtCount()),
		TotalHours:      resp.GetTotalHours(),
		PeriodLabel:     resp.GetPeriodLabel(),
	}, nil
}

func (c *Tasks) ListDepartments(ctx context.Context, companyID int64, period string) ([]domain.DepartmentStat, error) {
	rctx, cancel := context.WithTimeout(ctx, tasksTimeout)
	defer cancel()
	resp, err := c.stub.ListDepartments(rctx, &taskspb.ListDepartmentsRequest{CompanyId: companyID, Period: period})
	if err != nil {
		return nil, err
	}
	if e := resp.GetError(); e != nil {
		return nil, domain.NewError(e.Code, e.Message, int(e.HttpStatus))
	}
	out := make([]domain.DepartmentStat, 0, len(resp.GetDepartments()))
	for _, d := range resp.GetDepartments() {
		out = append(out, domain.DepartmentStat{ID: d.GetId(), Name: d.GetName(), NewCount: int(d.GetNewCount())})
	}
	return out, nil
}

func (c *Tasks) GetTopEmployees(ctx context.Context, companyID int64, period string, limit int) ([]domain.EmployeeStat, error) {
	rctx, cancel := context.WithTimeout(ctx, tasksTimeout)
	defer cancel()
	resp, err := c.stub.GetTopEmployees(rctx, &taskspb.GetTopEmployeesRequest{
		CompanyId: companyID, Period: period, Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	if e := resp.GetError(); e != nil {
		return nil, domain.NewError(e.Code, e.Message, int(e.HttpStatus))
	}
	out := make([]domain.EmployeeStat, 0, len(resp.GetEmployees()))
	for _, e := range resp.GetEmployees() {
		out = append(out, domain.EmployeeStat{FIO: e.GetFio(), TaskCount: int(e.GetTaskCount()), Hours: e.GetHours()})
	}
	return out, nil
}

func (c *Tasks) GetStatsByUnitTypes(ctx context.Context, companyID int64, period string) ([]domain.UnitTypeStat, error) {
	rctx, cancel := context.WithTimeout(ctx, tasksTimeout)
	defer cancel()
	resp, err := c.stub.GetStatsByUnitTypes(rctx, &taskspb.GetStatsByUnitTypesRequest{CompanyId: companyID, Period: period})
	if err != nil {
		return nil, err
	}
	if e := resp.GetError(); e != nil {
		return nil, domain.NewError(e.Code, e.Message, int(e.HttpStatus))
	}
	out := make([]domain.UnitTypeStat, 0, len(resp.GetUnitTypes()))
	for _, u := range resp.GetUnitTypes() {
		out = append(out, domain.UnitTypeStat{Name: u.GetUnitTypeName(), Hours: u.GetHours(), TaskCount: int(u.GetTaskCount())})
	}
	return out, nil
}

func (c *Tasks) GetStatsCalendar(ctx context.Context, companyID int64, period string) ([]domain.CalendarDayStat, error) {
	rctx, cancel := context.WithTimeout(ctx, tasksTimeout)
	defer cancel()
	resp, err := c.stub.GetStatsCalendar(rctx, &taskspb.GetStatsCalendarRequest{CompanyId: companyID, Period: period})
	if err != nil {
		return nil, err
	}
	if e := resp.GetError(); e != nil {
		return nil, domain.NewError(e.Code, e.Message, int(e.HttpStatus))
	}
	out := make([]domain.CalendarDayStat, 0, len(resp.GetDays()))
	for _, d := range resp.GetDays() {
		out = append(out, domain.CalendarDayStat{
			Date: d.GetDate(), NewCount: int(d.GetNewCount()), ClosedCount: int(d.GetClosedCount()), Hours: d.GetHours(),
		})
	}
	return out, nil
}

func (c *Tasks) SearchTasks(ctx context.Context, companyID int64, query string, limit int) ([]domain.TaskRef, error) {
	rctx, cancel := context.WithTimeout(ctx, tasksTimeout)
	defer cancel()
	resp, err := c.stub.SearchTasks(rctx, &taskspb.SearchTasksRequest{
		CompanyId: companyID, Query: query, Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	if e := resp.GetError(); e != nil {
		return nil, domain.NewError(e.Code, e.Message, int(e.HttpStatus))
	}
	out := make([]domain.TaskRef, 0, len(resp.GetTasks()))
	for _, t := range resp.GetTasks() {
		out = append(out, domain.TaskRef{ID: t.GetId(), Name: t.GetName(), Color: t.GetColor()})
	}
	return out, nil
}

// GetTaskLink — nil без ошибки на NOT_FOUND (задачи нет/чужая компания) —
// вызывающий (find_task) трактует это как «не нашёл», а не как сбой.
func (c *Tasks) GetTaskLink(ctx context.Context, companyID, taskID int64) (*domain.TaskRef, error) {
	rctx, cancel := context.WithTimeout(ctx, tasksTimeout)
	defer cancel()
	resp, err := c.stub.GetTaskLink(rctx, &taskspb.GetTaskLinkRequest{CompanyId: companyID, TaskId: taskID})
	if err != nil {
		return nil, err
	}
	if e := resp.GetError(); e != nil {
		if e.Code == "NOT_FOUND" {
			return nil, nil
		}
		return nil, domain.NewError(e.Code, e.Message, int(e.HttpStatus))
	}
	return &domain.TaskRef{
		ID: resp.GetId(), Name: resp.GetName(), Color: resp.GetColor(), ResponsibleFIO: resp.GetResponsibleFio(),
	}, nil
}
