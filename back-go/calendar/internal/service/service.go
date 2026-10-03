// Package service — бизнес-логика calendarsvc: календари, их поля (структура
// карточки) и записи, привязанные к дате/времени.
//
// Календарь лежит в пространстве — личном или команды (см. domain/access.go).
// Проверка прав — «какой у человека уровень», и она стоит на входе КАЖДОЙ
// операции. Сокет-события публикуются в Redis gw2:calendar:events (доставляет
// gatewaysvc) и адресуются аудитории календаря поимённо.
package service

import (
	"context"
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
)

type Service struct {
	repo  domain.CalendarRepository
	users domain.UserReader
	files domain.FileStore
	bus   domain.EventBus
	log   *slog.Logger
	// billing — лимиты тарифа (WithBilling; nil — ограничений нет).
	billing *billingclient.Client
}

type Deps struct {
	Repo  domain.CalendarRepository
	Users domain.UserReader
	Files domain.FileStore
	Bus   domain.EventBus
	Log   *slog.Logger
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, users: d.Users, files: d.Files, bus: d.Bus, log: d.Log}
}

/*
requireCalendar — календарь и проверка уровня доступа.

	Отсутствие доступа маскируем под 404: существование чужого календаря — само
	по себе сведения. НЕХВАТКУ уровня тому, кто календарь уже видит, называем
	честно, иначе «Сохранить» отвечало бы «не найдено» на открытой записи.
*/
func (s *Service) requireCalendar(ctx context.Context, userID, id int64, want string) (*domain.Calendar, error) {
	cal, err := s.repo.GetCalendar(ctx, id)
	if err != nil {
		return nil, err
	}
	if cal == nil {
		return nil, domain.ErrCalendarNotFound
	}
	access, err := s.repo.AccessOf(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if access == domain.AccessNone {
		return nil, domain.ErrCalendarNotFound
	}
	if !domain.AccessAtLeast(access, want) {
		if want == domain.AccessOwner {
			return nil, domain.ErrOwnerOnly
		}
		return nil, domain.ErrForbidden
	}
	cal.MyAccess = access
	return cal, nil
}

// quotaScope — чья квота платит за файлы календаря: файл календаря команды
// тратит место создателя команды, личного — самого хозяина.
func quotaScope(cal *domain.Calendar) (userID, companyID int64) {
	if cal.CompanyID != nil {
		return 0, *cal.CompanyID
	}
	return cal.OwnerID, 0
}

// audience — комнаты сокет-событий календаря.
func (s *Service) audience(ctx context.Context, calendarID int64) []string {
	ids, err := s.repo.Audience(ctx, calendarID)
	if err != nil {
		s.log.Warn("calendar.audience_failed", "calendar_id", calendarID, "error", err)
		return nil
	}
	rooms := make([]string, 0, len(ids))
	for _, id := range ids {
		rooms = append(rooms, events.UserRoom(id))
	}
	return rooms
}

// publish — событие аудитории календаря. Уровень доступа в полезной нагрузке
// не едет: у каждого получателя он свой.
func (s *Service) publish(ctx context.Context, calendarID int64, event string, payload any) {
	if rooms := s.audience(ctx, calendarID); len(rooms) > 0 {
		s.bus.Publish(ctx, event, rooms, payload)
	}
}

// requireMember — положить вещь в команду может только её участник.
func (s *Service) requireMember(ctx context.Context, userID int64, companyID *int64) error {
	if companyID == nil {
		return nil
	}
	role, err := s.users.TeamRole(ctx, userID, *companyID)
	if err != nil {
		return err
	}
	if !role.Member {
		return domain.ErrNotTeamMember
	}
	return nil
}
