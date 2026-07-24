package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

func period(start, end time.Time) dto.Period {
	return dto.Period{
		From: start.UTC().Format("2006-01-02"),
		To:   end.UTC().Format("2006-01-02"),
	}
}

func (s *Service) StatsCommon(ctx context.Context, start, end time.Time, companyID *int64) (*dto.StatsCommon, error) {
	metrics, err := s.stats.CommonMetrics(ctx, start, end, companyID)
	if err != nil {
		return nil, err
	}
	byHours, err := s.stats.TasksByHours(ctx, start, end, companyID)
	if err != nil {
		return nil, err
	}
	byEmployees, err := s.stats.TasksByEmployees(ctx, start, end, companyID)
	if err != nil {
		return nil, err
	}
	return &dto.StatsCommon{
		Period: period(start, end),
		Tasks: dto.TaskMetrics{
			Closed: metrics.Closed, Debt: metrics.Debt,
			Received: metrics.Received, Remaining: metrics.Remaining,
		},
		TasksByEmployees: dto.NewTaskByEmployees(byEmployees),
		TasksByHours:     dto.NewTaskByHours(byHours),
	}, nil
}

func (s *Service) StatsExtended(ctx context.Context, start, end time.Time, companyID *int64) (*dto.StatsExtended, error) {
	byTypes, err := s.stats.ByUnitTypes(ctx, start, end, companyID)
	if err != nil {
		return nil, err
	}
	byDepts, err := s.stats.ByDepartments(ctx, start, end, companyID)
	if err != nil {
		return nil, err
	}
	perUser, err := s.stats.ByUnitTypesPerUser(ctx, start, end, companyID)
	if err != nil {
		return nil, err
	}
	calendar, err := s.stats.Calendar(ctx, start, end, companyID)
	if err != nil {
		return nil, err
	}
	return &dto.StatsExtended{
		ByDepartments:      dto.NewDeptStats(byDepts),
		ByUnitTypes:        dto.NewUnitTypeStats(byTypes),
		ByUnitTypesPerUser: dto.NewUnitTypesPerUser(perUser),
		Calendar:           dto.NewCalendar(calendar),
	}, nil
}

// StatsUserTasks — задачи с участием сотрудника. Гарды доступа: чужие
// данные — MANAGER+, и только в пределах активной компании.
func (s *Service) StatsUserTasks(ctx context.Context, actor *domain.User, targetUserID int64,
	start, end time.Time) (*dto.StatsUserTasks, error) {

	if targetUserID != actor.ID {
		if actor.RoleLevel < domain.LevelManager {
			return nil, domain.NewError("FORBIDDEN", "Доступ запрещён", 403)
		}
		target, err := s.users.GetUser(ctx, targetUserID)
		if err != nil {
			return nil, err
		}
		if target == nil {
			return nil, domain.NewError("NOT_FOUND", "Сотрудник не найден", 404)
		}
		// Менеджер/Администратор — только в своей (активной) компании: цель
		// должна состоять в ней (членство в user_companies).
		if actor.CompanyID != nil {
			member, err := s.users.IsCompanyMember(ctx, targetUserID, *actor.CompanyID)
			if err != nil {
				return nil, err
			}
			if !member {
				return nil, domain.NewError("FORBIDDEN", "Доступ запрещён", 403)
			}
		}
	}

	tasks, err := s.stats.UserTasksDetail(ctx, targetUserID, actor.CompanyID, start, end)
	if err != nil {
		return nil, err
	}
	out := &dto.StatsUserTasks{Tasks: []dto.UserTaskHours{}, TasksCount: len(tasks)}
	for _, t := range tasks {
		out.Tasks = append(out.Tasks, dto.UserTaskHours{
			TaskID: t.TaskID, TaskName: t.TaskName, TotalHours: t.TotalHours,
		})
	}
	return out, nil
}

func (s *Service) StatsProfile(ctx context.Context, userID int64, companyID *int64, start, end time.Time) (*dto.StatsProfile, error) {
	stats, err := s.stats.ProfileStats(ctx, userID, companyID, start, end)
	if err != nil {
		return nil, err
	}
	byTypes := make([]dto.UnitTypePerUser, 0, len(stats.ByUnitTypes))
	for _, ut := range stats.ByUnitTypes {
		byTypes = append(byTypes, dto.UnitTypePerUser{
			Hours: ut.Hours, Name: ut.Name, TasksCount: ut.TasksCount, TypeID: ut.TypeID,
		})
	}
	return &dto.StatsProfile{
		ByUnitTypes: byTypes,
		Period:      period(start, end),
		TasksCount:  stats.TasksCount,
		TotalHours:  stats.TotalHours,
	}, nil
}

func (s *Service) StatsEmployees(ctx context.Context, companyID *int64) ([]dto.EmployeeRef, error) {
	items, err := s.stats.VisibleEmployees(ctx, companyID)
	if err != nil {
		return nil, err
	}
	return dto.NewEmployeeRefs(items), nil
}

func (s *Service) StatsResponsibles(ctx context.Context, companyID *int64) ([]dto.ResponsibleDTO, error) {
	items, err := s.stats.Responsibles(ctx, companyID)
	if err != nil {
		return nil, err
	}
	return dto.NewResponsibles(items), nil
}
