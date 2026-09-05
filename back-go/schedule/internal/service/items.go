package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

// ItemInput — занятие целиком (правка занятия — всегда полная форма).
type ItemInput struct {
	Weekday     int
	StartMin    int
	EndMin      int
	Title       string
	Short       string
	CategoryID  *int64
	Weeks       []int
	RepeatEvery *int
	RepeatFrom  *time.Time
	RepeatUntil *time.Time
	Data        map[string]any
}

func (in ItemInput) apply(item *domain.Item) {
	item.Weekday = in.Weekday
	item.StartMin = in.StartMin
	item.EndMin = in.EndMin
	item.Title = in.Title
	item.Short = in.Short
	item.CategoryID = in.CategoryID
	item.Weeks = in.Weeks
	item.RepeatEvery = in.RepeatEvery
	item.RepeatFrom = in.RepeatFrom
	item.RepeatUntil = in.RepeatUntil
	item.Data = in.Data
}

func (s *Service) CreateItem(ctx context.Context, userID, scheduleID int64, in ItemInput) (*domain.Item, error) {
	sc, err := s.requireOwner(ctx, userID, scheduleID)
	if err != nil {
		return nil, err
	}
	item := &domain.Item{ScheduleID: sc.ID}
	in.apply(item)
	fields, err := s.prepareItem(ctx, sc, item)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateItem(ctx, item, domain.SearchText(fields, item)); err != nil {
		return nil, err
	}
	s.publish(ctx, sc.ID, "schedule_item:created", itemPayload(sc.ID, item))
	return item, nil
}

func (s *Service) UpdateItem(ctx context.Context, userID, scheduleID, itemID int64, in ItemInput) (*domain.Item, error) {
	sc, err := s.requireOwner(ctx, userID, scheduleID)
	if err != nil {
		return nil, err
	}
	item, err := s.repo.GetItem(ctx, sc.ID, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, domain.ErrItemNotFound
	}
	in.apply(item)
	fields, err := s.prepareItem(ctx, sc, item)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateItem(ctx, item, domain.SearchText(fields, item)); err != nil {
		return nil, err
	}
	s.publish(ctx, sc.ID, "schedule_item:updated", itemPayload(sc.ID, item))
	return item, nil
}

func (s *Service) DeleteItem(ctx context.Context, userID, scheduleID, itemID int64) error {
	sc, err := s.requireOwner(ctx, userID, scheduleID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteItem(ctx, sc.ID, itemID); err != nil {
		return err
	}
	s.publish(ctx, sc.ID, "schedule_item:deleted", map[string]any{"schedule_id": sc.ID, "id": itemID})
	return nil
}

// prepareItem — привести занятие к допустимому виду: проверить время и повтор,
// сверить категорию со справочником ЭТОГО расписания и оставить в значениях
// только известные поля. Возвращает поля расписания — по ним считается
// search_text.
func (s *Service) prepareItem(ctx context.Context, sc *domain.Schedule, item *domain.Item) ([]*domain.Field, error) {
	if err := item.Normalize(sc.CycleWeeks); err != nil {
		return nil, err
	}
	if item.CategoryID != nil {
		cats, err := s.repo.ListCategories(ctx, sc.ID)
		if err != nil {
			return nil, err
		}
		found := false
		for _, c := range cats {
			if c.ID == *item.CategoryID {
				found = true
				break
			}
		}
		// Чужая категория молча отбрасывается: занятие важнее её цвета, а
		// ссылку на справочник другого расписания хранить нельзя.
		if !found {
			item.CategoryID = nil
		}
	}
	fields, err := s.repo.ListFields(ctx, sc.ID)
	if err != nil {
		return nil, err
	}
	data, err := domain.CoerceData(fields, item.Data)
	if err != nil {
		return nil, err
	}
	item.Data = data
	return fields, nil
}

func itemPayload(scheduleID int64, item *domain.Item) map[string]any {
	return map[string]any{"schedule_id": scheduleID, "item": item}
}
