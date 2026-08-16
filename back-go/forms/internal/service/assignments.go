package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
)

/* Контроль исполнения и напоминания.

   Назначить форму — значит выдать доступ уровня respond человеку или целой
   компании. Отсюда два следствия: автор видит поимённо, кто ответил, а кто
   нет, а назначенным уходит одно напоминание перед сроком. */

// Progress — сводка исполнения: сколько назначено, сколько ответило и кто есть
// кто. Показывает вкладка «Назначения» у автора формы.
type Progress struct {
	Assigned  int                `json:"assigned"`
	Responded int                `json:"responded"`
	People    []*domain.Assignee `json:"people"`
}

// Progress — кто ответил, а кто ещё нет (уровень view и выше: это часть
// собранных ответов).
func (s *Service) Progress(ctx context.Context, userID, formID int64) (*Progress, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.require(ctx, a, formID, domain.AccessView); err != nil {
		return nil, err
	}
	people, err := s.repo.Assignees(ctx, formID)
	if err != nil {
		return nil, err
	}
	out := &Progress{Assigned: len(people), People: people}
	for _, p := range people {
		if p.AnsweredAt != nil {
			out.Responded++
		}
	}
	return out, nil
}

// remindTick — как часто планировщик заглядывает за наступившими сроками.
const remindTick = 5 * time.Minute

// remindBatch — сколько сроков забирается за один заход.
const remindBatch = 100

/*
RunReminders — планировщик напоминаний о сроке ответа.

	Срок забирается атомарно (ClaimDueReminders помечает строку напомненной тем
	же запросом), поэтому при нескольких инстансах сервиса напоминание уходит
	ровно один раз. Тем, кто уже ответил, не напоминаем вовсе — это и есть весь
	смысл контроля исполнения.
*/
func (s *Service) RunReminders(ctx context.Context) {
	ticker := time.NewTicker(remindTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.remindDue(ctx)
		}
	}
}

func (s *Service) remindDue(ctx context.Context) {
	due, err := s.repo.ClaimDueReminders(ctx, time.Now(), remindBatch)
	if err != nil {
		s.log.Warn("forms.claim_due_failed", "error", err)
		return
	}
	for _, item := range due {
		recipients := make([]int64, 0, len(item.UserIDs))
		for _, id := range item.UserIDs {
			if id != item.OwnerID {
				recipients = append(recipients, id)
			}
		}
		if len(recipients) == 0 {
			continue
		}
		s.bus.Publish(ctx, "form:due", rooms(recipients), map[string]any{
			"form_id": item.FormID, "title": item.FormTitle,
			"due_at": item.DueAt, "user_ids": recipients,
		})
	}
}
