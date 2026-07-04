// Package records — общее ядро «настраиваемых записей» (реестры и календари):
// типы полей карточки, валидация значений, search_text, экспорт-фильтры и коды
// публичных ссылок. Сервисы registry/calendar держат собственные доменные
// модели (разные json-формы и расширения), но всю структурную логику берут
// отсюда — без копипасты.
package records

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/apierror"
)

// Типы полей карточки. Набор продублирован во фронте
// (front/src/utils/registryFields.js) — держать синхронным.
const (
	FieldImage    = "image"    // картинка (превью + полноэкранный просмотр)
	FieldFile     = "file"     // произвольный файл
	FieldText     = "text"     // текстовое поле (config.multiline — textarea)
	FieldNumber   = "number"   // число (config.pattern — опц. regex шаблона)
	FieldCheckbox = "checkbox" // галочка
	FieldSelect   = "select"   // выбор из вариантов (config.options, config.multiple)
	FieldLink     = "link"     // ссылка на сайт
	FieldDatetime = "datetime" // дата/время (config.year/month_day/time — части)
)

// FieldTypes — допустимые типы (для валидации структуры).
var FieldTypes = map[string]bool{
	FieldImage: true, FieldFile: true, FieldText: true, FieldNumber: true,
	FieldCheckbox: true, FieldSelect: true, FieldLink: true, FieldDatetime: true,
}

// FieldInfo — минимум сведений о поле для валидации значений и search_text
// (сервисы конвертируют в него свои доменные Field).
type FieldInfo struct {
	ID     int64
	Type   string
	Label  string
	Config map[string]any
}

// FieldID — строковый ключ поля в data-JSONB записи.
func FieldID(id int64) string { return strconv.FormatInt(id, 10) }

// NormalizeSpans — привести span'ы раскладки к допустимым границам
// (col 1..3, row ≥1) и гарантировать непустой config.
func NormalizeSpans(colSpan, rowSpan *int, config *map[string]any) {
	if *colSpan < 1 {
		*colSpan = 1
	}
	if *colSpan > 3 {
		*colSpan = 3
	}
	if *rowSpan < 1 {
		*rowSpan = 1
	}
	if *config == nil {
		*config = map[string]any{}
	}
}

// Options — варианты select-поля из config (пустой срез, если нет).
func Options(config map[string]any) []string {
	raw, ok := config["options"].([]any)
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

// Multiple — допускает ли select несколько значений.
func Multiple(config map[string]any) bool {
	b, _ := config["multiple"].(bool)
	return b
}

// NumberPattern — опциональная regex-маска числового поля ("" — без маски).
func NumberPattern(config map[string]any) string {
	s, _ := config["pattern"].(string)
	return s
}

// Exportable — можно ли выгружать поле в xlsx. Картинки и файлы — нет
// (они не сводятся к текстовой ячейке), остальные типы экспортируются.
func Exportable(fieldType string) bool {
	return fieldType != FieldImage && fieldType != FieldFile
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

// SearchText — сквозная строка поиска по значениям всех полей записи.
func SearchText(fields []FieldInfo, data map[string]any) string {
	var b strings.Builder
	for _, f := range fields {
		v, ok := data[FieldID(f.ID)]
		if !ok {
			continue
		}
		if part := SearchContribution(f.Type, v); part != "" {
			b.WriteString(part)
			b.WriteByte(' ')
		}
	}
	return strings.TrimSpace(b.String())
}

// CoerceData — оставить только значения определённых полей и проверить их по
// типу (number-маска, варианты select). Неизвестные ключи отбрасываются.
func CoerceData(fields []FieldInfo, data map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for _, f := range fields {
		key := FieldID(f.ID)
		v, ok := data[key]
		if !ok || v == nil {
			continue
		}
		if err := ValidateValue(f, v); err != nil {
			return nil, err
		}
		out[key] = v
	}
	return out, nil
}

// ValidateValue — проверка значения одного поля по его типу и config.
func ValidateValue(f FieldInfo, v any) error {
	switch f.Type {
	case FieldNumber:
		s := valueString(v)
		if pat := NumberPattern(f.Config); pat != "" && s != "" {
			re, err := regexp.Compile(pat)
			if err == nil && !re.MatchString(s) {
				return apierror.New("VALIDATION",
					"Значение поля «"+f.Label+"» не соответствует шаблону", 400)
			}
		}
	case FieldSelect:
		opts := Options(f.Config)
		if len(opts) == 0 {
			return nil
		}
		allowed := map[string]bool{}
		for _, o := range opts {
			allowed[o] = true
		}
		for _, chosen := range selectValues(v) {
			if !allowed[chosen] {
				return apierror.New("VALIDATION",
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

// NewShareCode — код-capability публичной ссылки (hex 32 символа).
func NewShareCode() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
