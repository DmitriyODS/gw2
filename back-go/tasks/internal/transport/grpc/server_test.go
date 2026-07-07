package grpc

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/status"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/taskspb"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
	gokitendpoint "github.com/go-kit/kit/endpoint"
)

// stubEndpoint — go-kit endpoint возвращающий фиксированный (response, err),
// с захватом последнего request для проверки сериализации proto → dto.
func stubEndpoint(resp any, err error, captured *any) gokitendpoint.Endpoint {
	return func(_ context.Context, request any) (any, error) {
		if captured != nil {
			*captured = request
		}
		return resp, err
	}
}

func TestGetStatsSummary_MapsFields(t *testing.T) {
	var captured any
	eps := endpoint.Endpoints{
		AssistantStatsSummary: stubEndpoint(&dto.AssistantSummary{
			PeriodLabel: "эта неделя", NewCount: 5, ClosedCount: 3,
			InProgressCount: 2, DebtCount: 1, TotalHours: 12.5,
		}, nil, &captured),
	}
	srv := NewServer(eps)

	resp, err := srv.GetStatsSummary(context.Background(), &taskspb.GetStatsSummaryRequest{
		CompanyId: 42, Period: "this_week",
	})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if resp.GetError() != nil {
		t.Fatalf("unexpected business error: %+v", resp.GetError())
	}
	if resp.GetNewCount() != 5 || resp.GetClosedCount() != 3 || resp.GetInProgressCount() != 2 ||
		resp.GetDebtCount() != 1 || resp.GetTotalHours() != 12.5 || resp.GetPeriodLabel() != "эта неделя" {
		t.Fatalf("response fields mismatch: %+v", resp)
	}

	req, ok := captured.(endpoint.AssistantPeriodRequest)
	if !ok {
		t.Fatalf("captured request has wrong type: %T", captured)
	}
	if req.CompanyID != 42 || req.Period != "this_week" {
		t.Fatalf("company_id/period not forwarded correctly: %+v", req)
	}
}

func TestGetStatsSummary_BusinessErrorInBand(t *testing.T) {
	eps := endpoint.Endpoints{
		AssistantStatsSummary: stubEndpoint(nil, domain.NewError("NOT_FOUND", "нет данных", 404), nil),
	}
	srv := NewServer(eps)

	resp, err := srv.GetStatsSummary(context.Background(), &taskspb.GetStatsSummaryRequest{CompanyId: 1})
	if err != nil {
		t.Fatalf("business errors must travel in-band, not as transport error: %v", err)
	}
	if resp.GetError() == nil || resp.GetError().Code != "NOT_FOUND" || resp.GetError().HttpStatus != 404 {
		t.Fatalf("expected NOT_FOUND/404 in Error field, got %+v", resp.GetError())
	}
}

func TestGetStatsSummary_TransportErrorBecomesInternal(t *testing.T) {
	eps := endpoint.Endpoints{
		AssistantStatsSummary: stubEndpoint(nil, errors.New("boom: db down"), nil),
	}
	srv := NewServer(eps)

	_, err := srv.GetStatsSummary(context.Background(), &taskspb.GetStatsSummaryRequest{CompanyId: 1})
	if err == nil {
		t.Fatal("expected a real gRPC transport error for a non-domain error")
	}
	if st, ok := status.FromError(err); !ok || st.Message() == "" {
		t.Fatalf("expected a status error, got %v", err)
	}
}

func TestListDepartments_MapsSliceAndPeriodLabel(t *testing.T) {
	eps := endpoint.Endpoints{
		AssistantDepartments: stubEndpoint(endpoint.AssistantListResult[[]dto.DeptStats]{
			Items: []dto.DeptStats{
				{DeptID: 1, Name: "Разработка", TasksCount: 7},
				{DeptID: 2, Name: "Поддержка", TasksCount: 3},
			},
			PeriodLabel: "этот месяц",
		}, nil, nil),
	}
	srv := NewServer(eps)

	resp, err := srv.ListDepartments(context.Background(), &taskspb.ListDepartmentsRequest{CompanyId: 1, Period: "this_month"})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if resp.GetPeriodLabel() != "этот месяц" {
		t.Fatalf("period_label = %q", resp.GetPeriodLabel())
	}
	if len(resp.GetDepartments()) != 2 {
		t.Fatalf("expected 2 departments, got %d", len(resp.GetDepartments()))
	}
	if resp.GetDepartments()[0].GetId() != 1 || resp.GetDepartments()[0].GetName() != "Разработка" ||
		resp.GetDepartments()[0].GetNewCount() != 7 {
		t.Fatalf("first department mismatch: %+v", resp.GetDepartments()[0])
	}
}

func TestGetTopEmployees_ForwardsCompanyPeriodLimit(t *testing.T) {
	var captured any
	eps := endpoint.Endpoints{
		AssistantTopEmployees: stubEndpoint(endpoint.AssistantListResult[[]dto.TaskByEmployee]{
			Items: []dto.TaskByEmployee{{FIO: "Иванов И.И.", TasksCount: 4, TotalHours: 20.5}},
		}, nil, &captured),
	}
	srv := NewServer(eps)

	resp, err := srv.GetTopEmployees(context.Background(), &taskspb.GetTopEmployeesRequest{
		CompanyId: 7, Period: "7d", Limit: 5,
	})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if len(resp.GetEmployees()) != 1 || resp.GetEmployees()[0].GetFio() != "Иванов И.И." ||
		resp.GetEmployees()[0].GetTaskCount() != 4 || resp.GetEmployees()[0].GetHours() != 20.5 {
		t.Fatalf("employee mapping mismatch: %+v", resp.GetEmployees())
	}
	req := captured.(endpoint.AssistantTopEmployeesRequest)
	if req.CompanyID != 7 || req.Period != "7d" || req.Limit != 5 {
		t.Fatalf("request not forwarded correctly: %+v", req)
	}
}

func TestGetStatsByUnitTypes_MapsFields(t *testing.T) {
	eps := endpoint.Endpoints{
		AssistantByUnitTypes: stubEndpoint(endpoint.AssistantListResult[[]dto.UnitTypeStats]{
			Items: []dto.UnitTypeStats{{Name: "Звонок", TotalHours: 3.5, TasksCount: 2}},
		}, nil, nil),
	}
	srv := NewServer(eps)

	resp, err := srv.GetStatsByUnitTypes(context.Background(), &taskspb.GetStatsByUnitTypesRequest{CompanyId: 1})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if len(resp.GetUnitTypes()) != 1 || resp.GetUnitTypes()[0].GetUnitTypeName() != "Звонок" ||
		resp.GetUnitTypes()[0].GetHours() != 3.5 || resp.GetUnitTypes()[0].GetTaskCount() != 2 {
		t.Fatalf("unit type mismatch: %+v", resp.GetUnitTypes())
	}
}

func TestGetStatsCalendar_MapsFields(t *testing.T) {
	eps := endpoint.Endpoints{
		AssistantCalendar: stubEndpoint(endpoint.AssistantListResult[[]dto.CalendarDay]{
			Items: []dto.CalendarDay{{Date: "2026-07-06", Received: 4, Closed: 2, TotalHours: 6}},
		}, nil, nil),
	}
	srv := NewServer(eps)

	resp, err := srv.GetStatsCalendar(context.Background(), &taskspb.GetStatsCalendarRequest{CompanyId: 1})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if len(resp.GetDays()) != 1 || resp.GetDays()[0].GetDate() != "2026-07-06" ||
		resp.GetDays()[0].GetNewCount() != 4 || resp.GetDays()[0].GetClosedCount() != 2 || resp.GetDays()[0].GetHours() != 6 {
		t.Fatalf("calendar day mismatch: %+v", resp.GetDays())
	}
}

func TestSearchTasks_MapsIDAndName(t *testing.T) {
	var captured any
	eps := endpoint.Endpoints{
		AssistantSearchTasks: stubEndpoint([]dto.Task{
			{ID: 10, Name: "Починить баг"},
			{ID: 11, Name: "Написать отчёт"},
		}, nil, &captured),
	}
	srv := NewServer(eps)

	resp, err := srv.SearchTasks(context.Background(), &taskspb.SearchTasksRequest{CompanyId: 9, Query: "баг", Limit: 5})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if len(resp.GetTasks()) != 2 || resp.GetTasks()[0].GetId() != 10 || resp.GetTasks()[0].GetName() != "Починить баг" {
		t.Fatalf("tasks mapping mismatch: %+v", resp.GetTasks())
	}
	// color — личный (per-user), у безличного gRPC-поиска его нет.
	if resp.GetTasks()[0].GetColor() != "" {
		t.Fatalf("expected empty color for a userless search, got %q", resp.GetTasks()[0].GetColor())
	}
	req := captured.(endpoint.AssistantSearchRequest)
	if req.CompanyID != 9 || req.Query != "баг" || req.Limit != 5 {
		t.Fatalf("request not forwarded correctly: %+v", req)
	}
}

func TestGetTaskLink_NotFoundWhenTaskMissingOrForeignCompany(t *testing.T) {
	// AssistantTaskLink (service layer) возвращает (nil, nil) — задачи нет
	// либо она из другой компании; транспорт обязан превратить это в
	// NOT_FOUND-ошибку в ответе, а не отдать пустой успешный ответ.
	eps := endpoint.Endpoints{
		AssistantTaskLink: stubEndpoint((*domain.Task)(nil), nil, nil),
	}
	srv := NewServer(eps)

	resp, err := srv.GetTaskLink(context.Background(), &taskspb.GetTaskLinkRequest{CompanyId: 1, TaskId: 999})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if resp.GetError() == nil || resp.GetError().Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND error, got %+v (id=%d)", resp.GetError(), resp.GetId())
	}
}

func TestGetTaskLink_ForwardsCompanyAndTaskID(t *testing.T) {
	var captured any
	eps := endpoint.Endpoints{
		AssistantTaskLink: stubEndpoint(&domain.Task{
			ID: 55, Name: "Развернуть прод",
			Responsible: &domain.UserRef{ID: 3, FIO: "Петров П.П."},
		}, nil, &captured),
	}
	srv := NewServer(eps)

	resp, err := srv.GetTaskLink(context.Background(), &taskspb.GetTaskLinkRequest{CompanyId: 4, TaskId: 55})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if resp.GetError() != nil {
		t.Fatalf("unexpected error: %+v", resp.GetError())
	}
	if resp.GetId() != 55 || resp.GetName() != "Развернуть прод" || resp.GetResponsibleFio() != "Петров П.П." {
		t.Fatalf("response mismatch: %+v", resp)
	}
	req := captured.(endpoint.AssistantTaskLinkRequest)
	if req.CompanyID != 4 || req.TaskID != 55 {
		t.Fatalf("company_id/task_id not forwarded: %+v", req)
	}
}

func TestGetTaskLink_NoResponsibleLeavesFioEmpty(t *testing.T) {
	eps := endpoint.Endpoints{
		AssistantTaskLink: stubEndpoint(&domain.Task{ID: 1, Name: "Без ответственного"}, nil, nil),
	}
	srv := NewServer(eps)

	resp, err := srv.GetTaskLink(context.Background(), &taskspb.GetTaskLinkRequest{CompanyId: 1, TaskId: 1})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if resp.GetResponsibleFio() != "" {
		t.Fatalf("expected empty responsible_fio, got %q", resp.GetResponsibleFio())
	}
}
