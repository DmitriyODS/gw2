package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
)

// ListOwned — личные ежедневники пользователя (вкладка «Мои»).
func (s *Service) ListOwned(ctx context.Context, userID int64) ([]*domain.Diary, error) {
	return s.repo.ListOwned(ctx, userID)
}

// ListShared — чужие ежедневники, открытые пользователю адресно (вкладка
// «Поделились»), read-only, с именем владельца.
func (s *Service) ListShared(ctx context.Context, userID int64) ([]*domain.Diary, error) {
	return s.repo.ListShared(ctx, userID)
}

// GetDiary — один ежедневник, доступный пользователю (свой или открытый адресно).
// Поле Shared проставляется для чужого, чтобы фронт показал read-only; CanCheck —
// можно ли отмечать записи выполненными.
func (s *Service) GetDiary(ctx context.Context, userID, id int64) (*domain.Diary, error) {
	d, canEdit, canCheck, err := s.requireReadable(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	d.Shared = !canEdit
	d.CanCheck = canCheck
	return d, nil
}

func (s *Service) CreateDiary(ctx context.Context, userID int64, name string) (*domain.Diary, error) {
	pos, err := s.repo.NextPosition(ctx, userID)
	if err != nil {
		return nil, err
	}
	d := &domain.Diary{OwnerID: userID, Name: name, Position: pos}
	if err := s.repo.CreateDiary(ctx, d); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "diary:created", []string{userRoom(userID)}, diaryPayload(d))
	return d, nil
}

func (s *Service) UpdateDiary(ctx context.Context, userID, id int64, name string) (*domain.Diary, error) {
	d, err := s.requireOwned(ctx, userID, id)
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

func (s *Service) DeleteDiary(ctx context.Context, userID, id int64) error {
	d, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return err
	}
	rooms := s.diaryRooms(ctx, d) // снимаем адресатов до удаления (каскад почистит связи)
	if err := s.repo.DeleteDiary(ctx, id); err != nil {
		return err
	}
	s.bus.Publish(ctx, "diary:deleted", rooms, map[string]any{"id": id, "owner_id": userID})
	return nil
}

func diaryPayload(d *domain.Diary) map[string]any {
	return map[string]any{
		"id": d.ID, "owner_id": d.OwnerID, "name": d.Name,
		"position": d.Position, "shared": d.Shared, "can_check": d.CanCheck,
		"owner_name": d.OwnerName, "owner_avatar": d.OwnerAvatar,
	}
}
