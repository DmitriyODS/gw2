// Package service — бизнес-логика calendarsvc: календари компании, их поля
// (структура карточки) и записи, привязанные к дате/времени. Структуру правит
// администратор компании, записи — любой её участник. Сокет-события клиентам
// публикуются в Redis gw2:calendar:events (доставляет gatewaysvc).
package service

import (
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
)

const roomAll = "all"

type Service struct {
	repo  domain.CalendarRepository
	files domain.FileStore
	bus   domain.EventBus
	log   *slog.Logger
}

type Deps struct {
	Repo  domain.CalendarRepository
	Files domain.FileStore
	Bus   domain.EventBus
	Log   *slog.Logger
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, files: d.Files, bus: d.Bus, log: d.Log}
}

// requireCalendar — календарь активной компании или доменная 404.
func (s *Service) requireCalendar(ctx domain.Ctx, companyID, id int64) (*domain.Calendar, error) {
	cal, err := s.repo.GetCalendar(ctx, id)
	if err != nil {
		return nil, err
	}
	if cal == nil || cal.CompanyID != companyID {
		return nil, domain.ErrCalendarNotFound
	}
	return cal, nil
}
