// Package service — бизнес-логика diarysvc: ежедневники (заметки-задачи,
// привязанные к дню), их записи, архив выполненных и шаринг (публичной ссылкой
// и адресно). Ежедневник лежит в пространстве — личном или команды; скрытый
// личный «Мой день» питает экран «Сегодня». Сокет-события клиентам публикуются
// в Redis gw2:diary:events (доставляет gatewaysvc).
package service

import (
	"log/slog"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

type Service struct {
	repo  domain.DiaryRepository
	users domain.UserReader
	bus   domain.EventBus
	log   *slog.Logger
	// billing — лимиты тарифа (WithBilling; nil — ограничений нет).
	billing *billingclient.Client
}

type Deps struct {
	Repo  domain.DiaryRepository
	Users domain.UserReader
	Bus   domain.EventBus
	Log   *slog.Logger
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, users: d.Users, bus: d.Bus, log: d.Log}
}

/*
require — ежедневник и проверка уровня доступа (см. domain/access.go).

	Отсутствие доступа маскируем под 404: существование чужого ежедневника —
	само по себе сведения. Нехватку уровня тому, кто ежедневник видит,
	называем честно: «только для чтения».
*/
func (s *Service) require(ctx domain.Ctx, userID, id int64, want string) (*domain.Diary, error) {
	d, err := s.repo.GetDiary(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, domain.ErrDiaryNotFound
	}
	access, err := s.repo.AccessOf(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if access == domain.AccessNone {
		return nil, domain.ErrDiaryNotFound
	}
	if !domain.AccessAtLeast(access, want) {
		if want == domain.AccessOwner {
			return nil, domain.ErrOwnerOnly
		}
		return nil, domain.ErrReadOnly
	}
	d.MyAccess = access
	d.CanCheck = domain.AccessAtLeast(access, domain.AccessCheck)
	d.Shared = !domain.AccessAtLeast(access, domain.AccessEdit)
	return d, nil
}

// diaryRooms — WS-комнаты доставки событий ежедневника: хозяин или участники
// команды плюс адресаты. События не утекают посторонним.
func (s *Service) diaryRooms(ctx domain.Ctx, d *domain.Diary) []string {
	ids, err := s.repo.Audience(ctx, d.ID)
	if err != nil {
		s.log.Warn("diary.audience_failed", "diary", d.ID, "error", err)
		return []string{userRoom(d.OwnerID)}
	}
	rooms := make([]string, 0, len(ids))
	for _, id := range ids {
		rooms = append(rooms, userRoom(id))
	}
	return rooms
}

// requireMember — положить ежедневник в команду может только её участник.
func (s *Service) requireMember(ctx domain.Ctx, userID int64, companyID *int64) error {
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

func userRoom(id int64) string { return "user_" + strconv.FormatInt(id, 10) }
