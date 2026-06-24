package service

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
)

// EntryList — выборка записей календаря за период.
type EntryList struct {
	Items []*domain.Entry `json:"items"`
}

// EntryListParams — сырые параметры запроса (из query-строки).
type EntryListParams struct {
	Search string
	From   *time.Time
	To     *time.Time
}

const entriesLimit = 2000

// ListEntries — записи календаря за диапазон дат (для просмотра дня/недели/
// месяца) с опциональным сквозным поиском.
func (s *Service) ListEntries(ctx context.Context, companyID, calendarID int64, p EntryListParams) (*EntryList, error) {
	if _, err := s.requireCalendar(ctx, companyID, calendarID); err != nil {
		return nil, err
	}
	return s.listEntriesByCalendar(ctx, calendarID, p)
}

// listEntriesByCalendar — ядро выборки (без проверки доступа; вызывающий уже
// проверил права или resolveShare). Используется и authed, и публичным доступом.
func (s *Service) listEntriesByCalendar(ctx context.Context, calendarID int64, p EntryListParams) (*EntryList, error) {
	items, err := s.repo.ListEntries(ctx, domain.EntryListFilter{
		CalendarID: calendarID,
		Search:     strings.TrimSpace(p.Search),
		From:       p.From,
		To:         p.To,
		Limit:      entriesLimit,
	})
	if err != nil {
		return nil, err
	}
	return &EntryList{Items: items}, nil
}

func (s *Service) GetEntry(ctx context.Context, companyID, calendarID, entryID int64) (*domain.Entry, error) {
	if _, err := s.requireCalendar(ctx, companyID, calendarID); err != nil {
		return nil, err
	}
	e, err := s.repo.GetEntry(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if e == nil || e.CalendarID != calendarID {
		return nil, domain.ErrEntryNotFound
	}
	return e, nil
}

func (s *Service) CreateEntry(ctx context.Context, companyID, calendarID, userID int64, eventAt time.Time, data map[string]any) (*domain.Entry, error) {
	if _, err := s.requireCalendar(ctx, companyID, calendarID); err != nil {
		return nil, err
	}
	if eventAt.IsZero() {
		return nil, domain.ErrEventAtRequired
	}
	fields, err := s.repo.ListFields(ctx, calendarID)
	if err != nil {
		return nil, err
	}
	clean, err := coerceData(fields, data)
	if err != nil {
		return nil, err
	}
	e := &domain.Entry{CalendarID: calendarID, EventAt: eventAt.Truncate(time.Minute), Data: clean, CreatedBy: &userID}
	if err := s.repo.CreateEntry(ctx, e, buildSearchText(fields, clean)); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "entry:created", []string{roomAll}, entryPayload(companyID, e))
	return e, nil
}

func (s *Service) UpdateEntry(ctx context.Context, companyID, calendarID, entryID int64, eventAt time.Time, data map[string]any) (*domain.Entry, error) {
	e, err := s.GetEntry(ctx, companyID, calendarID, entryID)
	if err != nil {
		return nil, err
	}
	if eventAt.IsZero() {
		return nil, domain.ErrEventAtRequired
	}
	fields, err := s.repo.ListFields(ctx, calendarID)
	if err != nil {
		return nil, err
	}
	clean, err := coerceData(fields, data)
	if err != nil {
		return nil, err
	}
	at := eventAt.Truncate(time.Minute)
	if err := s.repo.UpdateEntry(ctx, entryID, at, clean, buildSearchText(fields, clean)); err != nil {
		return nil, err
	}
	e.EventAt = at
	e.Data = clean
	s.bus.Publish(ctx, "entry:updated", []string{roomAll}, entryPayload(companyID, e))
	return e, nil
}

func (s *Service) DeleteEntry(ctx context.Context, companyID, calendarID, entryID int64) error {
	e, err := s.GetEntry(ctx, companyID, calendarID, entryID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteEntry(ctx, entryID); err != nil {
		return err
	}
	s.removeEntryFiles(e)
	s.bus.Publish(ctx, "entry:deleted", []string{roomAll}, map[string]any{
		"id": entryID, "calendar_id": calendarID, "company_id": companyID,
	})
	return nil
}

// DeleteEntries — массовое удаление выбранных записей.
func (s *Service) DeleteEntries(ctx context.Context, companyID, calendarID int64, ids []int64) (int64, error) {
	if _, err := s.requireCalendar(ctx, companyID, calendarID); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	// Снимаем файлы записей до удаления — после DELETE данные уже недоступны.
	entries, _ := s.repo.EntriesForExport(ctx, domain.EntryListFilter{CalendarID: calendarID}, ids)
	n, err := s.repo.DeleteEntries(ctx, calendarID, ids)
	if err != nil {
		return 0, err
	}
	s.removeEntryFiles(entries...)
	s.bus.Publish(ctx, "entry:bulk-deleted", []string{roomAll}, map[string]any{
		"ids": ids, "calendar_id": calendarID, "company_id": companyID,
	})
	return n, nil
}

// removeEntryFiles — удалить из хранилища файлы/картинки удаляемых записей.
func (s *Service) removeEntryFiles(entries ...*domain.Entry) {
	var paths []string
	for _, e := range entries {
		if e == nil {
			continue
		}
		for _, v := range e.Data {
			if p := fileValuePath(v); p != "" {
				paths = append(paths, p)
			}
		}
	}
	if len(paths) > 0 {
		s.files.Remove(paths)
	}
}

// fileValuePath — путь файла/картинки из значения поля. UploadedFile хранится
// как объект с ключом "path"; для прочих типов — пусто.
func fileValuePath(v any) string {
	if m, ok := v.(map[string]any); ok {
		if p, ok := m["path"].(string); ok {
			return p
		}
	}
	return ""
}

// ── Хелперы ──────────────────────────────────────────────────────

// buildSearchText — единая строка для поиска (текст/число/дата/список/ссылка).
func buildSearchText(fields []domain.Field, data map[string]any) string {
	var b strings.Builder
	for _, f := range fields {
		v, ok := data[domain.FieldID(f.ID)]
		if !ok {
			continue
		}
		if part := domain.SearchContribution(f.Type, v); part != "" {
			b.WriteString(part)
			b.WriteByte(' ')
		}
	}
	return strings.TrimSpace(b.String())
}

// coerceData — оставить только значения определённых полей и проверить их по
// типу (number-маска, варианты select). Неизвестные ключи отбрасываются.
func coerceData(fields []domain.Field, data map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for _, f := range fields {
		key := domain.FieldID(f.ID)
		v, ok := data[key]
		if !ok || v == nil {
			continue
		}
		if err := validateValue(f, v); err != nil {
			return nil, err
		}
		out[key] = v
	}
	return out, nil
}

func validateValue(f domain.Field, v any) error {
	switch f.Type {
	case domain.FieldNumber:
		s := valueString(v)
		if pat := f.NumberPattern(); pat != "" && s != "" {
			re, err := regexp.Compile(pat)
			if err == nil && !re.MatchString(s) {
				return domain.NewError("VALIDATION",
					"Значение поля «"+f.Label+"» не соответствует шаблону", 400)
			}
		}
	case domain.FieldSelect:
		opts := f.FieldOptions()
		if len(opts) == 0 {
			return nil
		}
		allowed := map[string]bool{}
		for _, o := range opts {
			allowed[o] = true
		}
		for _, chosen := range selectValues(v) {
			if !allowed[chosen] {
				return domain.NewError("VALIDATION",
					"Недопустимый вариант поля «"+f.Label+"»", 400)
			}
		}
	}
	return nil
}

func valueString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	switch n := v.(type) {
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	return ""
}

func selectValues(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		out := make([]string, 0, len(x))
		for _, it := range x {
			if s, ok := it.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func entryPayload(companyID int64, e *domain.Entry) map[string]any {
	return map[string]any{
		"id": e.ID, "calendar_id": e.CalendarID, "company_id": companyID,
		"event_at": e.EventAt, "data": e.Data, "created_by": e.CreatedBy,
		"created_at": e.CreatedAt, "updated_at": e.UpdatedAt,
	}
}
