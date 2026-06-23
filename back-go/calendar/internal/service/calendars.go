package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
)

// ListCalendars — календари компании с их полями (батч-загрузка без N+1).
func (s *Service) ListCalendars(ctx context.Context, companyID int64) ([]*domain.Calendar, error) {
	cals, err := s.repo.ListCalendars(ctx, companyID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(cals))
	for i, c := range cals {
		ids[i] = c.ID
	}
	byCal, err := s.repo.FieldsByCalendars(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, c := range cals {
		fields := byCal[c.ID]
		if fields == nil {
			fields = []domain.Field{}
		}
		c.Fields = fields
	}
	return cals, nil
}

// GetCalendar — один календарь компании с полями.
func (s *Service) GetCalendar(ctx context.Context, companyID, id int64) (*domain.Calendar, error) {
	cal, err := s.requireCalendar(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	fields, err := s.repo.ListFields(ctx, id)
	if err != nil {
		return nil, err
	}
	cal.Fields = fields
	return cal, nil
}

// CreateCalendar — новый календарь (структура полей задаётся отдельно).
func (s *Service) CreateCalendar(ctx context.Context, companyID, userID int64, name string) (*domain.Calendar, error) {
	pos, err := s.repo.NextCalendarPosition(ctx, companyID)
	if err != nil {
		return nil, err
	}
	cal := &domain.Calendar{CompanyID: companyID, Name: name, Position: pos, CreatedBy: &userID}
	if err := s.repo.CreateCalendar(ctx, cal); err != nil {
		return nil, err
	}
	cal.Fields = []domain.Field{}
	s.bus.Publish(ctx, "calendar:created", []string{roomAll}, calendarPayload(cal))
	return cal, nil
}

// UpdateCalendar — переименование (позиция не меняется).
func (s *Service) UpdateCalendar(ctx context.Context, companyID, id int64, name string) (*domain.Calendar, error) {
	cal, err := s.requireCalendar(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateCalendar(ctx, id, name, cal.Position); err != nil {
		return nil, err
	}
	cal.Name = name
	s.bus.Publish(ctx, "calendar:updated", []string{roomAll}, calendarPayload(cal))
	return cal, nil
}

func (s *Service) DeleteCalendar(ctx context.Context, companyID, id int64) error {
	if _, err := s.requireCalendar(ctx, companyID, id); err != nil {
		return err
	}
	if err := s.repo.DeleteCalendar(ctx, id); err != nil {
		return err
	}
	s.bus.Publish(ctx, "calendar:deleted", []string{roomAll}, map[string]any{
		"id": id, "company_id": companyID,
	})
	return nil
}

// ReplaceFields — полная замена набора полей. Отключённые (удалённые) поля
// вычищаются из данных всех записей с пересчётом search_text.
func (s *Service) ReplaceFields(ctx context.Context, companyID, id int64, fields []domain.Field) (*domain.Calendar, error) {
	cal, err := s.requireCalendar(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	for i := range fields {
		fields[i].CalendarID = id
		fields[i].Normalize()
	}
	removed, err := s.repo.ReplaceFields(ctx, id, fields)
	if err != nil {
		return nil, err
	}
	if len(removed) > 0 {
		if err := s.stripRemovedFields(ctx, id, fields, removed); err != nil {
			return nil, err
		}
	}
	cal.Fields = fields
	s.bus.Publish(ctx, "calendar:updated", []string{roomAll}, calendarPayload(cal))
	return cal, nil
}

// stripRemovedFields — удалить значения отключённых полей из всех записей и
// пересчитать search_text по актуальному набору полей.
func (s *Service) stripRemovedFields(ctx context.Context, calendarID int64, fields []domain.Field, removed []int64) error {
	entries, err := s.repo.AllEntries(ctx, calendarID)
	if err != nil {
		return err
	}
	for _, e := range entries {
		changed := false
		for _, fid := range removed {
			key := domain.FieldID(fid)
			if _, ok := e.Data[key]; ok {
				delete(e.Data, key)
				changed = true
			}
		}
		if !changed {
			continue
		}
		if err := s.repo.UpdateEntry(ctx, e.ID, nil, e.Data, buildSearchText(fields, e.Data)); err != nil {
			return err
		}
	}
	return nil
}

func calendarPayload(c *domain.Calendar) map[string]any {
	return map[string]any{
		"id": c.ID, "company_id": c.CompanyID, "name": c.Name,
		"position": c.Position, "fields": c.Fields,
	}
}
