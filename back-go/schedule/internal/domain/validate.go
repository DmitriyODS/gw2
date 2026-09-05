package domain

import (
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

// MinutesInDay — верхняя граница времени занятия (полночь следующего дня).
const MinutesInDay = 24 * 60

// MaxRepeatEvery — потолок своего правила повтора: «каждые N недель».
const MaxRepeatEvery = 52

// Normalize — привести занятие к допустимому виду и проверить его. Цикл нужен,
// чтобы отсеять номера недель, которых в нём нет.
func (i *Item) Normalize(cycleWeeks int) error {
	i.Title = strings.TrimSpace(i.Title)
	i.Short = strings.TrimSpace(i.Short)
	if i.Title == "" {
		return ErrTitleRequired
	}
	if r := []rune(i.Title); len(r) > 200 {
		i.Title = string(r[:200])
	}
	if r := []rune(i.Short); len(r) > 60 {
		i.Short = string(r[:60])
	}
	if i.Weekday < 0 || i.Weekday > 6 {
		return ErrBadWeekday
	}
	if i.StartMin < 0 || i.EndMin > MinutesInDay || i.EndMin <= i.StartMin {
		return ErrBadTime
	}
	if i.Data == nil {
		i.Data = map[string]any{}
	}

	// Способы повтора взаимоисключающие: своё правило отменяет номера недель,
	// иначе занятие описывалось бы двумя разными правилами сразу.
	if i.RepeatEvery != nil {
		if *i.RepeatEvery < 1 || *i.RepeatEvery > MaxRepeatEvery || i.RepeatFrom == nil {
			return ErrBadRepeat
		}
		from := MondayOf(*i.RepeatFrom)
		i.RepeatFrom = &from
		if i.RepeatUntil != nil {
			until := MondayOf(*i.RepeatUntil)
			if until.Before(from) {
				return ErrBadRepeat
			}
			i.RepeatUntil = &until
		}
		i.Weeks = []int{}
		return nil
	}
	i.RepeatFrom, i.RepeatUntil = nil, nil
	i.Weeks = NormalizeWeeks(i.Weeks, cycleWeeks)
	return nil
}

// FieldInfos — поля расписания в виде, понятном общему ядру записей.
func FieldInfos(fields []*Field) []records.FieldInfo {
	out := make([]records.FieldInfo, 0, len(fields))
	for _, f := range fields {
		out = append(out, records.FieldInfo{ID: f.ID, Type: f.Type, Label: f.Label, Config: f.Config})
	}
	return out
}

// CoerceData — оставить в значениях занятия только известные поля и проверить
// их по типу (ядро pkg/records — общее с реестрами и календарями).
func CoerceData(fields []*Field, data map[string]any) (map[string]any, error) {
	return records.CoerceData(FieldInfos(fields), data)
}

// SearchText — строка глобального поиска по занятию: название, короткое имя и
// значения полей. Категорию добавляет репозиторий — она живёт своей таблицей.
func SearchText(fields []*Field, item *Item) string {
	parts := []string{item.Title, item.Short, records.SearchText(FieldInfos(fields), item.Data)}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}

// NormalizeField — привести поле к допустимому виду.
func NormalizeField(f *Field) error {
	f.Label = strings.TrimSpace(f.Label)
	if r := []rune(f.Label); len(r) > 120 {
		f.Label = string(r[:120])
	}
	if !FieldTypes[f.Type] {
		return ErrBadFieldType
	}
	f.Normalize()
	return nil
}

// NormalizeCategoryColor — ключ палитры категории. Незнакомый цвет приводим к
// первому из набора: в разметку значение уходит именем токена --tag-*, и
// произвольная строка нарисовала бы пустоту.
func NormalizeCategoryColor(color string) string {
	color = strings.TrimSpace(strings.ToLower(color))
	for _, c := range CategoryColors {
		if c == color {
			return color
		}
	}
	return CategoryColors[0]
}

// CategoryColors — палитра категорий: та же восьмёрка цветов-тегов, что у
// задач (токены --tag-*), продублирована во фронте front/src/utils/taskColors.js.
var CategoryColors = []string{"red", "orange", "amber", "green", "teal", "blue", "violet", "pink"}
