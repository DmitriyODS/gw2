// Package service — бизнес-логика schedulesvc: расписания, их структура
//
// (категории и поля), занятия и шаринг.
// Расписание лежит в пространстве — личном или команды (см. domain/access.go);
// адресно и публичной ссылкой оно открывается только на чтение. На входе
// каждой операции стоит проверка уровня.
// Сокет-события клиентам публикуются в Redis gw2:schedule:events (доставляет
// gatewaysvc) и адресуются аудитории расписания ПОИМЁННО: общей комнаты у
// раздела нет.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

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

// require — расписание и проверка уровня доступа (см. domain/access.go).
// Без доступа — 404: существование чужого расписания не раскрываем. Тому, кто
// расписание видит, нехватку уровня называем честно.
func (s *Service) require(ctx context.Context, a Actor, id int64, want string) (*domain.Schedule, error) {
	sc, err := s.repo.GetSchedule(ctx, id)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, domain.ErrScheduleNotFound
	}
	access, err := s.repo.AccessOf(ctx, id, a.UserID, a.Companies)
	if err != nil {
		return nil, err
	}
	if access == domain.AccessNone {
		return nil, domain.ErrScheduleNotFound
	}
	if !domain.AccessAtLeast(access, want) {
		if want == domain.AccessOwner {
			return nil, domain.ErrOwnerOnly
		}
		return nil, domain.ErrReadOnly
	}
	sc.MyAccess = access
	sc.Shared = !domain.AccessAtLeast(access, domain.AccessEdit)
	return sc, nil
}

// requireOwner — право вести расписание: структура, занятия, настройки.
func (s *Service) requireOwner(ctx context.Context, userID, id int64) (*domain.Schedule, error) {
	return s.requireLevel(ctx, userID, id, domain.AccessEdit)
}

// requireManage — распоряжаться расписанием: удалить, раздать, перенести.
func (s *Service) requireManage(ctx context.Context, userID, id int64) (*domain.Schedule, error) {
	return s.requireLevel(ctx, userID, id, domain.AccessOwner)
}

func (s *Service) requireLevel(ctx context.Context, userID, id int64, want string) (*domain.Schedule, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.require(ctx, a, id, want)
}

// requireRead — расписание, доступное на чтение; canEdit — можно ли его вести.
func (s *Service) requireRead(ctx context.Context, a Actor, id int64) (*domain.Schedule, bool, error) {
	sc, err := s.require(ctx, a, id, domain.AccessView)
	if err != nil {
		return nil, false, err
	}
	return sc, !sc.Shared, nil
}

// requireMember — положить расписание в команду может только её участник.
func (s *Service) requireMember(a Actor, companyID *int64) error {
	if companyID == nil || slices.Contains(a.Companies, *companyID) {
		return nil
	}
	return domain.ErrNotTeamMember
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
		"id": sc.ID, "owner_id": sc.OwnerID, "company_id": sc.CompanyID,
		"team_access": sc.TeamAccess, "name": sc.Name,
		"cycle_weeks": sc.CycleWeeks, "cycle_anchor": sc.CycleAnchor.Format(domain.DateLayout),
		"week_labels": sc.WeekLabels, "timezone": sc.Timezone, "gap_min": sc.GapMin,
		"item_count": sc.ItemCount, "updated_at": sc.UpdatedAt,
	}
}
