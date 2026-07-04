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

// OnUnitStarted — событие ленты больше не создаётся (доска признания без
// машинного спама); хук сохранён как точка расширения gRPC-контракта.
func (s *Service) OnUnitStarted(ctx context.Context, h UnitHook) {}

// OnUnitStopped — начисления за работу без записи в ленту: сводка дня
// публикуется одним событием day_summary (см. maybeDaySummary).
func (s *Service) OnUnitStopped(ctx context.Context, h UnitHook) {
	if !s.grooveEnabled(ctx, h.CompanyID) {
		return
	}
	// 1 грув за каждые 5 минут работы; дневной кап источника «unit» = 15.
	s.AwardBeans(ctx, h.UserID, h.CompanyID, "unit", h.Minutes/5)
	// Работа растит питомца напрямую: XP за минуты юнита (кап в день,
	// сытость после кормления даёт множитель — см. AwardXP).
	s.AwardXP(ctx, h.UserID, h.CompanyID, "xp_unit",
		h.Minutes/domain.XPUnitMinutesPer, domain.XPUnitDailyCap)
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

	if !s.grooveEnabled(ctx, companyID) {
		return
	}
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
		s.AwardXP(ctx, heroUserID, companyID, "xp_task",
			domain.XPTaskClosed, domain.XPTaskDailyCap)
		s.AddRecovery(ctx, heroUserID, companyID, 1)
		s.BumpQuest(ctx, heroUserID, "tasks_closed", 1)
	}
	s.OnTaskClosedRaid(ctx, companyID)
}
