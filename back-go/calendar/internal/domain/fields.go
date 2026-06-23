package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// Normalize — привести span'ы к допустимым границам (col 1..3, row ≥1).
func (f *Field) Normalize() {
	if f.ColSpan < 1 {
		f.ColSpan = 1
	}
	if f.ColSpan > 3 {
		f.ColSpan = 3
	}
	if f.RowSpan < 1 {
		f.RowSpan = 1
	}
	if f.Config == nil {
		f.Config = map[string]any{}
	}
}

// SearchContribution — текстовое представление значения поля для search_text.
// Учитываются только осмысленные для поиска типы (текст, число, дата, список,
// ссылка). Картинки/файлы/галочки в общий поиск не попадают.
func SearchContribution(fieldType string, value any) string {
	if value == nil {
		return ""
	}
	switch fieldType {
	case FieldText, FieldNumber, FieldLink, FieldDatetime:
		return fmt.Sprintf("%v", value)
	case FieldSelect:
		switch v := value.(type) {
		case string:
			return v
		case []any:
			parts := make([]string, 0, len(v))
			for _, it := range v {
				parts = append(parts, fmt.Sprintf("%v", it))
			}
			return strings.Join(parts, " ")
		}
	}
	return ""
}

// FieldOptions — варианты select-поля из config (пустой срез, если нет).
func (f Field) FieldOptions() []string {
	raw, ok := f.Config["options"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// SelectMultiple — допускает ли select несколько значений.
func (f Field) SelectMultiple() bool {
	b, _ := f.Config["multiple"].(bool)
	return b
}

// NumberPattern — опциональная regex-маска числового поля ("" — без маски).
func (f Field) NumberPattern() string {
	s, _ := f.Config["pattern"].(string)
	return s
}

// FieldID — строковый ключ поля в Entry.Data.
func FieldID(id int64) string { return strconv.FormatInt(id, 10) }

// Exportable — можно ли выгружать поле в xlsx. Картинки и файлы — нет
// (они не сводятся к текстовой ячейке), остальные типы экспортируются.
func Exportable(fieldType string) bool {
	return fieldType != FieldImage && fieldType != FieldFile
}
