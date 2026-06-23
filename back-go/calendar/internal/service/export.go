package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
)

// ExportEntries — xlsx с выбранными полями за период. ids != nil → только эти
// записи, иначе все записи по фильтру (диапазон дат + поиск). Первой колонкой
// всегда идёт «Дата и время» записи; картинки/файлы исключаются.
func (s *Service) ExportEntries(ctx context.Context, companyID, calendarID int64, fieldIDs []int64, p EntryListParams, ids []int64) ([]byte, string, error) {
	cal, err := s.requireCalendar(ctx, companyID, calendarID)
	if err != nil {
		return nil, "", err
	}
	return s.buildExport(ctx, cal, fieldIDs, p, ids)
}

// buildExport — формирование xlsx по уже проверенному календарю (authed или
// публичный доступ по ссылке).
func (s *Service) buildExport(ctx context.Context, cal *domain.Calendar, fieldIDs []int64, p EntryListParams, ids []int64) ([]byte, string, error) {
	allFields, err := s.repo.ListFields(ctx, cal.ID)
	if err != nil {
		return nil, "", err
	}

	// Колонки — в порядке календаря, пересечение «экспортируемых» с запрошенными.
	want := map[int64]bool{}
	for _, id := range fieldIDs {
		want[id] = true
	}
	cols := make([]domain.Field, 0, len(allFields))
	for _, f := range allFields {
		if domain.Exportable(f.Type) && (len(want) == 0 || want[f.ID]) {
			cols = append(cols, f)
		}
	}

	entries, err := s.repo.EntriesForExport(ctx, domain.EntryListFilter{
		CalendarID: cal.ID,
		Search:     strings.TrimSpace(p.Search),
		From:       p.From,
		To:         p.To,
		Limit:      entriesLimit,
	}, ids)
	if err != nil {
		return nil, "", err
	}

	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Календарь"
	f.SetSheetName(f.GetSheetName(0), sheet)

	// Первая колонка — всегда дата/время записи.
	f.SetCellStr(sheet, "A1", "Дата и время")
	for ci, col := range cols {
		cell, _ := excelize.CoordinatesToCellName(ci+2, 1)
		f.SetCellStr(sheet, cell, col.Label)
	}
	for ri, e := range entries {
		f.SetCellStr(sheet, mustCell(1, ri+2), formatEventAt(e.EventAt))
		for ci, col := range cols {
			f.SetCellStr(sheet, mustCell(ci+2, ri+2), exportValue(col, e.Data[domain.FieldID(col.ID)]))
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), cal.Name, nil
}

func mustCell(col, row int) string {
	cell, _ := excelize.CoordinatesToCellName(col, row)
	return cell
}

func formatEventAt(t time.Time) string {
	return fmt.Sprintf("%02d.%02d.%d %02d:%02d", t.Day(), int(t.Month()), t.Year(), t.Hour(), t.Minute())
}

// exportValue — текстовое представление значения для ячейки (зеркало
// front textValue): галочка → Да/Нет, список → через запятую, дата → по конфигу.
func exportValue(field domain.Field, v any) string {
	if v == nil {
		return ""
	}
	switch field.Type {
	case domain.FieldCheckbox:
		if b, ok := v.(bool); ok && b {
			return "Да"
		}
		return "Нет"
	case domain.FieldSelect:
		switch x := v.(type) {
		case string:
			return x
		case []any:
			parts := make([]string, 0, len(x))
			for _, it := range x {
				parts = append(parts, fmt.Sprintf("%v", it))
			}
			return strings.Join(parts, ", ")
		}
		return ""
	case domain.FieldDatetime:
		return formatDateTime(v, field.Config)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatDateTime(v any, cfg map[string]any) string {
	s, ok := v.(string)
	if !ok || s == "" {
		return fmt.Sprintf("%v", v)
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	pad := func(n int) string { return fmt.Sprintf("%02d", n) }
	year := cfgBool(cfg, "year", true)
	monthDay := cfgBool(cfg, "month_day", true)
	withTime := cfgBool(cfg, "time", false)

	parts := []string{}
	switch {
	case monthDay && year:
		parts = append(parts, fmt.Sprintf("%s.%s.%d", pad(t.Day()), pad(int(t.Month())), t.Year()))
	case monthDay:
		parts = append(parts, fmt.Sprintf("%s.%s", pad(t.Day()), pad(int(t.Month()))))
	case year:
		parts = append(parts, fmt.Sprintf("%d", t.Year()))
	}
	if withTime {
		parts = append(parts, fmt.Sprintf("%s:%s", pad(t.Hour()), pad(t.Minute())))
	}
	return strings.Join(parts, " ")
}

func cfgBool(cfg map[string]any, key string, def bool) bool {
	if cfg == nil {
		return def
	}
	if b, ok := cfg[key].(bool); ok {
		return b
	}
	return def
}
