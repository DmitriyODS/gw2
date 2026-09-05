package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

/* Перенос расписания одним JSON.

   Значения полей и категории уезжают ИМЕНАМИ, а не id: файл открывают в другом
   аккаунте, где тех же id не существует, и ссылка на них потеряла бы половину
   карточки.

   Импорт понимает два формата — наш и прототипа («Расписание» на PHP/SQLite,
   откуда раздел вырос): в нём неделя описана как both/num/den, а вид занятия и
   место лежат отдельными ключами. Такой файл разворачивается в цикл из двух
   недель с числителем и знаменателем. */

const transferVersion = 1

type transferDoc struct {
	Version    int                `json:"version"`
	Schedule   transferSchedule   `json:"schedule"`
	Categories []transferCategory `json:"categories"`
	Fields     []transferField    `json:"fields"`
	Items      []transferItem     `json:"items"`
}

type transferSchedule struct {
	Name        string   `json:"name"`
	CycleWeeks  int      `json:"cycle_weeks"`
	CycleAnchor string   `json:"cycle_anchor"`
	WeekLabels  []string `json:"week_labels"`
	Timezone    string   `json:"timezone"`
	GapMin      int      `json:"gap_min"`
}

type transferCategory struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type transferField struct {
	Label       string         `json:"label"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	ColSpan     int            `json:"col_span"`
	RowSpan     int            `json:"row_span"`
	ShowOnBlock bool           `json:"show_on_block"`
	ShowInCard  bool           `json:"show_in_card"`
}

type transferItem struct {
	Weekday     int            `json:"weekday"`
	Start       string         `json:"start"`
	End         string         `json:"end"`
	Title       string         `json:"title"`
	Short       string         `json:"short"`
	Category    string         `json:"category"`
	Weeks       []int          `json:"weeks"`
	RepeatEvery *int           `json:"repeat_every,omitempty"`
	RepeatFrom  string         `json:"repeat_from,omitempty"`
	RepeatUntil string         `json:"repeat_until,omitempty"`
	Data        map[string]any `json:"data"`
}

// Export — расписание целиком одним JSON (перенос между аккаунтами).
func (s *Service) Export(ctx context.Context, userID, scheduleID int64) ([]byte, string, error) {
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

	doc := transferDoc{
		Version: transferVersion,
		Schedule: transferSchedule{
			Name: sc.Name, CycleWeeks: sc.CycleWeeks,
			CycleAnchor: sc.CycleAnchor.Format(domain.DateLayout),
			WeekLabels:  sc.WeekLabels, Timezone: sc.Timezone, GapMin: sc.GapMin,
		},
		Categories: []transferCategory{},
		Fields:     []transferField{},
		Items:      []transferItem{},
	}
	catByID := map[int64]string{}
	for _, c := range sc.Categories {
		catByID[c.ID] = c.Name
		doc.Categories = append(doc.Categories, transferCategory{Name: c.Name, Color: c.Color})
	}
	fieldByID := map[string]string{}
	for _, f := range sc.Fields {
		fieldByID[records.FieldID(f.ID)] = f.Label
		doc.Fields = append(doc.Fields, transferField{
			Label: f.Label, Type: f.Type, Config: f.Config, ColSpan: f.ColSpan,
			RowSpan: f.RowSpan, ShowOnBlock: f.ShowOnBlock, ShowInCard: f.ShowInCard,
		})
	}
	for _, item := range view.Items {
		row := transferItem{
			Weekday: item.Weekday, Start: hhmm(item.StartMin), End: hhmm(item.EndMin),
			Title: item.Title, Short: item.Short, Weeks: item.Weeks,
			RepeatEvery: item.RepeatEvery, Data: map[string]any{},
		}
		if item.CategoryID != nil {
			row.Category = catByID[*item.CategoryID]
		}
		if item.RepeatFrom != nil {
			row.RepeatFrom = item.RepeatFrom.Format(domain.DateLayout)
		}
		if item.RepeatUntil != nil {
			row.RepeatUntil = item.RepeatUntil.Format(domain.DateLayout)
		}
		for key, v := range item.Data {
			if label, ok := fieldByID[key]; ok {
				row.Data[label] = v
			}
		}
		doc.Items = append(doc.Items, row)
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	return raw, sc.Name, err
}

// Import — заменить содержимое расписания файлом (структура и занятия целиком).
func (s *Service) Import(ctx context.Context, userID, scheduleID int64, raw []byte) (*View, error) {
	sc, err := s.requireOwner(ctx, userID, scheduleID)
	if err != nil {
		return nil, err
	}
	doc, err := parseTransfer(raw)
	if err != nil {
		return nil, err
	}
	return s.applyTransfer(ctx, sc, doc, false)
}

// ImportNew — создать расписание из файла.
func (s *Service) ImportNew(ctx context.Context, userID int64, raw []byte, name string) (*View, error) {
	doc, err := parseTransfer(raw)
	if err != nil {
		return nil, err
	}
	if n := strings.TrimSpace(name); n != "" {
		doc.Schedule.Name = n
	}
	if strings.TrimSpace(doc.Schedule.Name) == "" {
		doc.Schedule.Name = "Расписание"
	}
	view, err := s.CreateSchedule(ctx, userID, ScheduleInput{
		Name: doc.Schedule.Name, CycleWeeks: doc.Schedule.CycleWeeks,
		CycleAnchor: parseDate(doc.Schedule.CycleAnchor), Timezone: doc.Schedule.Timezone,
	})
	if err != nil {
		return nil, err
	}
	return s.applyTransfer(ctx, view.Schedule, doc, true)
}

// applyTransfer — разложить файл по расписанию: настройки, категории, поля,
// занятия. Порядок важен: занятия ссылаются на свежие id категорий и полей.
func (s *Service) applyTransfer(ctx context.Context, sc *domain.Schedule, doc *transferDoc, fresh bool) (*View, error) {
	if !fresh {
		if name := strings.TrimSpace(doc.Schedule.Name); name != "" {
			sc.Name = name
		}
		sc.CycleWeeks = domain.NormalizeCycleWeeks(doc.Schedule.CycleWeeks)
		if anchor := parseDate(doc.Schedule.CycleAnchor); !anchor.IsZero() {
			sc.CycleAnchor = domain.MondayOf(anchor)
		}
		sc.Timezone = normalizeTimezone(doc.Schedule.Timezone)
		if doc.Schedule.GapMin > 0 {
			sc.GapMin = domain.NormalizeGap(doc.Schedule.GapMin)
		}
	}
	sc.WeekLabels = domain.NormalizeLabels(doc.Schedule.WeekLabels, sc.CycleWeeks)
	if err := s.repo.UpdateSchedule(ctx, sc); err != nil {
		return nil, err
	}

	cats := make([]*domain.Category, 0, len(doc.Categories))
	for _, c := range doc.Categories {
		name := strings.TrimSpace(c.Name)
		if name == "" {
			continue
		}
		cats = append(cats, &domain.Category{
			Name: trimRunes(name, 80), Color: domain.NormalizeCategoryColor(c.Color),
		})
	}
	if err := s.repo.ReplaceCategories(ctx, sc.ID, cats); err != nil {
		return nil, err
	}
	catByName := make(map[string]int64, len(cats))
	for _, c := range cats {
		catByName[strings.ToLower(c.Name)] = c.ID
	}

	fields := make([]*domain.Field, 0, len(doc.Fields))
	for _, f := range doc.Fields {
		field := &domain.Field{
			Label: f.Label, Type: f.Type, Config: f.Config, ColSpan: f.ColSpan,
			RowSpan: f.RowSpan, ShowOnBlock: f.ShowOnBlock, ShowInCard: f.ShowInCard,
		}
		if err := domain.NormalizeField(field); err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	if _, err := s.repo.ReplaceFields(ctx, sc.ID, fields); err != nil {
		return nil, err
	}
	fieldByLabel := make(map[string]int64, len(fields))
	for _, f := range fields {
		fieldByLabel[strings.ToLower(f.Label)] = f.ID
	}

	items := make([]*domain.Item, 0, len(doc.Items))
	for _, row := range doc.Items {
		item := &domain.Item{
			ScheduleID: sc.ID, Weekday: row.Weekday,
			StartMin: parseHHMM(row.Start), EndMin: parseHHMM(row.End),
			Title: row.Title, Short: row.Short, Weeks: row.Weeks,
			RepeatEvery: row.RepeatEvery, Data: map[string]any{},
		}
		if id, ok := catByName[strings.ToLower(strings.TrimSpace(row.Category))]; ok {
			item.CategoryID = &id
		}
		if d := parseDate(row.RepeatFrom); !d.IsZero() {
			item.RepeatFrom = &d
		}
		if d := parseDate(row.RepeatUntil); !d.IsZero() {
			item.RepeatUntil = &d
		}
		for label, v := range row.Data {
			if id, ok := fieldByLabel[strings.ToLower(strings.TrimSpace(label))]; ok {
				item.Data[records.FieldID(id)] = v
			}
		}
		// Занятие, не прошедшее проверку, пропускаем: чужой файл не должен
		// ронять импорт целиком из-за одной кривой строки.
		if err := item.Normalize(sc.CycleWeeks); err != nil {
			continue
		}
		data, err := domain.CoerceData(fields, item.Data)
		if err != nil {
			continue
		}
		item.Data = data
		items = append(items, item)
	}
	if err := s.repo.ReplaceItems(ctx, sc.ID, items, func(i *domain.Item) string {
		return domain.SearchText(fields, i)
	}); err != nil {
		return nil, err
	}
	s.publish(ctx, sc.ID, "schedule:updated", schedulePayload(sc))
	return s.view(ctx, sc, true)
}

// parseTransfer — разобрать файл: наш формат или формат прототипа.
func parseTransfer(raw []byte) (*transferDoc, error) {
	var probe struct {
		Schedule *json.RawMessage  `json:"schedule"`
		Items    []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, domain.ErrBadImport
	}
	if probe.Schedule != nil {
		var doc transferDoc
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, domain.ErrBadImport
		}
		return &doc, nil
	}
	if len(probe.Items) > 0 {
		return parseLegacy(raw)
	}
	return nil, domain.ErrBadImport
}

// legacyItem — занятие в формате прототипа.
type legacyItem struct {
	Day      int    `json:"day"`
	Category string `json:"category"` // study | work | extra
	Week     string `json:"week"`     // both | num | den
	Start    string `json:"start"`
	End      string `json:"end"`
	Title    string `json:"title"`
	Short    string `json:"short"`
	Kind     string `json:"kind"`
	Place    string `json:"place"`
}

// legacyCategories — категории прототипа в нашем справочнике.
var legacyCategories = []struct{ key, name, color string }{
	{"study", "Учёба", "blue"},
	{"work", "Работа", "green"},
	{"extra", "Доп", "violet"},
}

// parseLegacy — расписание из прототипа. Числитель и знаменатель — это цикл из
// двух недель, поэтому both → «каждую», num → неделя 1, den → неделя 2. Вид
// занятия и место становятся двумя текстовыми полями (место видно прямо на
// блоке, как было).
func parseLegacy(raw []byte) (*transferDoc, error) {
	var doc struct {
		Items []legacyItem `json:"items"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, domain.ErrBadImport
	}
	out := &transferDoc{
		Version: transferVersion,
		Schedule: transferSchedule{
			Name: "Расписание", CycleWeeks: 2,
			WeekLabels: []string{"Числитель", "Знаменатель"},
		},
		Categories: []transferCategory{},
		Fields:     []transferField{},
		Items:      []transferItem{},
	}
	usedCat := map[string]bool{}
	hasKind, hasPlace := false, false
	for _, it := range doc.Items {
		usedCat[it.Category] = true
		hasKind = hasKind || strings.TrimSpace(it.Kind) != ""
		hasPlace = hasPlace || strings.TrimSpace(it.Place) != ""
	}
	catName := map[string]string{}
	for _, c := range legacyCategories {
		catName[c.key] = c.name
		if usedCat[c.key] {
			out.Categories = append(out.Categories, transferCategory{Name: c.name, Color: c.color})
		}
	}
	if hasKind {
		out.Fields = append(out.Fields, transferField{
			Label: "Вид", Type: domain.FieldText, ColSpan: 1, RowSpan: 1, ShowInCard: true,
		})
	}
	if hasPlace {
		out.Fields = append(out.Fields, transferField{
			Label: "Место", Type: domain.FieldText, ColSpan: 1, RowSpan: 1,
			ShowOnBlock: true, ShowInCard: true,
		})
	}
	for _, it := range doc.Items {
		row := transferItem{
			Weekday: it.Day, Start: it.Start, End: it.End, Title: it.Title,
			Short: it.Short, Category: catName[it.Category], Data: map[string]any{},
		}
		switch it.Week {
		case "num":
			row.Weeks = []int{1}
		case "den":
			row.Weeks = []int{2}
		default:
			row.Weeks = []int{}
		}
		if hasKind && strings.TrimSpace(it.Kind) != "" {
			row.Data["Вид"] = it.Kind
		}
		if hasPlace && strings.TrimSpace(it.Place) != "" {
			row.Data["Место"] = it.Place
		}
		out.Items = append(out.Items, row)
	}
	return out, nil
}

// hhmm — минуты от полуночи как «ЧЧ:ММ».
func hhmm(min int) string { return fmt.Sprintf("%02d:%02d", min/60, min%60) }

// parseHHMM — «ЧЧ:ММ» в минуты от полуночи (кривое время даёт 0, и занятие
// отсеется проверкой «конец позже начала»).
func parseHHMM(s string) int {
	parts := strings.SplitN(strings.TrimSpace(s), ":", 2)
	if len(parts) != 2 {
		return 0
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 24 || m < 0 || m > 59 {
		return 0
	}
	return h*60 + m
}

func parseDate(s string) time.Time {
	if s = strings.TrimSpace(s); s == "" {
		return time.Time{}
	}
	t, err := time.Parse(domain.DateLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
