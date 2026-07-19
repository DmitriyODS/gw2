// Package grpc — исходящий gRPC-транспорт tasksvc (его зовёт aisvc:
// инструменты статистики и поиска задач ИИ-ассистента). Бизнес-ошибки
// уезжают полем Error в ответе — транспорт всегда отвечает OK.
package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/taskspb"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/service"
)

type Server struct {
	taskspb.UnimplementedTasksServiceServer
	eps endpoint.Endpoints
	// svc — прямой доступ для голосовых операций Алисы (alicesvc, voice.go):
	// тонкие вызовы без endpoint-обёрток.
	svc *service.Service
}

func NewServer(eps endpoint.Endpoints, svc *service.Service) *Server {
	return &Server{eps: eps, svc: svc}
}

func (s *Server) GetStatsSummary(ctx context.Context, req *taskspb.GetStatsSummaryRequest) (*taskspb.GetStatsSummaryResponse, error) {
	resp, err := s.eps.AssistantStatsSummary(ctx, endpoint.AssistantPeriodRequest{
		CompanyID: req.GetCompanyId(), Period: req.GetPeriod(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &taskspb.GetStatsSummaryResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(*dto.AssistantSummary)
	return &taskspb.GetStatsSummaryResponse{
		NewCount:        int32(r.NewCount),
		ClosedCount:     int32(r.ClosedCount),
		InProgressCount: int32(r.InProgressCount),
		DebtCount:       int32(r.DebtCount),
		TotalHours:      r.TotalHours,
		PeriodLabel:     r.PeriodLabel,
	}, nil
}

func (s *Server) ListDepartments(ctx context.Context, req *taskspb.ListDepartmentsRequest) (*taskspb.ListDepartmentsResponse, error) {
	resp, err := s.eps.AssistantDepartments(ctx, endpoint.AssistantPeriodRequest{
		CompanyID: req.GetCompanyId(), Period: req.GetPeriod(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &taskspb.ListDepartmentsResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(endpoint.AssistantListResult[[]dto.DeptStats])
	out := make([]*taskspb.DepartmentStat, 0, len(r.Items))
	for _, d := range r.Items {
		out = append(out, &taskspb.DepartmentStat{Id: d.DeptID, Name: d.Name, NewCount: int32(d.TasksCount)})
	}
	return &taskspb.ListDepartmentsResponse{Departments: out, PeriodLabel: r.PeriodLabel}, nil
}

func (s *Server) GetTopEmployees(ctx context.Context, req *taskspb.GetTopEmployeesRequest) (*taskspb.GetTopEmployeesResponse, error) {
	resp, err := s.eps.AssistantTopEmployees(ctx, endpoint.AssistantTopEmployeesRequest{
		CompanyID: req.GetCompanyId(), Period: req.GetPeriod(), Limit: int(req.GetLimit()),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &taskspb.GetTopEmployeesResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(endpoint.AssistantListResult[[]dto.TaskByEmployee])
	out := make([]*taskspb.EmployeeStat, 0, len(r.Items))
	for _, e := range r.Items {
		out = append(out, &taskspb.EmployeeStat{Fio: e.FIO, TaskCount: int32(e.TasksCount), Hours: e.TotalHours})
	}
	return &taskspb.GetTopEmployeesResponse{Employees: out, PeriodLabel: r.PeriodLabel}, nil
}

func (s *Server) GetStatsByUnitTypes(ctx context.Context, req *taskspb.GetStatsByUnitTypesRequest) (*taskspb.GetStatsByUnitTypesResponse, error) {
	resp, err := s.eps.AssistantByUnitTypes(ctx, endpoint.AssistantPeriodRequest{
		CompanyID: req.GetCompanyId(), Period: req.GetPeriod(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &taskspb.GetStatsByUnitTypesResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(endpoint.AssistantListResult[[]dto.UnitTypeStats])
	out := make([]*taskspb.UnitTypeStat, 0, len(r.Items))
	for _, u := range r.Items {
		out = append(out, &taskspb.UnitTypeStat{UnitTypeName: u.Name, Hours: u.TotalHours, TaskCount: int32(u.TasksCount)})
	}
	return &taskspb.GetStatsByUnitTypesResponse{UnitTypes: out, PeriodLabel: r.PeriodLabel}, nil
}

func (s *Server) GetStatsCalendar(ctx context.Context, req *taskspb.GetStatsCalendarRequest) (*taskspb.GetStatsCalendarResponse, error) {
	resp, err := s.eps.AssistantCalendar(ctx, endpoint.AssistantPeriodRequest{
		CompanyID: req.GetCompanyId(), Period: req.GetPeriod(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &taskspb.GetStatsCalendarResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	r := resp.(endpoint.AssistantListResult[[]dto.CalendarDay])
	out := make([]*taskspb.CalendarDayStat, 0, len(r.Items))
	for _, d := range r.Items {
		out = append(out, &taskspb.CalendarDayStat{
			Date: d.Date, NewCount: int32(d.Received), ClosedCount: int32(d.Closed), Hours: d.TotalHours,
		})
	}
	return &taskspb.GetStatsCalendarResponse{Days: out, PeriodLabel: r.PeriodLabel}, nil
}

func (s *Server) SearchTasks(ctx context.Context, req *taskspb.SearchTasksRequest) (*taskspb.SearchTasksResponse, error) {
	resp, err := s.eps.AssistantSearchTasks(ctx, endpoint.AssistantSearchRequest{
		CompanyID: req.GetCompanyId(), Query: req.GetQuery(), Limit: int(req.GetLimit()),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &taskspb.SearchTasksResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	items := resp.([]dto.Task)
	out := make([]*taskspb.TaskRef, 0, len(items))
	for _, t := range items {
		// color — личный (per-user); у безличного gRPC-поиска его нет.
		out = append(out, &taskspb.TaskRef{Id: t.ID, Name: t.Name})
	}
	return &taskspb.SearchTasksResponse{Tasks: out}, nil
}

func (s *Server) GetTaskLink(ctx context.Context, req *taskspb.GetTaskLinkRequest) (*taskspb.GetTaskLinkResponse, error) {
	resp, err := s.eps.AssistantTaskLink(ctx, endpoint.AssistantTaskLinkRequest{
		CompanyID: req.GetCompanyId(), TaskID: req.GetTaskId(),
	})
	if err != nil {
		if de := domain.AsDomainError(err); de != nil {
			return &taskspb.GetTaskLinkResponse{Error: pbError(de)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	task, _ := resp.(*domain.Task)
	if task == nil {
		return &taskspb.GetTaskLinkResponse{Error: &taskspb.Error{
			Code: "NOT_FOUND", Message: "Задача не найдена", HttpStatus: 404,
		}}, nil
	}
	out := &taskspb.GetTaskLinkResponse{Id: task.ID, Name: task.Name}
	if task.Responsible != nil {
		out.ResponsibleFio = task.Responsible.FIO
	}
	return out, nil
}

func pbError(e *domain.Error) *taskspb.Error {
	return &taskspb.Error{Code: e.Code, Message: e.Message, HttpStatus: int32(e.HTTPStatus)}
}
