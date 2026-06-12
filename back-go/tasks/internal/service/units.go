package service

import (
	"context"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

var errUnitNotFound = domain.NewError("NOT_FOUND", "Юнит не найден", 404)

func (s *Service) TaskUnits(ctx context.Context, taskID int64) ([]dto.Unit, error) {
	task, err := s.tasks.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errTaskNotFound
	}
	units, err := s.units.UnitsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return dto.NewUnits(units), nil
}

// ActiveUnit — активный юнит пользователя; nil → JSON null (200).
func (s *Service) ActiveUnit(ctx context.Context, userID int64) (*dto.Unit, error) {
	unit, err := s.units.ActiveUnitForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if unit == nil {
		return nil, nil
	}
	out := dto.NewUnit(unit)
	return &out, nil
}

func (s *Service) CreateUnit(ctx context.Context, taskID, userID int64, name string, unitTypeID int64) (*dto.Unit, error) {
	task, err := s.tasks.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domain.NewError("TASK_NOT_FOUND", "Задача не найдена", 404)
	}
	if task.IsArchived {
		return nil, domain.NewError("TASK_ARCHIVED", "Нельзя создать юнит для архивной задачи", 422)
	}

	unitType, err := s.unitTypes.GetUnitType(ctx, unitTypeID)
	if err != nil {
		return nil, err
	}
	if unitType == nil {
		return nil, domain.NewError("TYPE_NOT_FOUND", "Тип юнита не найден", 404)
	}
	// Тип юнита должен принадлежать той же компании, что и задача — иначе
	// это нарушение multi-tenancy (нельзя «протащить» чужой тип к задаче).
	if unitType.CompanyID != task.CompanyID {
		return nil, domain.NewError("TYPE_FOREIGN", "Тип юнита принадлежит другой компании", 422)
	}

	active, err := s.units.ActiveUnitForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, domain.NewError("ACTIVE_UNIT_EXISTS", "У вас уже есть активный юнит", 409)
	}

	unit := &domain.Unit{
		Name: name, UserID: userID, UnitTypeID: unitTypeID,
		TaskID: taskID, CompanyID: task.CompanyID,
	}
	if err := s.units.CreateUnit(ctx, unit); err != nil {
		return nil, err
	}
	s.log.Info("unit.start", "unit_id", unit.ID, "task_id", taskID, "user_id", userID)
	s.groove.OnUnitStarted(unit, task.Name)

	// Перечитываем с user/unit_type для дампа и сокет-события.
	created, err := s.units.GetUnit(ctx, unit.ID)
	if err != nil {
		return nil, err
	}
	out := dto.NewUnit(created)
	s.bus.Publish(ctx, "unit:started", []string{roomAll}, out)
	return &out, nil
}

func (s *Service) UpdateUnit(ctx context.Context, unitID, actorID int64, actorLevel int, req dto.UnitUpdate) (*dto.Unit, error) {
	unit, err := s.units.GetUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	if unit == nil {
		return nil, errUnitNotFound
	}
	if unit.UserID != actorID && actorLevel < domain.LevelManager {
		return nil, domain.NewError("FORBIDDEN", "Недостаточно прав для редактирования чужого юнита", 403)
	}

	fields := map[string]any{}
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.UnitTypeID != nil {
		unitType, err := s.unitTypes.GetUnitType(ctx, *req.UnitTypeID)
		if err != nil {
			return nil, err
		}
		if unitType == nil {
			return nil, domain.NewError("TYPE_NOT_FOUND", "Тип юнита не найден", 404)
		}
		fields["unit_type_id"] = *req.UnitTypeID
	}
	if req.DatetimeStart != nil {
		fields["datetime_start"] = *req.DatetimeStart
	}
	if req.DatetimeEndSet {
		fields["datetime_end"] = req.DatetimeEnd
	}
	if err := s.units.UpdateUnitFields(ctx, unitID, fields); err != nil {
		return nil, err
	}

	updated, err := s.units.GetUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	out := dto.NewUnit(updated)
	// Payload как во Flask: дамп + явные unit_id/task_id.
	s.bus.Publish(ctx, "unit:updated", []string{roomAll}, struct {
		dto.Unit
		UnitID int64 `json:"unit_id"`
	}{Unit: out, UnitID: updated.ID})
	return &out, nil
}

func (s *Service) StopUnit(ctx context.Context, unitID, actorID int64, actorLevel int) (*dto.Unit, error) {
	unit, err := s.units.GetUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	if unit == nil {
		return nil, errUnitNotFound
	}
	if unit.DatetimeEnd != nil {
		return nil, domain.NewError("ALREADY_STOPPED", "Юнит уже завершён", 422)
	}
	if unit.UserID != actorID && actorLevel < domain.LevelManager {
		return nil, domain.NewError("FORBIDDEN", "Недостаточно прав для остановки чужого юнита", 403)
	}

	end, err := s.units.StopUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	unit.DatetimeEnd = &end
	s.log.Info("unit.stop", "unit_id", unitID, "user_id", actorID)

	taskName := ""
	if task, err := s.tasks.GetTask(ctx, unit.TaskID); err == nil && task != nil {
		taskName = task.Name
	}
	s.groove.OnUnitStopped(unit, taskName)

	s.bus.Publish(ctx, "unit:stopped", []string{roomAll}, map[string]any{
		"unit_id":      unit.ID,
		"task_id":      unit.TaskID,
		"user_id":      unit.UserID,
		"datetime_end": dto.ISO(end),
	})
	// Принудительная остановка чужого юнита — приватное уведомление владельцу.
	if unit.UserID != actorID {
		stopperFIO := "Неизвестный"
		if stopper, err := s.users.GetUser(ctx, actorID); err == nil && stopper != nil {
			stopperFIO = stopper.FIO
		}
		s.bus.Publish(ctx, "unit:force_stopped", []string{userRoom(unit.UserID)}, map[string]any{
			"unit_id":        unit.ID,
			"stopped_by_fio": stopperFIO,
		})
	}

	out := dto.NewUnit(unit)
	return &out, nil
}

func (s *Service) DeleteUnit(ctx context.Context, unitID, actorID int64, actorLevel int) error {
	unit, err := s.units.GetUnit(ctx, unitID)
	if err != nil {
		return err
	}
	if unit == nil {
		return errUnitNotFound
	}
	if unit.UserID != actorID && actorLevel < domain.LevelManager {
		return domain.NewError("FORBIDDEN", "Недостаточно прав для удаления чужого юнита", 403)
	}
	if err := s.units.DeleteUnit(ctx, unitID); err != nil {
		return err
	}
	s.log.Info("unit.delete", "unit_id", unitID, "task_id", unit.TaskID, "user_id", actorID)
	s.bus.Publish(ctx, "unit:deleted", []string{roomAll}, map[string]any{
		"unit_id": unitID, "task_id": unit.TaskID, "user_id": unit.UserID,
	})
	return nil
}

func userRoom(userID int64) string {
	return "user_" + strconv.FormatInt(userID, 10)
}
