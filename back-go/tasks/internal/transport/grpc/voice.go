// Голосовые операции навыка Алисы (alicesvc): создание/закрытие задач,
// список открытых, справочники и юниты от имени конкретного пользователя.
// Зовут сервис напрямую (без endpoint-обёрток — операции тонкие).
package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DmitriyODS/gw2/back-go/pkg/gen/taskspb"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// voiceErr — доменная ошибка в поле ответа, прочее — Internal.
func voiceErr(err error) (*taskspb.Error, error) {
	if de := domain.AsDomainError(err); de != nil {
		return pbError(de), nil
	}
	return nil, status.Error(codes.Internal, err.Error())
}

func (s *Server) CreateTask(ctx context.Context, req *taskspb.CreateTaskRequest) (*taskspb.CreateTaskResponse, error) {
	deptID := req.GetDepartmentId()
	if deptID == 0 {
		depts, err := s.svc.ListDepartments(ctx, req.GetCompanyId())
		if err != nil {
			pe, ierr := voiceErr(err)
			return &taskspb.CreateTaskResponse{Error: pe}, ierr
		}
		switch len(depts) {
		case 0:
			return &taskspb.CreateTaskResponse{Error: &taskspb.Error{
				Code: "NO_DEPARTMENTS", Message: "В компании нет отделов", HttpStatus: 422,
			}}, nil
		case 1:
			deptID = depts[0].ID
		default:
			return &taskspb.CreateTaskResponse{Error: &taskspb.Error{
				Code: "DEPARTMENT_REQUIRED", Message: "Нужно выбрать отдел", HttpStatus: 422,
			}}, nil
		}
	}
	task, err := s.svc.CreateTask(ctx, req.GetUserId(), req.GetCompanyId(), dto.TaskCreate{
		Name: req.GetName(), DepartmentID: deptID,
	})
	if err != nil {
		pe, ierr := voiceErr(err)
		return &taskspb.CreateTaskResponse{Error: pe}, ierr
	}
	return &taskspb.CreateTaskResponse{Id: task.ID, Name: task.Name}, nil
}

func (s *Server) CloseTask(ctx context.Context, req *taskspb.CloseTaskRequest) (*taskspb.CloseTaskResponse, error) {
	companyID := req.GetCompanyId()
	task, err := s.svc.ArchiveTask(ctx, req.GetTaskId(), req.GetUserId(), &companyID)
	if err != nil {
		pe, ierr := voiceErr(err)
		return &taskspb.CloseTaskResponse{Error: pe}, ierr
	}
	return &taskspb.CloseTaskResponse{Name: task.Name}, nil
}

func (s *Server) ListOpenTasks(ctx context.Context, req *taskspb.ListOpenTasksRequest) (*taskspb.ListOpenTasksResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 5
	}
	companyID := req.GetCompanyId()
	f := domain.TaskListFilter{
		CurrentUserID: req.GetUserId(),
		CompanyID:     &companyID,
		Tab:           "active",
		Sort:          "last_activity",
		Page:          1,
		PerPage:       limit,
	}
	if req.GetOnlyMine() {
		uid := req.GetUserId()
		f.ResponsibleUserID = &uid
	}
	list, err := s.svc.ListTasks(ctx, f)
	if err != nil {
		pe, ierr := voiceErr(err)
		return &taskspb.ListOpenTasksResponse{Error: pe}, ierr
	}
	out := make([]*taskspb.TaskRef, 0, len(list.Items))
	for _, t := range list.Items {
		out = append(out, &taskspb.TaskRef{Id: t.ID, Name: t.Name})
	}
	return &taskspb.ListOpenTasksResponse{Tasks: out, Total: int32(list.Total)}, nil
}

func (s *Server) ListAllDepartments(ctx context.Context, req *taskspb.ListAllDepartmentsRequest) (*taskspb.ListAllDepartmentsResponse, error) {
	depts, err := s.svc.ListDepartments(ctx, req.GetCompanyId())
	if err != nil {
		pe, ierr := voiceErr(err)
		return &taskspb.ListAllDepartmentsResponse{Error: pe}, ierr
	}
	out := make([]*taskspb.CatalogItem, 0, len(depts))
	for _, d := range depts {
		out = append(out, &taskspb.CatalogItem{Id: d.ID, Name: d.Name})
	}
	return &taskspb.ListAllDepartmentsResponse{Departments: out}, nil
}

func (s *Server) ListUnitTypes(ctx context.Context, req *taskspb.ListUnitTypesRequest) (*taskspb.ListUnitTypesResponse, error) {
	types, err := s.svc.ListUnitTypes(ctx, req.GetCompanyId())
	if err != nil {
		pe, ierr := voiceErr(err)
		return &taskspb.ListUnitTypesResponse{Error: pe}, ierr
	}
	out := make([]*taskspb.CatalogItem, 0, len(types))
	for _, t := range types {
		out = append(out, &taskspb.CatalogItem{Id: t.ID, Name: t.Name})
	}
	return &taskspb.ListUnitTypesResponse{UnitTypes: out}, nil
}

func (s *Server) StartUnit(ctx context.Context, req *taskspb.StartUnitRequest) (*taskspb.StartUnitResponse, error) {
	companyID := req.GetCompanyId()
	typeID := req.GetUnitTypeId()
	if typeID == 0 {
		types, err := s.svc.ListUnitTypes(ctx, companyID)
		if err != nil {
			pe, ierr := voiceErr(err)
			return &taskspb.StartUnitResponse{Error: pe}, ierr
		}
		switch len(types) {
		case 0:
			return &taskspb.StartUnitResponse{Error: &taskspb.Error{
				Code: "NO_UNIT_TYPES", Message: "В компании нет типов юнитов", HttpStatus: 422,
			}}, nil
		case 1:
			typeID = types[0].ID
		default:
			return &taskspb.StartUnitResponse{Error: &taskspb.Error{
				Code: "UNIT_TYPE_REQUIRED", Message: "Нужно выбрать тип юнита", HttpStatus: 422,
			}}, nil
		}
	}
	task, err := s.svc.GetTaskInCompany(ctx, req.GetTaskId(), req.GetUserId(), &companyID)
	if err != nil {
		pe, ierr := voiceErr(err)
		return &taskspb.StartUnitResponse{Error: pe}, ierr
	}
	name := req.GetName()
	if name == "" {
		name = task.Name
	}
	unit, err := s.svc.CreateUnit(ctx, req.GetTaskId(), req.GetUserId(), &companyID, name, typeID)
	if err != nil {
		pe, ierr := voiceErr(err)
		return &taskspb.StartUnitResponse{Error: pe}, ierr
	}
	return &taskspb.StartUnitResponse{UnitId: unit.ID, TaskName: task.Name}, nil
}

func (s *Server) StopActiveUnit(ctx context.Context, req *taskspb.StopActiveUnitRequest) (*taskspb.StopActiveUnitResponse, error) {
	stopped, err := s.svc.StopActiveUnit(ctx, req.GetUserId())
	if err != nil {
		pe, ierr := voiceErr(err)
		return &taskspb.StopActiveUnitResponse{Error: pe}, ierr
	}
	if stopped == nil {
		return &taskspb.StopActiveUnitResponse{Error: &taskspb.Error{
			Code: "NO_ACTIVE_UNIT", Message: "Активного юнита нет", HttpStatus: 404,
		}}, nil
	}
	resp := &taskspb.StopActiveUnitResponse{UnitName: stopped.Name}
	if t, err := s.svc.GetTask(ctx, stopped.TaskID, req.GetUserId()); err == nil {
		resp.TaskName = t.Name
	}
	if stopped.DatetimeEnd != nil {
		resp.Minutes = int32(time.Time(*stopped.DatetimeEnd).Sub(time.Time(stopped.DatetimeStart)).Minutes())
	}
	return resp, nil
}

func (s *Server) GetActiveUnit(ctx context.Context, req *taskspb.GetActiveUnitRequest) (*taskspb.GetActiveUnitResponse, error) {
	active, err := s.svc.ActiveUnit(ctx, req.GetUserId())
	if err != nil {
		pe, ierr := voiceErr(err)
		return &taskspb.GetActiveUnitResponse{Error: pe}, ierr
	}
	if active == nil {
		return &taskspb.GetActiveUnitResponse{Active: false}, nil
	}
	resp := &taskspb.GetActiveUnitResponse{
		Active:   true,
		UnitId:   active.ID,
		TaskId:   active.TaskID,
		UnitName: active.Name,
		Minutes:  int32(time.Since(time.Time(active.DatetimeStart)).Minutes()),
	}
	if t, err := s.svc.GetTask(ctx, active.TaskID, req.GetUserId()); err == nil {
		resp.TaskName = t.Name
	}
	return resp, nil
}
