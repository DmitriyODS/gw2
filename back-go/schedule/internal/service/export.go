package service

import (
	"context"
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

// weekdayNames — подписи дней в выгрузке (0 — понедельник).
var weekdayNames = []string{"Понедельник", "Вторник", "Среда", "Четверг",
	"Пятница", "Суббота", "Воскресенье"}

// ExportXLSX — расписание таблицей: ЛИСТ НА КАЖДУЮ НЕДЕЛЮ ЦИКЛА. Иначе
// двухнедельное расписание пришлось бы читать с колонкой «в какие недели», а
// на бумаге человеку нужна готовая неделя.
func (s *Service) ExportXLSX(ctx context.Context, userID, scheduleID int64) ([]byte, string, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	sc, _, err := s.requireRead(ctx, a, scheduleID)
	if err != nil {
		return nil, "", err
	}
	view, err := s.view(ctx, sc, false)
	if err != nil {
		return nil, "", err
	}

	catByID := map[int64]string{}
	for _, c := range sc.Categories {
		catByID[c.ID] = c.Name
	}
	fields := make([]*domain.Field, 0, len(sc.Fields))
	for _, f := range sc.Fields {
		if records.Exportable(f.Type) {
			fields = append(fields, f)
		}
	}

	f := excelize.NewFile()
	defer f.Close()

	for week := 1; week <= sc.CycleWeeks; week++ {
		sheet := weekSheetName(sc, week)
		if week == 1 {
			f.SetSheetName(f.GetSheetName(0), sheet)
		} else if _, err := f.NewSheet(sheet); err != nil {
			return nil, "", err
		}

		header := []string{"День", "Начало", "Конец", "Занятие", "Категория"}
		for _, fl := range fields {
			header = append(header, fl.Label)
		}
		for i, title := range header {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellStr(sheet, cell, title)
		}

		row := 1
		for weekday := 0; weekday < 7; weekday++ {
			for _, item := range view.Items {
				if item.Weekday != weekday || !itemInWeek(item, week) {
					continue
				}
				row++
				line := []string{weekdayNames[weekday], hhmm(item.StartMin), hhmm(item.EndMin),
					item.Title, ""}
				if item.CategoryID != nil {
					line[4] = catByID[*item.CategoryID]
				}
				for _, fl := range fields {
					line = append(line, records.SearchContribution(fl.Type, item.Data[records.FieldID(fl.ID)]))
				}
				for i, value := range line {
					cell, _ := excelize.CoordinatesToCellName(i+1, row)
					f.SetCellStr(sheet, cell, value)
				}
			}
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), sc.Name, nil
}

// itemInWeek — попадает ли занятие в неделю цикла на выгрузке. Занятие со
// своим правилом повтора («каждые N недель») к неделям цикла не привязано и
// идёт во все листы — на бумаге лучше показать его лишний раз, чем потерять.
func itemInWeek(item *domain.Item, week int) bool {
	if item.RepeatEvery != nil {
		return true
	}
	if len(item.Weeks) == 0 {
		return true
	}
	for _, w := range item.Weeks {
		if w == week {
			return true
		}
	}
	return false
}

// weekSheetName — имя листа: своё название недели, если задано.
func weekSheetName(sc *domain.Schedule, week int) string {
	if sc.CycleWeeks == 1 {
		return "Расписание"
	}
	if label := sc.WeekLabel(week); label != "" {
		return excelizeSheetName(label)
	}
	return fmt.Sprintf("Неделя %d", week)
}

// excelizeSheetName — имя листа без запрещённых Excel символов и не длиннее 31.
func excelizeSheetName(name string) string {
	const forbidden = `:\/?*[]`
	out := make([]rune, 0, len(name))
	for _, r := range name {
		skip := false
		for _, bad := range forbidden {
			if r == bad {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, r)
		}
	}
	if len(out) > 31 {
		out = out[:31]
	}
	if len(out) == 0 {
		return "Лист"
	}
	return string(out)
}
