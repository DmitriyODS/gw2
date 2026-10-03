package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

// ListCalendars — личные календари и календари всех команд человека с полями
// (батч-загрузка без N+1).
func (s *Service) ListCalendars(ctx context.Context, userID int64) ([]*domain.Calendar, error) {
	cals, err := s.repo.ListCalendars(ctx, userID)
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

// GetCalendar — один доступный календарь с полями.
func (s *Service) GetCalendar(ctx context.Context, userID, id int64) (*domain.Calendar, error) {
	cal, err := s.requireCalendar(ctx, userID, id, domain.AccessView)
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

// CreateCalendar — новый календарь (структура полей задаётся отдельно) в личном
// пространстве (companyID == nil) либо в команде, где автор состоит.
func (s *Service) CreateCalendar(ctx context.Context, userID int64, companyID *int64, name string) (*domain.Calendar, error) {
	if err := s.ensureLimit(ctx, userID); err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, userID, companyID); err != nil {
		return nil, err
	}
	pos, err := s.repo.NextCalendarPosition(ctx, userID)
	if err != nil {
		return nil, err
	}
	cal := &domain.Calendar{
		OwnerID: userID, CompanyID: companyID, TeamAccess: domain.AccessEdit,
		Name: name, Position: pos, CreatedBy: &userID,
	}
	if err := s.repo.CreateCalendar(ctx, cal); err != nil {
		return nil, err
	}
	cal.Fields = []domain.Field{}
	cal.MyAccess = domain.AccessOwner
	s.publish(ctx, cal.ID, "calendar:created", calendarPayload(cal))
	return cal, nil
}

// UpdateCalendar — переименование (позиция не меняется). Правка структуры —
// уровень admin.
func (s *Service) UpdateCalendar(ctx context.Context, userID, id int64, name string) (*domain.Calendar, error) {
	cal, err := s.requireCalendar(ctx, userID, id, domain.AccessAdmin)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateCalendar(ctx, id, name, cal.Position); err != nil {
		return nil, err
	}
	cal.Name = name
	if cal.Fields, err = s.repo.ListFields(ctx, id); err != nil {
		return nil, err
	}
	s.publish(ctx, id, "calendar:updated", calendarPayload(cal))
	return cal, nil
}

/*
MoveCalendar — сменить пространство календаря и уровень участников команды.

	Распоряжается этим «владелец»: хозяин личного календаря, у календаря
	команды — автор и администраторы. Забирая календарь к себе, человек
	становится его хозяином. Учёт файлов переезжает ДО календаря: сверка
	«Хранилища» стёрла бы файлы, оставшиеся в журнале прежнего плательщика.
	Кто доступ потерял, получает событие удаления.
*/
func (s *Service) MoveCalendar(ctx context.Context, userID, id int64, companyID *int64, teamAccess string) (*domain.Calendar, error) {
	cal, err := s.requireCalendar(ctx, userID, id, domain.AccessOwner)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, userID, companyID); err != nil {
		return nil, err
	}
	next := *cal
	if companyID == nil {
		next.OwnerID = userID
	}
	next.CompanyID = companyID
	if teamAccess != "" {
		next.TeamAccess = domain.NormalizeTeamAccess(teamAccess)
	}

	before := s.audience(ctx, id)
	oldUser, oldCompany := quotaScope(cal)
	if newUser, newCompany := quotaScope(&next); newUser != oldUser || newCompany != oldCompany {
		if err := s.moveEntryFiles(ctx, &next); err != nil {
			return nil, domain.ErrMoveFailed
		}
	}
	if err := s.repo.MoveCalendar(ctx, id, next.OwnerID, next.CompanyID, next.TeamAccess); err != nil {
		return nil, err
	}
	if next.Fields, err = s.repo.ListFields(ctx, id); err != nil {
		return nil, err
	}
	after := s.audience(ctx, id)
	if gone := missing(before, after); len(gone) > 0 {
		s.bus.Publish(ctx, "calendar:deleted", gone, map[string]any{"id": id})
	}
	if len(after) > 0 {
		s.bus.Publish(ctx, "calendar:updated", after, calendarPayload(&next))
	}
	if next.MyAccess, err = s.repo.AccessOf(ctx, id, userID); err != nil {
		return nil, err
	}
	return &next, nil
}

// moveEntryFiles — переписать файлы записей на плательщика календаря cal.
func (s *Service) moveEntryFiles(ctx context.Context, cal *domain.Calendar) error {
	entries, err := s.repo.AllEntries(ctx, cal.ID)
	if err != nil {
		return err
	}
	var paths []string
	for _, e := range entries {
		for _, f := range records.DataFiles(e.Data) {
			paths = append(paths, f.Keys()...)
		}
	}
	userID, companyID := quotaScope(cal)
	return s.files.MoveFor(ctx, userID, companyID, paths)
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

// DeleteCalendar — только владелец (у календаря команды — автор или
// администратор команды).
func (s *Service) DeleteCalendar(ctx context.Context, userID, id int64) error {
	cal, err := s.requireCalendar(ctx, userID, id, domain.AccessOwner)
	if err != nil {
		return err
	}
	// Аудиторию и файлы узнаём ДО удаления: после него звать и чистить нечего.
	rooms := s.audience(ctx, id)
	entries, err := s.repo.AllEntries(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteCalendar(ctx, id); err != nil {
		return err
	}
	s.removeEntryFiles(ctx, cal, entries...)
	if len(rooms) > 0 {
		s.bus.Publish(ctx, "calendar:deleted", rooms, map[string]any{"id": id})
	}
	return nil
}

// ReplaceFields — полная замена набора полей. Отключённые (удалённые) поля
// вычищаются из данных всех записей с пересчётом search_text.
func (s *Service) ReplaceFields(ctx context.Context, userID, id int64, fields []domain.Field) (*domain.Calendar, error) {
	cal, err := s.requireCalendar(ctx, userID, id, domain.AccessAdmin)
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
		if err := s.stripRemovedFields(ctx, cal, fields, removed); err != nil {
			return nil, err
		}
	}
	cal.Fields = fields
	s.publish(ctx, id, "calendar:updated", calendarPayload(cal))
	return cal, nil
}

// stripRemovedFields — удалить значения отключённых полей из всех записей и
// пересчитать search_text по актуальному набору полей.
func (s *Service) stripRemovedFields(ctx context.Context, cal *domain.Calendar, fields []domain.Field, removed []int64) error {
	entries, err := s.repo.AllEntries(ctx, cal.ID)
	if err != nil {
		return err
	}
	var orphans []string
	for _, e := range entries {
		changed := false
		for _, fid := range removed {
			key := domain.FieldID(fid)
			if v, ok := e.Data[key]; ok {
				if p := fileValuePath(v); p != "" {
					orphans = append(orphans, p)
				}
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
	if len(orphans) > 0 {
		userID, companyID := quotaScope(cal)
		s.files.RemoveFor(ctx, userID, companyID, orphans)
	}
	return nil
}

func calendarPayload(c *domain.Calendar) map[string]any {
	return map[string]any{
		"id": c.ID, "owner_id": c.OwnerID, "company_id": c.CompanyID,
		"team_access": c.TeamAccess, "name": c.Name,
		"position": c.Position, "fields": c.Fields,
	}
}
