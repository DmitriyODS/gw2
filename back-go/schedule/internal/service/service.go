// Package service — бизнес-логика schedulesvc: расписания, их структура
//
// (категории и поля), занятия и шаринг.
// Расписание принадлежит ЧЕЛОВЕКУ и не зависит от компании; другим оно доступно
// ТОЛЬКО НА ЧТЕНИЕ — публичной ссылкой или адресно. Поэтому проверок ролей
// компании здесь нет: на входе каждой операции стоит «владелец» либо «есть
// доступ на чтение».
// Сокет-события клиентам публикуются в Redis gw2:schedule:events (доставляет
// gatewaysvc) и адресуются аудитории расписания ПОИМЁННО: общей комнаты у
// раздела нет — расписание не является достоянием компании.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

type Service struct {
	repo  domain.ScheduleRepository
	users domain.UserReader
	bus   domain.EventBus
	log   *slog.Logger
}

type Deps struct {
	Repo  domain.ScheduleRepository
	Users domain.UserReader
	Bus   domain.EventBus
	Log   *slog.Logger
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, users: d.Users, bus: d.Bus, log: d.Log}
}

// Actor — кто выполняет операцию. Компании нужны для доступа через шары: их
// список считается один раз на запрос, а не на каждую проверку.
type Actor struct {
	UserID    int64
	Companies []int64
}

func (s *Service) actor(ctx context.Context, userID int64) (Actor, error) {
	companies, err := s.users.CompaniesOf(ctx, userID)
	if err != nil {
		return Actor{}, err
	}
	return Actor{UserID: userID, Companies: companies}, nil
}

// requireOwner — расписание во владении: правка структуры, занятий и шаринга.
// Чужое (и несуществующее) — 404: существование чужого расписания не
// раскрываем даже тому, кто его видит.
func (s *Service) requireOwner(ctx context.Context, userID, id int64) (*domain.Schedule, error) {
	sc, err := s.repo.GetSchedule(ctx, id)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, domain.ErrScheduleNotFound
	}
	if sc.OwnerID != userID {
		// Тому, кто расписание уже видит, называем причину честно: иначе
		// кнопка правки отвечала бы «не найдено» на открытом расписании.
		access, err := s.repo.HasAccess(ctx, id, userID, s.companiesOf(ctx, userID))
		if err != nil {
			return nil, err
		}
		if access {
			return nil, domain.ErrReadOnly
		}
		return nil, domain.ErrScheduleNotFound
	}
	return sc, nil
}

// requireRead — расписание, доступное на чтение: своё (canEdit) либо открытое
// адресно. Чужое без доступа — 404.
func (s *Service) requireRead(ctx context.Context, a Actor, id int64) (sc *domain.Schedule, canEdit bool, err error) {
	sc, err = s.repo.GetSchedule(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if sc == nil {
		return nil, false, domain.ErrScheduleNotFound
	}
	if sc.OwnerID == a.UserID {
		return sc, true, nil
	}
	ok, err := s.repo.HasAccess(ctx, id, a.UserID, a.Companies)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, domain.ErrScheduleNotFound
	}
	sc.Shared = true
	return sc, false, nil
}

// companiesOf — компании пользователя без падения: список нужен лишь для того,
// чтобы отличить «не видит» от «видит, но только читает».
func (s *Service) companiesOf(ctx context.Context, userID int64) []int64 {
	ids, err := s.users.CompaniesOf(ctx, userID)
	if err != nil {
		s.log.Warn("schedule.companies_failed", "user_id", userID, "error", err)
		return []int64{}
	}
	return ids
}

// publish — событие аудитории расписания.
func (s *Service) publish(ctx context.Context, scheduleID int64, event string, payload any) {
	ids, err := s.repo.Audience(ctx, scheduleID)
	if err != nil {
		s.log.Warn("schedule.audience_failed", "schedule_id", scheduleID, "error", err)
		return
	}
	if len(ids) == 0 {
		return
	}
	rooms := make([]string, 0, len(ids))
	for _, id := range ids {
		rooms = append(rooms, fmt.Sprintf("user_%d", id))
	}
	s.bus.Publish(ctx, event, rooms, payload)
}

// schedulePayload — снимок расписания для сокет-события. Занятия не тащим:
// открытый экран перечитывает их сам.
func schedulePayload(sc *domain.Schedule) map[string]any {
	return map[string]any{
		"id": sc.ID, "owner_id": sc.OwnerID, "name": sc.Name,
		"cycle_weeks": sc.CycleWeeks, "cycle_anchor": sc.CycleAnchor.Format(domain.DateLayout),
		"week_labels": sc.WeekLabels, "timezone": sc.Timezone, "gap_min": sc.GapMin,
		"item_count": sc.ItemCount, "updated_at": sc.UpdatedAt,
	}
}
