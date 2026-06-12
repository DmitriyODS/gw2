package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

// Хуки доменных событий других сервисов (gRPC, fire-and-forget со стороны
// вызывающего). Семантика прежних feed_service.on_*: геймификация не должна
// ломать основной флоу — все ошибки только логируются.

type UnitHook struct {
	CompanyID int64
	UserID    int64
	UnitID    int64
	UnitName  string
	TaskID    int64
	TaskName  string
	Minutes   int
}

func taskNamePayload(name string) any {
	if name == "" {
		return nil
	}
	return name
}

func (s *Service) OnUnitStarted(ctx context.Context, h UnitHook) {
	_, err := s.recordEvent(ctx, h.CompanyID, &h.UserID, "unit_started", map[string]any{
		"unit_id":   h.UnitID,
		"unit_name": h.UnitName,
		"task_id":   h.TaskID,
		"task_name": taskNamePayload(h.TaskName),
	}, false)
	if err != nil {
		s.log.Warn("groove.hook_failed", "hook", "unit_started", "error", err)
	}
}

func (s *Service) OnUnitStopped(ctx context.Context, h UnitHook) {
	_, err := s.recordEvent(ctx, h.CompanyID, &h.UserID, "unit_stopped", map[string]any{
		"unit_id":   h.UnitID,
		"unit_name": h.UnitName,
		"task_id":   h.TaskID,
		"task_name": taskNamePayload(h.TaskName),
		"minutes":   h.Minutes,
	}, false)
	if err != nil {
		s.log.Warn("groove.hook_failed", "hook", "unit_stopped", "error", err)
		return
	}
	// 1 грув за каждые 5 минут работы; дневной кап источника «unit» = 15.
	s.AwardBeans(ctx, h.UserID, h.CompanyID, "unit", h.Minutes/5)
	// Работа лечит больного Грувика (совсем короткие юниты не считаются).
	if h.Minutes >= domain.RecoveryMinUnitMinutes {
		s.AddRecovery(ctx, h.UserID, h.CompanyID, 1)
	}
	// Дневной квест: завершённые юниты и минуты в фокусе.
	s.BumpQuest(ctx, h.UserID, "units_finished", 1)
	if h.Minutes > 0 {
		s.BumpQuest(ctx, h.UserID, "unit_minutes", h.Minutes)
	}
}

func (s *Service) OnTaskClosed(ctx context.Context, companyID, heroUserID,
	taskID int64, taskName string) {

	var hero *int64
	if heroUserID > 0 {
		hero = &heroUserID
	}
	_, err := s.recordEvent(ctx, companyID, hero, "task_closed", map[string]any{
		"task_id":   taskID,
		"task_name": taskName,
	}, true)
	if err != nil {
		s.log.Warn("groove.hook_failed", "hook", "task_closed", "error", err)
		return
	}
	if heroUserID > 0 {
		s.AwardBeans(ctx, heroUserID, companyID, "task_closed", 5)
		s.AddRecovery(ctx, heroUserID, companyID, 1)
		s.BumpQuest(ctx, heroUserID, "tasks_closed", 1)
	}
	s.OnTaskClosedRaid(ctx, companyID)
}
