package service

import (
	"bytes"
	"context"
	"time"

	"github.com/xuri/excelize/v2"
)

// XLSX-экспорты статистики (excelize вместо openpyxl): те же листы,
// заголовки и порядок колонок, что в stats_service.export_*_xlsx.

func (s *Service) ExportCommonXLSX(ctx context.Context, start, end time.Time, companyID *int64) ([]byte, error) {
	data, err := s.StatsCommon(ctx, start, end, companyID)
	if err != nil {
		return nil, err
	}

	wb := excelize.NewFile()
	defer wb.Close() //nolint:errcheck

	sheet1 := "Задачи за период"
	if err := wb.SetSheetName("Sheet1", sheet1); err != nil {
		return nil, err
	}
	rows1 := [][]any{
		{"Показатель", "Значение"},
		{"Долг", data.Tasks.Debt},
		{"Поступило", data.Tasks.Received},
		{"Закрыто", data.Tasks.Closed},
		{"Осталось", data.Tasks.Remaining},
	}
	if err := writeRows(wb, sheet1, rows1); err != nil {
		return nil, err
	}

	sheet2 := "Задачи по часам"
	if _, err := wb.NewSheet(sheet2); err != nil {
		return nil, err
	}
	rows2 := [][]any{{"Задача", "Суммарные часы"}}
	for _, row := range data.TasksByHours {
		rows2 = append(rows2, []any{row.Name, row.TotalHours})
	}
	if err := writeRows(wb, sheet2, rows2); err != nil {
		return nil, err
	}

	sheet3 := "По сотрудникам"
	if _, err := wb.NewSheet(sheet3); err != nil {
		return nil, err
	}
	rows3 := [][]any{{"Сотрудник", "Задач", "Суммарные часы"}}
	for _, row := range data.TasksByEmployees {
		rows3 = append(rows3, []any{row.FIO, row.TasksCount, row.TotalHours})
	}
	if err := writeRows(wb, sheet3, rows3); err != nil {
		return nil, err
	}

	return saveXLSX(wb)
}

func (s *Service) ExportExtendedXLSX(ctx context.Context, start, end time.Time, companyID *int64) ([]byte, error) {
	data, err := s.StatsExtended(ctx, start, end, companyID)
	if err != nil {
		return nil, err
	}

	wb := excelize.NewFile()
	defer wb.Close() //nolint:errcheck

	sheet1 := "По типам юнитов"
	if err := wb.SetSheetName("Sheet1", sheet1); err != nil {
		return nil, err
	}
	rows1 := [][]any{{"Тип юнита", "Суммарные часы", "Уникальных задач"}}
	for _, row := range data.ByUnitTypes {
		rows1 = append(rows1, []any{row.Name, row.TotalHours, row.TasksCount})
	}
	if err := writeRows(wb, sheet1, rows1); err != nil {
		return nil, err
	}

	sheet2 := "По отделам"
	if _, err := wb.NewSheet(sheet2); err != nil {
		return nil, err
	}
	rows2 := [][]any{{"Отдел", "Задач"}}
	for _, row := range data.ByDepartments {
		rows2 = append(rows2, []any{row.Name, row.TasksCount})
	}
	if err := writeRows(wb, sheet2, rows2); err != nil {
		return nil, err
	}

	sheet3 := "По типам и сотрудникам"
	if _, err := wb.NewSheet(sheet3); err != nil {
		return nil, err
	}
	rows3 := [][]any{{"Сотрудник", "Тип юнита", "Часы", "Задач"}}
	for _, userRow := range data.ByUnitTypesPerUser {
		for _, ut := range userRow.UnitTypes {
			rows3 = append(rows3, []any{userRow.FIO, ut.Name, ut.Hours, ut.TasksCount})
		}
	}
	if err := writeRows(wb, sheet3, rows3); err != nil {
		return nil, err
	}

	return saveXLSX(wb)
}

func writeRows(wb *excelize.File, sheet string, rows [][]any) error {
	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return err
		}
		if err := wb.SetSheetRow(sheet, cell, &row); err != nil {
			return err
		}
	}
	return nil
}

func saveXLSX(wb *excelize.File) ([]byte, error) {
	var buf bytes.Buffer
	if err := wb.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
