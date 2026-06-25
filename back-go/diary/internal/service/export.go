package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
)

// ExportEntries — xlsx записей ежедневника за вкладку (активные/архив). ids !=
// nil → только эти записи, иначе все по фильтру (диапазон/поиск). Доступно
// владельцу и адресату.
func (s *Service) ExportEntries(ctx context.Context, userID, diaryID int64, p ListParams, ids []int64) ([]byte, string, error) {
	d, _, err := s.requireReadable(ctx, userID, diaryID)
	if err != nil {
		return nil, "", err
	}
	return s.buildExport(ctx, d, p, ids)
}

func (s *Service) buildExport(ctx context.Context, d *domain.Diary, p ListParams, ids []int64) ([]byte, string, error) {
	f := domain.EntryListFilter{
		DiaryID:  d.ID,
		Archived: p.Archived,
		Search:   strings.TrimSpace(p.Search),
		From:     p.From,
		To:       p.To,
		Limit:    entriesLimit,
	}
	entries, err := s.repo.EntriesForExport(ctx, f, ids)
	if err != nil {
		return nil, "", err
	}

	xf := excelize.NewFile()
	defer xf.Close()
	const sheet = "Ежедневник"
	xf.SetSheetName(xf.GetSheetName(0), sheet)

	headers := []string{"Дата", "Время", "Название", "Описание", "Статус"}
	for ci, h := range headers {
		xf.SetCellStr(sheet, mustCell(ci+1, 1), h)
	}
	for ri, e := range entries {
		xf.SetCellStr(sheet, mustCell(1, ri+2), formatDate(e.Date))
		xf.SetCellStr(sheet, mustCell(2, ri+2), formatTimeRange(e.StartMin, e.EndMin))
		xf.SetCellStr(sheet, mustCell(3, ri+2), e.Title)
		xf.SetCellStr(sheet, mustCell(4, ri+2), e.Description)
		xf.SetCellStr(sheet, mustCell(5, ri+2), statusLabel(e.Done))
	}

	buf, err := xf.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), d.Name, nil
}

func mustCell(col, row int) string {
	cell, _ := excelize.CoordinatesToCellName(col, row)
	return cell
}

func formatDate(t time.Time) string {
	return fmt.Sprintf("%02d.%02d.%d", t.Day(), int(t.Month()), t.Year())
}

func formatTimeRange(start, end *int) string {
	if start == nil && end == nil {
		return ""
	}
	if end == nil {
		return hhmm(*start)
	}
	if start == nil {
		return "–" + hhmm(*end)
	}
	return hhmm(*start) + "–" + hhmm(*end)
}

func hhmm(min int) string {
	if min < 0 {
		min = 0
	}
	return fmt.Sprintf("%02d:%02d", min/60, min%60)
}

func statusLabel(done bool) string {
	if done {
		return "Выполнено"
	}
	return "Активно"
}
