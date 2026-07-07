package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// Хуки доменных событий tasksvc (gRPC, fire-and-forget со стороны
// вызывающего): геймификация не должна ломать основной флоу задач/юнитов —
// все ошибки только логируются.

type UnitHook struct {
	CompanyID int64
	UserID    int64
	UnitID    int64
	UnitName  string
	TaskID    int64
	TaskName  string
	Minutes   int
}

// OnUnitStarted — точка расширения контракта; питомец не реагирует на старт.
func (s *Service) OnUnitStarted(ctx context.Context, h UnitHook) {}

// OnUnitStopped — начисление кудосов/XP/лечения/квеста за завершённый юнит.
func (s *Service) OnUnitStopped(ctx context.Context, h UnitHook) {
	if !s.grooveEnabled(ctx, h.CompanyID) {
		return
	}
	// 1 кудос за каждые 5 минут работы; дневной кап источника «unit» = 15.
	s.AwardKudos(ctx, h.UserID, h.CompanyID, "unit", h.Minutes/5)
	// Работа растит питомца напрямую: XP за минуты юнита (кап в день,
	// сытость после кормления даёт множитель — см. AwardXP).
	s.AwardXP(ctx, h.UserID, h.CompanyID, "xp_unit",
		h.Minutes/domain.XPUnitMinutesPer, domain.XPUnitDailyCap)
	// Работа лечит больного питомца (совсем короткие юниты не считаются).
	if h.Minutes >= domain.RecoveryMinUnitMinutes {
		s.AddRecovery(ctx, h.UserID, h.CompanyID, 1)
	}
	// Дневной квест: завершённые юниты и минуты в фокусе.
	s.BumpQuest(ctx, h.UserID, "units_finished", 1)
	if h.Minutes > 0 {
		s.BumpQuest(ctx, h.UserID, "unit_minutes", h.Minutes)
	}
}

// OnTaskClosed — начисление кудосов/XP/лечения/квеста герою закрытия.
func (s *Service) OnTaskClosed(ctx context.Context, companyID, heroUserID,
	taskID int64, taskName string) {

	if !s.grooveEnabled(ctx, companyID) || heroUserID <= 0 {
		return
	}
	s.AwardKudos(ctx, heroUserID, companyID, "task_closed", 5)
	s.AwardXP(ctx, heroUserID, companyID, "xp_task",
		domain.XPTaskClosed, domain.XPTaskDailyCap)
	s.AddRecovery(ctx, heroUserID, companyID, 1)
	s.BumpQuest(ctx, heroUserID, "tasks_closed", 1)
}
