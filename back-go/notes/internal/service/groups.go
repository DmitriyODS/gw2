package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

func (s *Service) ListGroups(ctx context.Context, userID int64) ([]*domain.Group, error) {
	return s.repo.ListGroups(ctx, userID)
}

func (s *Service) CreateGroup(ctx context.Context, userID int64, name string) (*domain.Group, error) {
	pos, err := s.repo.NextGroupPosition(ctx, userID)
	if err != nil {
		return nil, err
	}
	g := &domain.Group{OwnerID: userID, Name: name, Position: pos}
	if err := s.repo.CreateGroup(ctx, g); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "note_group:created", []string{userRoom(userID)}, g)
	return g, nil
}

func (s *Service) UpdateGroup(ctx context.Context, userID, id int64, name string) (*domain.Group, error) {
	g, err := s.requireGroupOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateGroup(ctx, id, name); err != nil {
		return nil, err
	}
	g.Name = name
	s.bus.Publish(ctx, "note_group:updated", []string{userRoom(userID)}, g)
	return g, nil
}

// DeleteGroup — удаляет группу и связи; заметки остаются (в «Все» и других
// своих группах).
func (s *Service) DeleteGroup(ctx context.Context, userID, id int64) error {
	if _, err := s.requireGroupOwned(ctx, userID, id); err != nil {
		return err
	}
	if err := s.repo.DeleteGroup(ctx, id); err != nil {
		return err
	}
	s.bus.Publish(ctx, "note_group:deleted", []string{userRoom(userID)}, map[string]any{"id": id, "owner_id": userID})
	return nil
}
