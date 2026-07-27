// Package service — бизнес-логика remindersvc: личные напоминания пользователя
// (разовые и повторяющиеся, со свободным текстом либо привязкой к записи
// ежедневника/календаря) и планировщик их срабатывания. Напоминание принадлежит
// одному пользователю и кросс-компанийно. Сработавшее уходит сокет-событием
// reminder:fire в Redis gw2:reminder:events: gatewaysvc доставляет его открытым
// вкладкам (тост + системное/десктопное уведомление), pushsvc — FCM-пушем тем,
// кто офлайн.
package service

import (
	"log/slog"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/reminder/internal/domain"
)

type Service struct {
	repo domain.ReminderRepository
	bus  domain.EventBus
	log  *slog.Logger
}

type Deps struct {
	Repo domain.ReminderRepository
	Bus  domain.EventBus
	Log  *slog.Logger
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, bus: d.Bus, log: d.Log}
}

// requireOwned — напоминание во владении пользователя или доменная 404 (чужое
// от несуществующего не отличаем — id не должен подсказывать существование).
func (s *Service) requireOwned(ctx domain.Ctx, userID, id int64) (*domain.Reminder, error) {
	r, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil || r.OwnerID != userID {
		return nil, domain.ErrNotFound
	}
	return r, nil
}

func userRoom(id int64) string { return "user_" + strconv.FormatInt(id, 10) }

// publish — событие напоминания в комнату его владельца (напоминания личные,
// аудитории у них нет).
func (s *Service) publish(ctx domain.Ctx, event string, r *domain.Reminder) {
	s.bus.Publish(ctx, event, []string{userRoom(r.OwnerID)}, r)
}
