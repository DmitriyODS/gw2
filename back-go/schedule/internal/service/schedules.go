package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

// View — расписание целиком: настройки, структура и ВСЕ занятия. Их десятки,
// поэтому отдаём одним запросом, а по неделям фильтрует клиент — правило
// повтора у него зеркальное (front/src/utils/scheduleCycle.js).
type View struct {
	Schedule *domain.Schedule `json:"schedule"`
	Items    []*domain.Item   `json:"items"`
	CanEdit  bool             `json:"can_edit"`
}

// ScheduleInput — создание расписания.
type ScheduleInput struct {
	Name        string
	CycleWeeks  int
	CycleAnchor time.Time
	Timezone    string
}

// ScheduleUpdate — частичная правка: nil-поля не меняются.
type ScheduleUpdate struct {
	Name        *string
	CycleWeeks  *int
	CycleAnchor *time.Time
	WeekLabels  []string
	Timezone    *string
	GapMin      *int
}

// ListSchedules — свои расписания и открытые адресно, вместе со структурой
// (категории и поля приезжают пачкой — списку слева нужны цвета).
func (s *Service) ListSchedules(ctx context.Context, userID int64, shared bool) ([]*domain.Schedule, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	var list []*domain.Schedule
	if shared {
		list, err = s.repo.ListShared(ctx, a.UserID, a.Companies)
	} else {
		list, err = s.repo.ListOwned(ctx, a.UserID)
	}
	if err != nil {
		return nil, err
	}
	return list, s.attachStructure(ctx, list)
}

// attachStructure — категории и поля для списка расписаний двумя запросами
// вместо двух на каждое расписание.
func (s *Service) attachStructure(ctx context.Context, list []*domain.Schedule) error {
	ids := make([]int64, 0, len(list))
	for _, sc := range list {
		ids = append(ids, sc.ID)
	}
	cats, err := s.repo.CategoriesBySchedules(ctx, ids)
	if err != nil {
		return err
	}
	fields, err := s.repo.FieldsBySchedules(ctx, ids)
	if err != nil {
		return err
	}
	for _, sc := range list {
		sc.Categories = orEmptyCategories(cats[sc.ID])
		sc.Fields = orEmptyFields(fields[sc.ID])
	}
	return nil
}

func orEmptyCategories(v []*domain.Category) []*domain.Category {
	if v == nil {
		return []*domain.Category{}
	}
	return v
}

func orEmptyFields(v []*domain.Field) []*domain.Field {
	if v == nil {
		return []*domain.Field{}
	}
	return v
}

// GetSchedule — расписание с занятиями (экран раздела).
func (s *Service) GetSchedule(ctx context.Context, userID, id int64) (*View, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	sc, canEdit, err := s.requireRead(ctx, a, id)
	if err != nil {
		return nil, err
	}
	return s.view(ctx, sc, canEdit)
}

func (s *Service) view(ctx context.Context, sc *domain.Schedule, canEdit bool) (*View, error) {
	cats, err := s.repo.ListCategories(ctx, sc.ID)
	if err != nil {
		return nil, err
	}
	fields, err := s.repo.ListFields(ctx, sc.ID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ListItems(ctx, sc.ID)
	if err != nil {
		return nil, err
	}
	sc.Categories = orEmptyCategories(cats)
	sc.Fields = orEmptyFields(fields)
	return &View{Schedule: sc, Items: items, CanEdit: canEdit}, nil
}

// CreateSchedule — новое расписание. Якорь цикла приводится к понедельнику:
// цикл считается неделями, и середина недели точкой отсчёта быть не может.
func (s *Service) CreateSchedule(ctx context.Context, userID int64, in ScheduleInput) (*View, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, domain.ErrNameRequired
	}
	pos, err := s.repo.NextPosition(ctx, userID)
	if err != nil {
		return nil, err
	}
	weeks := domain.NormalizeCycleWeeks(in.CycleWeeks)
	anchor := in.CycleAnchor
	if anchor.IsZero() {
		anchor = time.Now().UTC()
	}
	sc := &domain.Schedule{
		OwnerID:     userID,
		Name:        name,
		CycleWeeks:  weeks,
		CycleAnchor: domain.MondayOf(anchor),
		WeekLabels:  domain.NormalizeLabels(nil, weeks),
		Timezone:    normalizeTimezone(in.Timezone),
		GapMin:      20,
		Position:    pos,
	}
	if err := s.repo.CreateSchedule(ctx, sc); err != nil {
		return nil, err
	}
	s.publish(ctx, sc.ID, "schedule:created", schedulePayload(sc))
	return s.view(ctx, sc, true)
}

// UpdateSchedule — правка настроек. Возвращает, сколько занятий пришлось
// тронуть: при УКОРОЧЕНИИ цикла номера недель за его пределами отбрасываются,
// и занятие, у которого не осталось ни одной, становится еженедельным. Молча
// пропасть со шкалы оно не должно, поэтому число уезжает клиенту — он говорит
// об этом человеку.
func (s *Service) UpdateSchedule(ctx context.Context, userID, id int64, up ScheduleUpdate) (*View, int, error) {
	sc, err := s.requireOwner(ctx, userID, id)
	if err != nil {
		return nil, 0, err
	}
	if up.Name != nil {
		name := strings.TrimSpace(*up.Name)
		if name == "" {
			return nil, 0, domain.ErrNameRequired
		}
		sc.Name = name
	}
	cycleChanged := false
	if up.CycleWeeks != nil {
		weeks := domain.NormalizeCycleWeeks(*up.CycleWeeks)
		cycleChanged = weeks != sc.CycleWeeks
		sc.CycleWeeks = weeks
	}
	if up.CycleAnchor != nil {
		sc.CycleAnchor = domain.MondayOf(*up.CycleAnchor)
	}
	if up.WeekLabels != nil {
		sc.WeekLabels = up.WeekLabels
	}
	if up.Timezone != nil {
		sc.Timezone = normalizeTimezone(*up.Timezone)
	}
	if up.GapMin != nil {
		sc.GapMin = domain.NormalizeGap(*up.GapMin)
	}
	// Названия недель всегда живут длиной цикла: лишние отбрасываются,
	// недостающие добираются пустыми.
	sc.WeekLabels = domain.NormalizeLabels(sc.WeekLabels, sc.CycleWeeks)

	if err := s.repo.UpdateSchedule(ctx, sc); err != nil {
		return nil, 0, err
	}
	adjusted := 0
	if cycleChanged {
		if adjusted, err = s.repo.NormalizeItemWeeks(ctx, sc.ID, sc.CycleWeeks); err != nil {
			return nil, 0, err
		}
	}
	s.publish(ctx, sc.ID, "schedule:updated", schedulePayload(sc))
	view, err := s.view(ctx, sc, true)
	return view, adjusted, err
}

func (s *Service) DeleteSchedule(ctx context.Context, userID, id int64) error {
	sc, err := s.requireOwner(ctx, userID, id)
	if err != nil {
		return err
	}
	// Событие уходит ДО удаления: после него аудитории уже не собрать.
	s.publish(ctx, sc.ID, "schedule:deleted", map[string]any{"id": sc.ID})
	return s.repo.DeleteSchedule(ctx, id)
}

// normalizeTimezone — зона расписания (IANA). Незнакомая заменяется московской:
// по ней считаются «сегодня» и «сейчас», и молчаливый UTC сдвинул бы день.
func normalizeTimezone(tz string) string {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return "Europe/Moscow"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return "Europe/Moscow"
	}
	return tz
}
