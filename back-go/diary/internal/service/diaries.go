package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
)

// ListOwned — ежедневники пространств пользователя: личные и всех его команд
// (вкладка «Мои»).
func (s *Service) ListOwned(ctx context.Context, userID int64) ([]*domain.Diary, error) {
	return s.repo.ListOwned(ctx, userID)
}

// ListShared — чужие ежедневники, открытые пользователю адресно (вкладка
// «Поделились»), с именем владельца.
func (s *Service) ListShared(ctx context.Context, userID int64) ([]*domain.Diary, error) {
	return s.repo.ListShared(ctx, userID)
}

// GetDiary — один доступный ежедневник. Shared — чужой без права вести
// записи (фронт показывает read-only); CanCheck — можно ли отмечать записи.
func (s *Service) GetDiary(ctx context.Context, userID, id int64) (*domain.Diary, error) {
	return s.require(ctx, userID, id, domain.AccessView)
}

// MyDay — скрытый «Мой день» человека: в него ложатся дела экрана «Сегодня»
// и всё, что положено «во время» из других разделов.
func (s *Service) MyDay(ctx context.Context, userID int64) (*domain.Diary, error) {
	d, err := s.repo.MyDay(ctx, userID)
	if err != nil {
		return nil, err
	}
	d.MyAccess, d.CanCheck = domain.AccessOwner, true
	return d, nil
}

// CreateDiary — новый ежедневник в личном пространстве (companyID == nil)
// либо в команде, где автор состоит.
func (s *Service) CreateDiary(ctx context.Context, userID int64, companyID *int64, name string) (*domain.Diary, error) {
	if err := s.ensureLimit(ctx, userID); err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, userID, companyID); err != nil {
		return nil, err
	}
	pos, err := s.repo.NextPosition(ctx, userID)
	if err != nil {
		return nil, err
	}
	d := &domain.Diary{OwnerID: userID, CompanyID: companyID, TeamAccess: domain.AccessEdit,
		Name: name, Position: pos}
	if err := s.repo.CreateDiary(ctx, d); err != nil {
		return nil, err
	}
	d.MyAccess, d.CanCheck = domain.AccessOwner, true
	s.bus.Publish(ctx, "diary:created", s.diaryRooms(ctx, d), diaryPayload(d))
	return d, nil
}

// requireManaged — распоряжаться ежедневником (переименовать, удалить,
// раздать, перенести) может владелец; «Мой день» служебный и не меняется.
func (s *Service) requireManaged(ctx context.Context, userID, id int64) (*domain.Diary, error) {
	d, err := s.require(ctx, userID, id, domain.AccessOwner)
	if err != nil {
		return nil, err
	}
	if d.Kind == domain.KindMyDay {
		return nil, domain.ErrMyDayFixed
	}
	return d, nil
}

func (s *Service) UpdateDiary(ctx context.Context, userID, id int64, name string) (*domain.Diary, error) {
	d, err := s.requireManaged(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateDiary(ctx, id, name); err != nil {
		return nil, err
	}
	d.Name = name
	s.bus.Publish(ctx, "diary:updated", s.diaryRooms(ctx, d), diaryPayload(d))
	return d, nil
}

/*
MoveDiary — сменить пространство ежедневника и уровень участников команды.

	Распоряжается владелец; забирая ежедневник к себе, человек становится его
	хозяином. Кто доступ потерял, получает событие удаления.
*/
func (s *Service) MoveDiary(ctx context.Context, userID, id int64, companyID *int64, teamAccess string) (*domain.Diary, error) {
	d, err := s.requireManaged(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, userID, companyID); err != nil {
		return nil, err
	}
	before := s.diaryRooms(ctx, d)
	if companyID == nil {
		d.OwnerID = userID
	}
	d.CompanyID = companyID
	if teamAccess != "" {
		d.TeamAccess = domain.NormalizeTeamAccess(teamAccess)
	}
	if err := s.repo.MoveDiary(ctx, id, d.OwnerID, d.CompanyID, d.TeamAccess); err != nil {
		return nil, err
	}
	after := s.diaryRooms(ctx, d)
	if gone := missing(before, after); len(gone) > 0 {
		s.bus.Publish(ctx, "diary:deleted", gone, map[string]any{"id": id, "owner_id": d.OwnerID})
	}
	s.bus.Publish(ctx, "diary:updated", after, diaryPayload(d))
	return s.require(ctx, userID, id, domain.AccessView)
}

// missing — комнаты, которые были в before и пропали из after.
func missing(before, after []string) []string {
	kept := make(map[string]bool, len(after))
	for _, r := range after {
		kept[r] = true
	}
	out := []string{}
	for _, r := range before {
		if !kept[r] {
			out = append(out, r)
		}
	}
	return out
}

func (s *Service) DeleteDiary(ctx context.Context, userID, id int64) error {
	d, err := s.requireManaged(ctx, userID, id)
	if err != nil {
		return err
	}
	rooms := s.diaryRooms(ctx, d) // снимаем адресатов до удаления (каскад почистит связи)
	if err := s.repo.DeleteDiary(ctx, id); err != nil {
		return err
	}
	s.bus.Publish(ctx, "diary:deleted", rooms, map[string]any{"id": id, "owner_id": d.OwnerID})
	return nil
}

func diaryPayload(d *domain.Diary) map[string]any {
	return map[string]any{
		"id": d.ID, "owner_id": d.OwnerID, "company_id": d.CompanyID,
		"team_access": d.TeamAccess, "kind": d.Kind, "name": d.Name,
		"position": d.Position, "shared": d.Shared, "can_check": d.CanCheck,
		"owner_name": d.OwnerName, "owner_avatar": d.OwnerAvatar,
	}
}
