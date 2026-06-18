package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

// ExportRecords — xlsx с выбранными полями. ids != nil → только эти записи,
// иначе все записи реестра по фильтру search. Экспортируются только текстовые
// типы полей (картинки/файлы исключаются). Возвращает байты файла и имя реестра.
func (s *Service) ExportRecords(ctx context.Context, companyID, registryID int64, fieldIDs []int64, search string, ids []int64) ([]byte, string, error) {
	reg, err := s.requireRegistry(ctx, companyID, registryID)
	if err != nil {
		return nil, "", err
	}
	return s.buildExport(ctx, reg, fieldIDs, search, ids)
}

// buildExport — формирование xlsx по уже проверенному реестру (authed или
// публичный доступ по ссылке).
func (s *Service) buildExport(ctx context.Context, reg *domain.Registry, fieldIDs []int64, search string, ids []int64) ([]byte, string, error) {
	allFields, err := s.repo.ListFields(ctx, reg.ID)
	if err != nil {
		return nil, "", err
	}

	// Колонки — в порядке реестра, пересечение «экспортируемых» с запрошенными.
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
	if len(cols) == 0 {
		return nil, "", domain.NewError("VALIDATION", "Выберите хотя бы одно поле для экспорта", 400)
	}

	records, err := s.repo.RecordsForExport(ctx, reg.ID, search, ids)
	if err != nil {
		return nil, "", err
	}

	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Реестр"
	f.SetSheetName(f.GetSheetName(0), sheet)

	for ci, col := range cols {
		cell, _ := excelize.CoordinatesToCellName(ci+1, 1)
		f.SetCellStr(sheet, cell, col.Label)
	}
	for ri, rec := range records {
		for ci, col := range cols {
			cell, _ := excelize.CoordinatesToCellName(ci+1, ri+2)
			f.SetCellStr(sheet, cell, exportValue(col, rec.Data[domain.FieldID(col.ID)]))
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), reg.Name, nil
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
