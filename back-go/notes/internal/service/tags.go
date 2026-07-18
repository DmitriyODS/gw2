package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

func (s *Service) ListTags(ctx context.Context, userID int64) ([]*domain.Tag, error) {
	return s.repo.ListTags(ctx, userID)
}

func (s *Service) CreateTag(ctx context.Context, userID int64, name, color string) (*domain.Tag, error) {
	if color != "" && !domain.NoteColors[color] {
		return nil, domain.ErrBadColor
	}
	pos, err := s.repo.NextTagPosition(ctx, userID)
	if err != nil {
		return nil, err
	}
	t := &domain.Tag{OwnerID: userID, Name: name, Color: color, Position: pos}
	if err := s.repo.CreateTag(ctx, t); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "note_tag:created", []string{userRoom(userID)}, t)
	return t, nil
}

func (s *Service) UpdateTag(ctx context.Context, userID, id int64, name, color string) (*domain.Tag, error) {
	if color != "" && !domain.NoteColors[color] {
		return nil, domain.ErrBadColor
	}
	t, err := s.requireTagOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateTag(ctx, id, name, color); err != nil {
		return nil, err
	}
	t.Name, t.Color = name, color
	s.bus.Publish(ctx, "note_tag:updated", []string{userRoom(userID)}, t)
	return t, nil
}

// DeleteTag — удаляет тег и связи; заметки остаются.
func (s *Service) DeleteTag(ctx context.Context, userID, id int64) error {
	if _, err := s.requireTagOwned(ctx, userID, id); err != nil {
		return err
	}
	if err := s.repo.DeleteTag(ctx, id); err != nil {
		return err
	}
	s.bus.Publish(ctx, "note_tag:deleted", []string{userRoom(userID)}, map[string]any{"id": id, "owner_id": userID})
	return nil
}
