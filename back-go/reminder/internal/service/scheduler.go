package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/reminder/internal/domain"
)

const (
	// tickInterval — как часто проверяем наступившие сроки. Минута — предел
	// точности самих напоминаний (секунды пользователь не задаёт), поэтому
	// частить незачем.
	tickInterval = 30 * time.Second
	// claimBatch — сколько напоминаний забираем за один проход.
	claimBatch = 200
)

// RunScheduler — цикл срабатывания напоминаний до отмены контекста.
//
// Забор наступивших сроков атомарен (репозиторий помечает их отработанными в
// том же запросе, которым выбирает), поэтому нескольким инстансам сервиса одно
// напоминание не достанется дважды. Следующий срок повтора считается ОТ
// ТЕКУЩЕГО МОМЕНТА: после простоя сервис не выстреливает пачкой пропущенных
// уведомлений, а встаёт на ближайший будущий срок.
func (s *Service) RunScheduler(ctx context.Context) error {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	s.fireDue(ctx) // не ждём первого тика: после рестарта сроки могли наступить
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.fireDue(ctx)
		}
	}
}

// fireDue — разослать все наступившие напоминания.
func (s *Service) fireDue(ctx context.Context) {
	now := time.Now()
	due, err := s.repo.ClaimDue(ctx, now, claimBatch)
	if err != nil {
		s.log.Warn("reminders.claim_failed", "error", err)
		return
	}
	for _, r := range due {
		s.fire(ctx, r, now)
	}
	if len(due) > 0 {
		s.log.Info("reminders.fired", "count", len(due))
	}
}

// fire — событие клиентам и перевод напоминания на следующий срок.
func (s *Service) fire(ctx context.Context, r *domain.Reminder, now time.Time) {
	firedAt := r.RemindAt
	r.LastFiredAt = &firedAt

	if next, ok := r.NextFire(now); ok {
		r.RemindAt = next
		r.Active = true
	} else {
		r.Active = false // разовое отработало — в журнал
	}
	if err := s.repo.Update(ctx, r); err != nil {
		// Событие всё равно отправляем: пользователю важнее получить
		// напоминание, чем аккуратность следующего срока (ClaimDue уже снял
		// его с очереди, повторной рассылки не будет).
		s.log.Warn("reminders.reschedule_failed", "id", r.ID, "error", err)
	}
	s.bus.Publish(ctx, "reminder:fire", []string{userRoom(r.OwnerID)}, map[string]any{
		"id":       r.ID,
		"owner_id": r.OwnerID,
		"title":    r.Title,
		"note":     r.Note,
		"fired_at": firedAt,
		"link":     r.Link,
		"next_at":  nextAt(r),
	})
	s.publish(ctx, "reminder:updated", r)
}

// nextAt — следующий срок для клиента (nil у отработавшего разового).
func nextAt(r *domain.Reminder) *time.Time {
	if !r.Active {
		return nil
	}
	t := r.RemindAt
	return &t
}
