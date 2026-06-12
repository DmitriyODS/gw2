package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// ── Типы юнитов ──────────────────────────────────────────────────

func (s *Service) ListUnitTypes(ctx context.Context, companyID int64) ([]dto.UnitTypeDTO, error) {
	items, err := s.unitTypes.ListUnitTypes(ctx, companyID)
	if err != nil {
		return nil, err
	}
	return dto.NewUnitTypes(items), nil
}

func (s *Service) CreateUnitType(ctx context.Context, companyID int64, name string) (*dto.UnitTypeDTO, error) {
	existing, err := s.unitTypes.GetUnitTypeByName(ctx, name, companyID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, domain.NewError("DUPLICATE", "Тип юнита с таким именем уже существует", 409)
	}
	ut := &domain.UnitType{Name: name, CompanyID: companyID}
	if err := s.unitTypes.CreateUnitType(ctx, ut); err != nil {
		return nil, err
	}
	return &dto.UnitTypeDTO{ID: ut.ID, Name: ut.Name}, nil
}

func (s *Service) UpdateUnitType(ctx context.Context, companyID, typeID int64, name string) (*dto.UnitTypeDTO, error) {
	ut, err := s.unitTypes.GetUnitType(ctx, typeID)
	if err != nil {
		return nil, err
	}
	if ut == nil || ut.CompanyID != companyID {
		return nil, domain.NewError("NOT_FOUND", "Тип юнита не найден", 404)
	}
	existing, err := s.unitTypes.GetUnitTypeByName(ctx, name, companyID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID != typeID {
		return nil, domain.NewError("DUPLICATE", "Тип юнита с таким именем уже существует", 409)
	}
	if err := s.unitTypes.UpdateUnitTypeName(ctx, typeID, name); err != nil {
		return nil, err
	}
	return &dto.UnitTypeDTO{ID: typeID, Name: name}, nil
}

// DeleteUnitType — каскадно удаляет все юниты с этим типом (FK CASCADE).
func (s *Service) DeleteUnitType(ctx context.Context, companyID, typeID int64) error {
	ut, err := s.unitTypes.GetUnitType(ctx, typeID)
	if err != nil {
		return err
	}
	if ut == nil || ut.CompanyID != companyID {
		return domain.NewError("NOT_FOUND", "Тип юнита не найден", 404)
	}
	return s.unitTypes.DeleteUnitType(ctx, typeID)
}

// ── Отделы ───────────────────────────────────────────────────────

func (s *Service) ListDepartments(ctx context.Context, companyID int64) ([]dto.DepartmentDTO, error) {
	items, err := s.depts.ListDepartments(ctx, companyID)
	if err != nil {
		return nil, err
	}
	return dto.NewDepartments(items), nil
}

func (s *Service) CreateDepartment(ctx context.Context, companyID int64, name string) (*dto.DepartmentDTO, error) {
	existing, err := s.depts.GetDepartmentByName(ctx, name, companyID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, domain.NewError("DUPLICATE", "Отдел с таким именем уже существует", 409)
	}
	d := &domain.Department{Name: name, CompanyID: companyID}
	if err := s.depts.CreateDepartment(ctx, d); err != nil {
		return nil, err
	}
	return &dto.DepartmentDTO{ID: d.ID, Name: d.Name}, nil
}

func (s *Service) UpdateDepartment(ctx context.Context, companyID, deptID int64, name string) (*dto.DepartmentDTO, error) {
	dept, err := s.depts.GetDepartment(ctx, deptID)
	if err != nil {
		return nil, err
	}
	if dept == nil || dept.CompanyID != companyID {
		return nil, domain.NewError("NOT_FOUND", "Отдел не найден", 404)
	}
	existing, err := s.depts.GetDepartmentByName(ctx, name, companyID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID != deptID {
		return nil, domain.NewError("DUPLICATE", "Отдел с таким именем уже существует", 409)
	}
	if err := s.depts.UpdateDepartmentName(ctx, deptID, name); err != nil {
		return nil, err
	}
	return &dto.DepartmentDTO{ID: deptID, Name: name}, nil
}

func (s *Service) DeleteDepartment(ctx context.Context, companyID, deptID int64) error {
	dept, err := s.depts.GetDepartment(ctx, deptID)
	if err != nil {
		return err
	}
	if dept == nil || dept.CompanyID != companyID {
		return domain.NewError("NOT_FOUND", "Отдел не найден", 404)
	}
	return s.depts.DeleteDepartment(ctx, deptID)
}

// ── Этапы ────────────────────────────────────────────────────────

func (s *Service) ListStages(ctx context.Context, companyID int64) ([]dto.StageDTO, error) {
	items, err := s.stages.ListStages(ctx, companyID)
	if err != nil {
		return nil, err
	}
	return dto.NewStages(items), nil
}

func (s *Service) CreateStage(ctx context.Context, companyID int64, name, color string) (*dto.StageDTO, error) {
	existing, err := s.stages.GetStageByName(ctx, name, companyID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, domain.NewError("DUPLICATE", "Этап с таким именем уже существует", 409)
	}
	order, err := s.stages.NextStageOrder(ctx, companyID)
	if err != nil {
		return nil, err
	}
	stage := &domain.Stage{Name: name, Color: color, CompanyID: companyID, Order: order}
	if err := s.stages.CreateStage(ctx, stage); err != nil {
		return nil, err
	}
	out := dto.NewStage(stage)
	return &out, nil
}

func (s *Service) UpdateStage(ctx context.Context, companyID, stageID int64, name, color *string) (*dto.StageDTO, error) {
	stage, err := s.stages.GetStage(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil || stage.CompanyID != companyID {
		return nil, domain.NewError("NOT_FOUND", "Этап не найден", 404)
	}
	if name != nil && *name != stage.Name {
		existing, err := s.stages.GetStageByName(ctx, *name, companyID)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != stageID {
			return nil, domain.NewError("DUPLICATE", "Этап с таким именем уже существует", 409)
		}
	}
	fields := map[string]any{}
	if name != nil {
		fields["name"] = *name
		stage.Name = *name
	}
	if color != nil {
		fields["color"] = *color
		stage.Color = *color
	}
	if err := s.stages.UpdateStageFields(ctx, stageID, fields); err != nil {
		return nil, err
	}
	out := dto.NewStage(stage)
	return &out, nil
}

func (s *Service) DeleteStage(ctx context.Context, companyID, stageID int64) error {
	stage, err := s.stages.GetStage(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil || stage.CompanyID != companyID {
		return domain.NewError("NOT_FOUND", "Этап не найден", 404)
	}
	return s.stages.DeleteStage(ctx, stageID)
}

func (s *Service) ReorderStages(ctx context.Context, companyID int64, orderedIDs []int64) ([]dto.StageDTO, error) {
	if err := s.stages.ReorderStages(ctx, companyID, orderedIDs); err != nil {
		return nil, err
	}
	return s.ListStages(ctx, companyID)
}
