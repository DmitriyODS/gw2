package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

/* Структура расписания: справочник категорий (он же раскраска блоков) и набор
   дополнительных полей карточки занятия. И то и другое правит только владелец. */

// CategoryInput — категория: название и цвет из палитры --tag-*.
type CategoryInput struct {
	Name  string
	Color string
}

func (s *Service) CreateCategory(ctx context.Context, userID, scheduleID int64, in CategoryInput) (*domain.Category, error) {
	sc, err := s.requireOwner(ctx, userID, scheduleID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "Без названия"
	}
	pos, err := s.repo.NextCategoryPosition(ctx, sc.ID)
	if err != nil {
		return nil, err
	}
	c := &domain.Category{
		ScheduleID: sc.ID,
		Name:       trimRunes(name, 80),
		Color:      domain.NormalizeCategoryColor(in.Color),
		Position:   pos,
	}
	if err := s.repo.CreateCategory(ctx, c); err != nil {
		return nil, err
	}
	s.publish(ctx, sc.ID, "schedule:updated", schedulePayload(sc))
	return c, nil
}

func (s *Service) UpdateCategory(ctx context.Context, userID, scheduleID, id int64, in CategoryInput) (*domain.Category, error) {
	sc, err := s.requireOwner(ctx, userID, scheduleID)
	if err != nil {
		return nil, err
	}
	cats, err := s.repo.ListCategories(ctx, sc.ID)
	if err != nil {
		return nil, err
	}
	var current *domain.Category
	for _, c := range cats {
		if c.ID == id {
			current = c
			break
		}
	}
	if current == nil {
		return nil, domain.ErrCategoryNotFound
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		current.Name = trimRunes(name, 80)
	}
	current.Color = domain.NormalizeCategoryColor(in.Color)
	if err := s.repo.UpdateCategory(ctx, current); err != nil {
		return nil, err
	}
	s.publish(ctx, sc.ID, "schedule:updated", schedulePayload(sc))
	return current, nil
}

// DeleteCategory — занятия остаются, просто теряют категорию (ON DELETE SET NULL).
func (s *Service) DeleteCategory(ctx context.Context, userID, scheduleID, id int64) error {
	sc, err := s.requireOwner(ctx, userID, scheduleID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteCategory(ctx, sc.ID, id); err != nil {
		return err
	}
	s.publish(ctx, sc.ID, "schedule:updated", schedulePayload(sc))
	return nil
}

// ReplaceFields — полная замена набора полей. Значения отключённых полей
// вычищаются из занятий с пересчётом search_text: иначе по удалённому полю
// продолжал бы находиться поиск, а данные висели бы мёртвым грузом в data.
func (s *Service) ReplaceFields(ctx context.Context, userID, scheduleID int64, fields []*domain.Field) (*View, error) {
	sc, err := s.requireOwner(ctx, userID, scheduleID)
	if err != nil {
		return nil, err
	}
	for _, f := range fields {
		if err := domain.NormalizeField(f); err != nil {
			return nil, err
		}
	}
	removed, err := s.repo.ReplaceFields(ctx, sc.ID, fields)
	if err != nil {
		return nil, err
	}
	if len(removed) > 0 {
		if err := s.stripRemovedFields(ctx, sc.ID, fields, removed); err != nil {
			return nil, err
		}
	}
	s.publish(ctx, sc.ID, "schedule:updated", schedulePayload(sc))
	return s.view(ctx, sc, true)
}

func (s *Service) stripRemovedFields(ctx context.Context, scheduleID int64, fields []*domain.Field, removed []int64) error {
	items, err := s.repo.ListItems(ctx, scheduleID)
	if err != nil {
		return err
	}
	for _, item := range items {
		changed := false
		for _, fid := range removed {
			key := records.FieldID(fid)
			if _, ok := item.Data[key]; ok {
				delete(item.Data, key)
				changed = true
			}
		}
		if !changed {
			continue
		}
		if err := s.repo.UpdateItem(ctx, item, domain.SearchText(fields, item)); err != nil {
			return err
		}
	}
	return nil
}

func trimRunes(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		return string(r[:max])
	}
	return s
}
