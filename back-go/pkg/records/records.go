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
	"time"

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
	// FieldStock — «Наличие»: позиция на месте, пока её не забрали. Значение —
	// {taken: bool, until: "YYYY-MM-DD"}; пусто (и taken=false) означает «в
	// наличии», поэтому у новых записей поле молчит и ничего не весит.
	FieldStock = "stock"
)

// FieldTypes — допустимые типы (для валидации структуры).
var FieldTypes = map[string]bool{
	FieldImage: true, FieldFile: true, FieldText: true, FieldNumber: true,
	FieldCheckbox: true, FieldSelect: true, FieldLink: true, FieldDatetime: true,
	FieldStock: true,
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

// numberRe — что вообще принимается в числовое поле. Держать в паре с
// регэкспом сортировки (registry/internal/repository/postgres/records.go):
// в базе обязано лежать ровно то, что Postgres приведёт к numeric, иначе
// сортировка по такому полю падает на первой же кривой ячейке.
var numberRe = regexp.MustCompile(`^[+-]?([0-9]+([.][0-9]*)?|[.][0-9]+)$`)

// ParseNumber — значение числового поля как число; ok=false — это не число.
func ParseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if !numberRe.MatchString(s) {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// trimNum — граница в сообщении об ошибке без хвоста «.000000»: «не меньше 0».
func trimNum(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// NumberBound — граница числового поля из config (min/max); ok=false, если
// граница не задана. Значение приезжает из JSONB, поэтому принимаем и число,
// и строку.
func NumberBound(config map[string]any, key string) (float64, bool) {
	switch v := config[key].(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		return ParseNumber(v)
	}
	return 0, false
}

// Stock — значение поля «Наличие». Пустое значение = позиция на месте, поэтому
// «в наличии» ничего в записи не занимает.
type Stock struct {
	Taken bool   `json:"taken"`
	Until string `json:"until,omitempty"` // «забрали до» — YYYY-MM-DD, необязательна
}

// stockDateRe — дата в формате YYYY-MM-DD; время здесь лишнее: позицию забирают
// на дни, а не на минуты.
var stockDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// StockValue — разбор значения поля «Наличие» из JSONB (или из формы).
func StockValue(v any) Stock {
	m, ok := v.(map[string]any)
	if !ok {
		return Stock{}
	}
	taken, _ := m["taken"].(bool)
	until, _ := m["until"].(string)
	return Stock{Taken: taken, Until: strings.TrimSpace(until)}
}

// StockText — человеческая надпись значения: её видят и таблица, и выгрузка.
func StockText(v any) string {
	st := StockValue(v)
	if !st.Taken {
		return "В наличии"
	}
	if st.Until == "" {
		return "Забрали"
	}
	if t, err := time.Parse("2006-01-02", st.Until); err == nil {
		return "Забрали до " + t.Format("02.01.2006")
	}
	return "Забрали до " + st.Until
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
	case FieldStock:
		// В поиск попадает только «забрали»: пометка «в наличии» стоит у
		// большинства записей и в общей строке поиска ничего не различает.
		st := StockValue(value)
		if !st.Taken {
			return ""
		}
		return strings.TrimSpace("забрали " + st.Until)
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
		s := strings.TrimSpace(valueString(v))
		if s == "" {
			return nil // пустое значение — «не заполнено», это законно
		}
		// Числовое поле обязано хранить ЧИСЛО: буквы в нём роняли сортировку
		// всего списка (Postgres не умеет приводить «уточнить» к numeric).
		num, ok := ParseNumber(s)
		if !ok {
			return apierror.New("VALIDATION",
				"Поле «"+f.Label+"» принимает только число", 400)
		}
		if min, has := NumberBound(f.Config, "min"); has && num < min {
			return apierror.New("VALIDATION",
				fmt.Sprintf("Поле «%s»: не меньше %s", f.Label, trimNum(min)), 400)
		}
		if max, has := NumberBound(f.Config, "max"); has && num > max {
			return apierror.New("VALIDATION",
				fmt.Sprintf("Поле «%s»: не больше %s", f.Label, trimNum(max)), 400)
		}
		if pat := NumberPattern(f.Config); pat != "" {
			re, err := regexp.Compile(pat)
			if err == nil && !re.MatchString(s) {
				return apierror.New("VALIDATION",
					"Значение поля «"+f.Label+"» не соответствует шаблону", 400)
			}
		}
	case FieldStock:
		m, ok := v.(map[string]any)
		if !ok {
			return apierror.New("VALIDATION",
				"Поле «"+f.Label+"»: непонятное значение наличия", 400)
		}
		until, _ := m["until"].(string)
		if until = strings.TrimSpace(until); until != "" && !stockDateRe.MatchString(until) {
			return apierror.New("VALIDATION",
				"Поле «"+f.Label+"»: дата возврата в формате ГГГГ-ММ-ДД", 400)
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

// StoredValue — файл, лежащий в значении поля записи (типы image и file).
// Значение хранится объектом {path, name, mime, size}.
type StoredValue struct {
	Path string
	Name string
}

// FileValue — файл из значения поля; ok=false для всех прочих типов.
func FileValue(v any) (StoredValue, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return StoredValue{}, false
	}
	path, ok := m["path"].(string)
	if !ok || path == "" {
		return StoredValue{}, false
	}
	name, _ := m["name"].(string)
	return StoredValue{Path: path, Name: name}, true
}

// DataFiles — все файлы записи (раздел «Настройки → Хранилище» и чистка
// хранилища при удалении записи).
func DataFiles(data map[string]any) []StoredValue {
	out := []StoredValue{}
	for _, v := range data {
		if f, ok := FileValue(v); ok {
			out = append(out, f)
		}
	}
	return out
}

// DataWithoutFiles — значения записи без файлов с перечисленными путями:
// поле остаётся, но становится пустым. Второе значение — менялось ли
// что-нибудь, третье — какие пути ушли.
func DataWithoutFiles(data map[string]any, drop map[string]bool) (map[string]any, bool, []string) {
	removed := []string{}
	out := make(map[string]any, len(data))
	for key, v := range data {
		if f, ok := FileValue(v); ok && drop[f.Path] {
			removed = append(removed, f.Path)
			continue
		}
		out[key] = v
	}
	if len(removed) == 0 {
		return data, false, nil
	}
	return out, true, removed
}

// NewShareCode — код-capability публичной ссылки (hex 32 символа).
func NewShareCode() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
