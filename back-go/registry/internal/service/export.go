package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

// ExportParams — что выгружаем: колонки плюс набор записей (фильтр экрана либо
// явно выбранные). Набор описывается так же, как у массового удаления и печати
// QR: «выбрано» на всех экранах означает одно и то же.
type ExportParams struct {
	FieldIDs  []int64
	Selection BulkParams
}

// ExportRecords — xlsx с выбранными полями. Экспортируются только сводимые к
// ячейке типы (картинки и файлы исключаются). Возвращает байты и имя реестра.
func (s *Service) ExportRecords(ctx context.Context, userID, registryID int64, p ExportParams) ([]byte, string, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	reg, err := s.require(ctx, a, registryID, domain.AccessView)
	if err != nil {
		return nil, "", err
	}
	return s.buildExport(ctx, reg, p)
}

// buildExport — формирование xlsx по уже проверенному реестру (авторизованный
// доступ или публичный по ссылке).
func (s *Service) buildExport(ctx context.Context, reg *domain.Registry, p ExportParams) ([]byte, string, error) {
	allFields, err := s.repo.ListFields(ctx, reg.ID)
	if err != nil {
		return nil, "", err
	}

	// Колонки — в порядке реестра, пересечение «экспортируемых» с запрошенными.
	want := map[int64]bool{}
	for _, id := range p.FieldIDs {
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

	filter, err := s.selectionFilter(ctx, reg, p.Selection)
	if err != nil {
		return nil, "", err
	}
	recs, err := s.repo.RecordsForExport(ctx, filter)
	if err != nil {
		return nil, "", err
	}
	// В учётном реестре состояние позиции — такая же колонка отчёта, как
	// остальные: выгрузка без неё отвечает на половину вопросов.
	if err := s.attachIssues(ctx, reg, recs); err != nil {
		return nil, "", err
	}

	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Реестр"
	f.SetSheetName(f.GetSheetName(0), sheet)

	now := time.Now()
	for ci, col := range cols {
		cell, _ := excelize.CoordinatesToCellName(ci+1, 1)
		f.SetCellStr(sheet, cell, col.Label)
	}
	if reg.Accounting {
		cell, _ := excelize.CoordinatesToCellName(len(cols)+1, 1)
		f.SetCellStr(sheet, cell, "Состояние")
	}
	for ri, rec := range recs {
		for ci, col := range cols {
			cell, _ := excelize.CoordinatesToCellName(ci+1, ri+2)
			f.SetCellStr(sheet, cell, exportValue(col, rec.Data[domain.FieldID(col.ID)]))
		}
		if reg.Accounting {
			cell, _ := excelize.CoordinatesToCellName(len(cols)+1, ri+2)
			f.SetCellStr(sheet, cell, issueText(rec.Issue, now))
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), reg.Name, nil
}

// issueText — состояние позиции строкой (зеркало плашки в таблице).
func issueText(issue *domain.Issue, now time.Time) string {
	switch issue.State(now) {
	case domain.StockOverdue:
		return fmt.Sprintf("Просрочено на %d дн. (%s)", issue.OverdueDays(now), issue.HolderName)
	case domain.StockIssued:
		return fmt.Sprintf("Выдано до %s (%s)", issue.DueAt.Format("02.01.2006"), issue.HolderName)
	case domain.StockNoDue:
		return fmt.Sprintf("Выдано без срока (%s)", issue.HolderName)
	default:
		return "В наличии"
	}
}

// exportValue — текстовое представление значения для ячейки (зеркало
// front textValue): галочка → свои надписи, список → через запятую,
// дата → по включённым частям.
func exportValue(field domain.Field, v any) string {
	if v == nil {
		return ""
	}
	switch field.Type {
	case domain.FieldCheckbox:
		b, _ := v.(bool)
		return records.CheckboxText(field.Config, b)
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

// formatDateTime — дата по включённым частям (зеркало utils/registryFields.js).
func formatDateTime(v any, cfg map[string]any) string {
	s, ok := v.(string)
	if !ok || s == "" {
		return fmt.Sprintf("%v", v)
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	p := records.DateConfig(cfg)
	pad := func(n int) string { return fmt.Sprintf("%02d", n) }

	date := []string{}
	if p.Day {
		date = append(date, pad(t.Day()))
	}
	if p.Month {
		date = append(date, pad(int(t.Month())))
	}
	if p.Year {
		date = append(date, fmt.Sprintf("%d", t.Year()))
	}
	clock := []string{}
	if p.Hours {
		clock = append(clock, pad(t.Hour()))
	}
	if p.Minutes {
		clock = append(clock, pad(t.Minute()))
	}
	if p.Seconds {
		clock = append(clock, pad(t.Second()))
	}

	parts := []string{}
	if len(date) > 0 {
		parts = append(parts, strings.Join(date, "."))
	}
	if len(clock) > 0 {
		parts = append(parts, strings.Join(clock, ":"))
	}
	return strings.Join(parts, " ")
}
