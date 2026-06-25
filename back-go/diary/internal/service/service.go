// Package service — бизнес-логика diarysvc: личные ежедневники пользователя
// (заметки-задачи, привязанные к дню), их записи, архив выполненных и шаринг
// (публичной ссылкой и адресно, read-only). Скоуп — по владельцу (не по
// компании): ежедневник личный и кросс-компанийный. Сокет-события клиентам
// публикуются в Redis gw2:diary:events (доставляет gatewaysvc).
package service

import (
	"log/slog"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
)

type Service struct {
	repo  domain.DiaryRepository
	users domain.UserReader
	bus   domain.EventBus
	log   *slog.Logger
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

// requireOwned — ежедневник во владении пользователя или доменная 404. Для всех
// изменяющих операций и управления (правка, записи, шаринг).
func (s *Service) requireOwned(ctx domain.Ctx, userID, id int64) (*domain.Diary, error) {
	d, err := s.repo.GetDiary(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil || d.OwnerID != userID {
		return nil, domain.ErrDiaryNotFound
	}
	return d, nil
}

// requireReadable — ежедневник, доступный пользователю на чтение: свой (canEdit)
// или открытый адресно (read-only). Неизвестный/чужой без доступа — 404.
func (s *Service) requireReadable(ctx domain.Ctx, userID, id int64) (*domain.Diary, bool, error) {
	d, err := s.repo.GetDiary(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if d == nil {
		return nil, false, domain.ErrDiaryNotFound
	}
	if d.OwnerID == userID {
		return d, true, nil
	}
	ok, err := s.repo.HasMember(ctx, id, userID)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, domain.ErrDiaryNotFound
	}
	return d, false, nil
}

// diaryRooms — WS-комнаты доставки событий ежедневника: владелец + все, кому он
// открыт адресно. Так чужие read-only клиенты получают изменения в реальном
// времени, и при этом события не утекают посторонним.
func (s *Service) diaryRooms(ctx domain.Ctx, d *domain.Diary) []string {
	rooms := []string{userRoom(d.OwnerID)}
	ids, err := s.repo.MemberIDs(ctx, d.ID)
	if err != nil {
		s.log.Warn("diary.member_ids_failed", "diary", d.ID, "error", err)
		return rooms
	}
	for _, id := range ids {
		rooms = append(rooms, userRoom(id))
	}
	return rooms
}

func userRoom(id int64) string { return "user_" + strconv.FormatInt(id, 10) }
