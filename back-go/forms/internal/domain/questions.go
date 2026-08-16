package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

/* Типы вопросов и работа со значениями ответов.

   Набор продублирован во фронте (front/src/utils/formFields.js) — держать
   синхронным. Значение ответа хранится в JSONB по строковому id вопроса, и его
   форма зависит от типа:

     short_text, paragraph  — строка
     radio, dropdown        — строка (вариант; при config.other — свой текст)
     checkbox               — массив строк
     scale, rating          — число
     grid_radio             — {строка: столбец}
     grid_checkbox          — {строка: [столбцы]}
     date                   — "ГГГГ-ММ-ДД" либо "ГГГГ-ММ-ДДTЧЧ:ММ" (config.with_time)
     time                   — "ЧЧ:ММ"
     file                   — массив загруженных файлов
     note                   — ответа не имеет вовсе */

const (
	QShortText    = "short_text"
	QParagraph    = "paragraph"
	QRadio        = "radio"
	QCheckbox     = "checkbox"
	QDropdown     = "dropdown"
	QScale        = "scale"
	QRating       = "rating"
	QGridRadio    = "grid_radio"
	QGridCheckbox = "grid_checkbox"
	QDate         = "date"
	QTime         = "time"
	QFile         = "file"
	QNote         = "note" // блок с текстом/картинкой: ответа не требует
	// QBooking — «Запись»: варианты с ограниченным числом мест (смена, время,
	// место в зале). Отвечающий видит остаток, занятые варианты недоступны, а
	// последнее место при одновременной отправке достаётся ровно одному —
	// остаток сверяет сервер под локом формы.
	QBooking = "booking"
)

// QuestionTypes — допустимые типы (валидация структуры формы).
var QuestionTypes = map[string]bool{
	QShortText: true, QParagraph: true, QRadio: true, QCheckbox: true,
	QDropdown: true, QScale: true, QRating: true, QGridRadio: true,
	QGridCheckbox: true, QDate: true, QTime: true, QFile: true, QNote: true,
	QBooking: true,
}

// Границы, общие для всех форм.
const (
	MaxOptions   = 200      // вариантов у одного вопроса
	MaxGridRows  = 50       // строк/столбцов сетки
	MaxFiles     = 10       // файлов в одном файловом вопросе
	MaxFileSize  = 1 << 30  // потолок файла ответа (едет частями)
	MaxImageSize = 2 << 20  // потолок картинки-иллюстрации
	ScaleMaxTop  = 10       // верх линейной шкалы
	RatingMaxTop = 10       // максимум звёзд
)

// Answerable — требует ли тип ответа (у пояснительного блока ответа нет).
func Answerable(qType string) bool { return qType != QNote }

// Choice — типы с заранее известным набором вариантов.
func Choice(qType string) bool {
	return qType == QRadio || qType == QCheckbox || qType == QDropdown || qType == QBooking
}

// Bookable — тип с ограниченным числом мест на вариант («Запись»).
func Bookable(qType string) bool { return qType == QBooking }

// Grid — типы-сетки.
func Grid(qType string) bool { return qType == QGridRadio || qType == QGridCheckbox }

// Branching — умеет ли тип уводить на другой раздел по выбранному варианту.
// Только одиночный выбор: у множественного «вариантов-победителей» несколько.
func Branching(qType string) bool { return qType == QRadio || qType == QDropdown }

// QuestionID — строковый ключ вопроса в Answers.
func QuestionID(id int64) string { return strconv.FormatInt(id, 10) }

// ── Чтение config ────────────────────────────────────────────────

// Options — варианты выбора.
func (q Question) Options() []string { return stringList(q.Config["options"]) }

/*
Capacity — сколько мест у варианта «Записи» (0 — вариант закрыт).

	Число хранится в config.capacity по названию варианта: набор мест у каждого
	свой (утренняя смена на десять человек, вечерняя на три).
*/
func (q Question) Capacity(option string) int {
	raw, _ := q.Config["capacity"].(map[string]any)
	if raw == nil {
		return 0
	}
	return intOf(raw[option], 0)
}

/*
VisibleIf — условие показа вопроса: id вопроса-источника и ожидаемые ответы

	(пустой список — «любой непустой ответ»). ok=false — условия нет, вопрос
	показывается всегда.
*/
func (q Question) VisibleIf() (questionID int64, values []string, ok bool) {
	id := int64(intOf(q.Config["visible_question_id"], 0))
	if id <= 0 || id == q.ID {
		return 0, nil, false
	}
	return id, stringList(q.Config["visible_values"]), true
}

// OptionOther — разрешён ли свой вариант («Другое»).
func (q Question) OptionOther() bool { return boolOf(q.Config["other"]) }

// GridRows / GridCols — строки и столбцы сетки.
func (q Question) GridRows() []string { return stringList(q.Config["rows"]) }
func (q Question) GridCols() []string { return stringList(q.Config["cols"]) }

// RequireEachRow — требовать ответ в каждой строке сетки.
func (q Question) RequireEachRow() bool { return boolOf(q.Config["require_each_row"]) }

// Scale — границы линейной шкалы (нижняя 0 или 1, верхняя 2..10).
func (q Question) Scale() (min, max int) {
	min, max = intOf(q.Config["min"], 1), intOf(q.Config["max"], 5)
	if min != 0 {
		min = 1
	}
	if max < min+1 {
		max = min + 1
	}
	if max > ScaleMaxTop {
		max = ScaleMaxTop
	}
	return min, max
}

// RatingMax — сколько делений у оценки (3..10).
func (q Question) RatingMax() int {
	max := intOf(q.Config["max"], 5)
	if max < 3 {
		max = 3
	}
	if max > RatingMaxTop {
		max = RatingMaxTop
	}
	return max
}

// FileLimits — сколько файлов и какого размера принимает файловый вопрос.
func (q Question) FileLimits() (count int, sizeBytes int64) {
	count = intOf(q.Config["max_files"], 1)
	if count < 1 {
		count = 1
	}
	if count > MaxFiles {
		count = MaxFiles
	}
	mb := intOf(q.Config["max_size_mb"], 10)
	if mb < 1 {
		mb = 1
	}
	size := int64(mb) << 20
	if size > MaxFileSize {
		size = MaxFileSize
	}
	return count, size
}

// WithTime — спрашивает ли вопрос-дата ещё и время.
func (q Question) WithTime() bool { return boolOf(q.Config["with_time"]) }

// Validation — проверка текстового ответа: {kind, pattern, hint, min, max}.
type Validation struct {
	Kind    string // none | number | email | url | regex | length
	Pattern string
	Hint    string
	Min     float64
	Max     float64
	HasMin  bool
	HasMax  bool
}

// Validation — правило проверки текстового вопроса.
func (q Question) Validation() Validation {
	raw, _ := q.Config["validation"].(map[string]any)
	if raw == nil {
		return Validation{Kind: "none"}
	}
	v := Validation{
		Kind:    strings.TrimSpace(stringOf(raw["kind"])),
		Pattern: strings.TrimSpace(stringOf(raw["pattern"])),
		Hint:    strings.TrimSpace(stringOf(raw["hint"])),
	}
	if v.Kind == "" {
		v.Kind = "none"
	}
	v.Min, v.HasMin = numberOf(raw["min"])
	v.Max, v.HasMax = numberOf(raw["max"])
	return v
}

// Target — куда уводит выбранный вариант: "" — обычный порядок, "submit" —
// отправить форму, иначе id раздела строкой.
func (q Question) Target(option string) string {
	raw, _ := q.Config["targets"].(map[string]any)
	if raw == nil {
		return ""
	}
	switch v := raw[option].(type) {
	case string:
		if v == NextNext {
			return ""
		}
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	}
	return ""
}

// ── Нормализация структуры ───────────────────────────────────────

// Normalize — привести вопрос к допустимому виду: обрезать лишние варианты,
// зажать границы, убрать настройки чужих типов и ключ теста у типов, которые
// нечем проверить.
func (q *Question) Normalize() {
	if q.Config == nil {
		q.Config = map[string]any{}
	}
	if q.AnswerKey == nil {
		q.AnswerKey = map[string]any{}
	}
	if !QuestionTypes[q.Type] {
		q.Type = QShortText
	}
	switch {
	case Choice(q.Type):
		q.Config["options"] = trimList(q.Options(), MaxOptions)
	case Grid(q.Type):
		q.Config["rows"] = trimList(q.GridRows(), MaxGridRows)
		q.Config["cols"] = trimList(q.GridCols(), MaxGridRows)
	case q.Type == QScale:
		min, max := q.Scale()
		q.Config["min"], q.Config["max"] = min, max
	case q.Type == QRating:
		q.Config["max"] = q.RatingMax()
	case q.Type == QFile:
		count, size := q.FileLimits()
		q.Config["max_files"], q.Config["max_size_mb"] = count, int(size>>20)
	}
	// «Запись»: места держим только у существующих вариантов — иначе удалённый
	// вариант продолжал бы занимать места в расчётах остатка.
	if q.Type == QBooking {
		capacity := map[string]any{}
		for _, opt := range q.Options() {
			capacity[opt] = q.Capacity(opt)
		}
		q.Config["capacity"] = capacity
		delete(q.Config, "other") // свой вариант мест не имеет
	}
	if q.Points < 0 {
		q.Points = 0
	}
	// Пояснительный блок ничего не спрашивает: ни обязательности, ни баллов.
	if q.Type == QNote {
		q.Required, q.Points = false, 0
		q.AnswerKey = map[string]any{}
	}
	// Ветвление держится на вариантах: у прочих типов оно ни к чему не привязано.
	if !Branching(q.Type) {
		delete(q.Config, "targets")
	}
}

// ── Значения ответов ─────────────────────────────────────────────

var (
	emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s.]+(\.[^@\s.]+)+$`)
	urlRe   = regexp.MustCompile(`^(https?://)?[^\s/$.?#][^\s]*$`)
	dateRe  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	dateTmRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}$`)
	timeRe  = regexp.MustCompile(`^\d{2}:\d{2}$`)
	numRe   = regexp.MustCompile(`^[+-]?([0-9]+([.][0-9]*)?|[.][0-9]+)$`)
)

/*
CoerceAnswer — привести значение к хранимому виду и проверить его.

	Ответ приходит и от участника, и от гостя по ссылке, поэтому проверка здесь
	единственная и обязательная: клиенту верить нельзя. Пустое значение законно
	(обязательность проверяется отдельно — она зависит от пройденного маршрута
	разделов, а не от самого вопроса).
*/
func (q Question) CoerceAnswer(v any) (any, error) {
	if v == nil || !Answerable(q.Type) {
		return nil, nil
	}
	switch q.Type {
	case QShortText, QParagraph:
		return q.coerceText(v)
	case QRadio, QDropdown, QBooking:
		return q.coerceChoice(v)
	case QCheckbox:
		return q.coerceChecks(v)
	case QScale, QRating:
		return q.coerceNumber(v)
	case QGridRadio, QGridCheckbox:
		return q.coerceGrid(v)
	case QDate:
		return q.coerceDate(v)
	case QTime:
		s := strings.TrimSpace(stringOf(v))
		if s == "" {
			return nil, nil
		}
		if !timeRe.MatchString(s) {
			return nil, q.invalid("укажите время в виде ЧЧ:ММ")
		}
		return s, nil
	case QFile:
		return q.coerceFiles(v)
	}
	return nil, nil
}

func (q Question) coerceText(v any) (any, error) {
	s := strings.TrimSpace(stringOf(v))
	if s == "" {
		return nil, nil
	}
	val := q.Validation()
	switch val.Kind {
	case "number":
		if !numRe.MatchString(s) {
			return nil, q.invalid("принимается только число")
		}
		n, _ := strconv.ParseFloat(s, 64)
		if val.HasMin && n < val.Min {
			return nil, q.invalid(fmt.Sprintf("не меньше %s", trimNum(val.Min)))
		}
		if val.HasMax && n > val.Max {
			return nil, q.invalid(fmt.Sprintf("не больше %s", trimNum(val.Max)))
		}
	case "email":
		if !emailRe.MatchString(s) {
			return nil, q.invalid("непохоже на адрес почты")
		}
	case "url":
		if !urlRe.MatchString(s) {
			return nil, q.invalid("непохоже на ссылку")
		}
	case "regex":
		// Кривой шаблон — вина составителя формы, а не отвечающего: на нём
		// вопрос ведёт себя как обычный текст, а не отвергает всё подряд.
		if val.Pattern != "" {
			if re, err := regexp.Compile(val.Pattern); err == nil && !re.MatchString(s) {
				msg := val.Hint
				if msg == "" {
					msg = "ответ не соответствует шаблону"
				}
				return nil, q.invalid(msg)
			}
		}
	case "length":
		n := float64(len([]rune(s)))
		if val.HasMin && n < val.Min {
			return nil, q.invalid(fmt.Sprintf("не короче %s символов", trimNum(val.Min)))
		}
		if val.HasMax && n > val.Max {
			return nil, q.invalid(fmt.Sprintf("не длиннее %s символов", trimNum(val.Max)))
		}
	}
	return s, nil
}

func (q Question) coerceChoice(v any) (any, error) {
	s := strings.TrimSpace(stringOf(v))
	if s == "" {
		return nil, nil
	}
	if !q.allowed(s) {
		return nil, q.invalid("выбран вариант, которого нет в списке")
	}
	return s, nil
}

func (q Question) coerceChecks(v any) (any, error) {
	items := stringList(v)
	out := make([]any, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		s := strings.TrimSpace(item)
		if s == "" || seen[s] {
			continue
		}
		if !q.allowed(s) {
			return nil, q.invalid("выбран вариант, которого нет в списке")
		}
		seen[s] = true
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil, nil
	}
	if min := intOf(q.Config["min_choices"], 0); min > 0 && len(out) < min {
		return nil, q.invalid(fmt.Sprintf("выберите не меньше %d вариантов", min))
	}
	if max := intOf(q.Config["max_choices"], 0); max > 0 && len(out) > max {
		return nil, q.invalid(fmt.Sprintf("выберите не больше %d вариантов", max))
	}
	return out, nil
}

// allowed — законен ли выбранный вариант. Свой текст проходит, только если у
// вопроса включено «Другое».
func (q Question) allowed(value string) bool {
	for _, o := range q.Options() {
		if o == value {
			return true
		}
	}
	return q.OptionOther()
}

func (q Question) coerceNumber(v any) (any, error) {
	n, ok := numberOf(v)
	if !ok {
		return nil, nil
	}
	min, max := 1, q.RatingMax()
	if q.Type == QScale {
		min, max = q.Scale()
	}
	if n < float64(min) || n > float64(max) {
		return nil, q.invalid(fmt.Sprintf("значение вне диапазона %d…%d", min, max))
	}
	return int(n), nil
}

func (q Question) coerceGrid(v any) (any, error) {
	raw, ok := v.(map[string]any)
	if !ok {
		return nil, nil
	}
	rows, cols := q.GridRows(), q.GridCols()
	rowSet, colSet := set(rows), set(cols)
	out := map[string]any{}
	for row, value := range raw {
		if !rowSet[row] {
			continue // строку убрали из сетки — старое значение отбрасываем
		}
		if q.Type == QGridRadio {
			s := strings.TrimSpace(stringOf(value))
			if s == "" {
				continue
			}
			if !colSet[s] {
				return nil, q.invalid("выбран столбец, которого нет в сетке")
			}
			out[row] = s
			continue
		}
		picked := make([]any, 0, len(cols))
		seen := map[string]bool{}
		for _, c := range stringList(value) {
			if c == "" || seen[c] {
				continue
			}
			if !colSet[c] {
				return nil, q.invalid("выбран столбец, которого нет в сетке")
			}
			seen[c] = true
			picked = append(picked, c)
		}
		if len(picked) > 0 {
			out[row] = picked
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	// «Ответ в каждой строке» — свойство самого вопроса, а не обязательности:
	// заполнил половину сетки — значит не ответил.
	if q.RequireEachRow() && len(out) < len(rows) {
		return nil, q.invalid("ответьте в каждой строке")
	}
	return out, nil
}

func (q Question) coerceDate(v any) (any, error) {
	s := strings.TrimSpace(stringOf(v))
	if s == "" {
		return nil, nil
	}
	s = strings.Replace(s, " ", "T", 1)
	if q.WithTime() {
		if !dateTmRe.MatchString(s) {
			return nil, q.invalid("укажите дату и время")
		}
		return s, nil
	}
	if !dateRe.MatchString(s) {
		return nil, q.invalid("укажите дату в виде ГГГГ-ММ-ДД")
	}
	return s, nil
}

// coerceFiles — файлы уже загружены отдельной ручкой, здесь остаются их
// метаданные. Ключ хранилища клиент подделать не может: файл с чужим путём
// просто не найдётся при выдаче, а лишние ключи мы отбрасываем.
func (q Question) coerceFiles(v any) (any, error) {
	list, ok := v.([]any)
	if !ok {
		return nil, nil
	}
	count, _ := q.FileLimits()
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		path := strings.TrimSpace(stringOf(m["path"]))
		if path == "" {
			continue
		}
		file := map[string]any{
			"path": path,
			"name": stringOf(m["name"]),
			"mime": stringOf(m["mime"]),
		}
		if size, ok := numberOf(m["size"]); ok {
			file["size"] = int64(size)
		}
		if thumb := strings.TrimSpace(stringOf(m["thumb"])); thumb != "" {
			file["thumb"] = thumb
		}
		out = append(out, file)
	}
	if len(out) == 0 {
		return nil, nil
	}
	if len(out) > count {
		return nil, q.invalid(fmt.Sprintf("можно приложить не больше %d файлов", count))
	}
	return out, nil
}

// Filled — есть ли в значении ответ (проверка обязательности).
func Filled(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(x) != ""
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

// AnswerText — текстовое представление ответа: строка поиска, ячейка выгрузки и
// колонка таблицы ответов (зеркало utils/formFields.js:answerText).
func AnswerText(q Question, v any) string {
	if v == nil {
		return ""
	}
	switch q.Type {
	case QCheckbox:
		return strings.Join(stringList(v), ", ")
	case QGridRadio, QGridCheckbox:
		raw, _ := v.(map[string]any)
		parts := make([]string, 0, len(raw))
		// Порядок строк — как в сетке: карта в Go неупорядочена, а колонка
		// выгрузки не должна прыгать от ответа к ответу.
		for _, row := range q.GridRows() {
			value, ok := raw[row]
			if !ok {
				continue
			}
			if q.Type == QGridRadio {
				parts = append(parts, row+": "+stringOf(value))
				continue
			}
			parts = append(parts, row+": "+strings.Join(stringList(value), ", "))
		}
		return strings.Join(parts, "; ")
	case QFile:
		list, _ := v.([]any)
		names := make([]string, 0, len(list))
		for _, item := range list {
			m, _ := item.(map[string]any)
			if name := stringOf(m["name"]); name != "" {
				names = append(names, name)
			}
		}
		return strings.Join(names, ", ")
	case QDate:
		return strings.Replace(stringOf(v), "T", " ", 1)
	default:
		return stringOf(v)
	}
}

// SearchText — сквозная строка поиска по ответу.
func SearchText(questions []Question, answers map[string]any) string {
	var b strings.Builder
	for _, q := range questions {
		v, ok := answers[QuestionID(q.ID)]
		if !ok {
			continue
		}
		if part := AnswerText(q, v); part != "" {
			b.WriteString(part)
			b.WriteByte(' ')
		}
	}
	return strings.TrimSpace(b.String())
}

// AnswerFiles — файлы, приложенные к ответу (чистка хранилища и раздел
// «Настройки → Хранилище»).
func AnswerFiles(answers map[string]any) []UploadedFile {
	out := []UploadedFile{}
	for _, v := range answers {
		list, ok := v.([]any)
		if !ok {
			continue
		}
		for _, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			path := stringOf(m["path"])
			if path == "" {
				continue
			}
			size, _ := numberOf(m["size"])
			out = append(out, UploadedFile{
				Path: path, Name: stringOf(m["name"]), Mime: stringOf(m["mime"]),
				Size: int64(size), Thumb: stringOf(m["thumb"]),
			})
		}
	}
	return out
}

// AnswersWithoutFiles — ответы без файлов с перечисленными ключами (раздел
// «Хранилище» удаляет файл, сам ответ остаётся). Второе значение — менялось ли
// что-нибудь, третье — какие ключи ушли.
func AnswersWithoutFiles(answers map[string]any, drop map[string]bool) (map[string]any, bool, []string) {
	removed := []string{}
	out := make(map[string]any, len(answers))
	for key, v := range answers {
		list, ok := v.([]any)
		if !ok {
			out[key] = v
			continue
		}
		kept := make([]any, 0, len(list))
		for _, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			path := stringOf(m["path"])
			if path != "" && drop[path] {
				removed = append(removed, path)
				if thumb := stringOf(m["thumb"]); thumb != "" {
					removed = append(removed, thumb)
				}
				continue
			}
			kept = append(kept, item)
		}
		if len(kept) > 0 {
			out[key] = kept
		}
	}
	if len(removed) == 0 {
		return answers, false, nil
	}
	return out, true, removed
}

// ── Режим теста ──────────────────────────────────────────────────

// Grade — сколько баллов даёт ответ. Проверяется только то, что можно сверить
// однозначно: выбор, сетка, число и текст из списка принимаемых.
func Grade(q Question, v any) int {
	if q.Points <= 0 || len(q.AnswerKey) == 0 {
		return 0
	}
	switch q.Type {
	case QRadio, QDropdown:
		if strings.EqualFold(stringOf(v), stringOf(q.AnswerKey["value"])) {
			return q.Points
		}
	case QShortText:
		got := strings.TrimSpace(strings.ToLower(stringOf(v)))
		if got == "" {
			return 0
		}
		for _, want := range stringList(q.AnswerKey["values"]) {
			if got == strings.TrimSpace(strings.ToLower(want)) {
				return q.Points
			}
		}
	case QCheckbox:
		want := set(stringList(q.AnswerKey["values"]))
		got := stringList(v)
		if len(want) == 0 || len(got) != len(want) {
			return 0
		}
		for _, g := range got {
			if !want[g] {
				return 0
			}
		}
		return q.Points
	case QScale, QRating:
		want, ok := numberOf(q.AnswerKey["number"])
		got, gotOK := numberOf(v)
		if ok && gotOK && want == got {
			return q.Points
		}
	case QGridRadio, QGridCheckbox:
		want, _ := q.AnswerKey["grid"].(map[string]any)
		got, _ := v.(map[string]any)
		if len(want) == 0 {
			return 0
		}
		for row, wantVal := range want {
			gotVal, ok := got[row]
			if !ok {
				return 0
			}
			if q.Type == QGridRadio {
				if stringOf(gotVal) != stringOf(wantVal) {
					return 0
				}
				continue
			}
			wantSet := set(stringList(wantVal))
			gotList := stringList(gotVal)
			if len(gotList) != len(wantSet) {
				return 0
			}
			for _, g := range gotList {
				if !wantSet[g] {
					return 0
				}
			}
		}
		return q.Points
	}
	return 0
}

// MaxScore — максимум баллов формы.
func MaxScore(questions []Question) int {
	total := 0
	for _, q := range questions {
		if q.Points > 0 {
			total += q.Points
		}
	}
	return total
}

// ── Мелочи разбора JSON ──────────────────────────────────────────
//
// Значения ответов приезжают из JSONB, поэтому число бывает и float64, и
// строкой. Эти три обёртки — единственный способ их читать: и здесь, и в
// сервисе (сводка, выгрузка).

// Text — значение строкой.
func Text(v any) string { return stringOf(v) }

// List — значение списком строк (одиночное значение — список из одного).
func List(v any) []string { return stringList(v) }

// Number — значение числом; ok=false, если это не число.
func Number(v any) (float64, bool) { return numberOf(v) }

func stringOf(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case bool:
		return strconv.FormatBool(x)
	case nil:
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func stringList(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s := stringOf(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if x == "" {
			return nil
		}
		return []string{x}
	}
	return nil
}

func numberOf(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		s := strings.TrimSpace(x)
		if s == "" || !numRe.MatchString(s) {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		return f, err == nil
	}
	return 0, false
}

func intOf(v any, def int) int {
	if n, ok := numberOf(v); ok {
		return int(n)
	}
	return def
}

func boolOf(v any) bool {
	b, _ := v.(bool)
	return b
}

func set(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, s := range items {
		out[s] = true
	}
	return out
}

// trimList — обрезать набор до потолка и выкинуть пустые строки. Результат —
// []any: он уезжает обратно в JSONB.
func trimList(items []string, max int) []any {
	out := make([]any, 0, len(items))
	for _, s := range items {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
		if len(out) >= max {
			break
		}
	}
	return out
}

func trimNum(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// invalid — ошибка значения с названием вопроса: человеку важно, ГДЕ он
// ошибся, а вопросов на странице много.
func (q Question) invalid(msg string) error {
	title := strings.TrimSpace(q.Title)
	if title == "" {
		title = "Вопрос"
	}
	return NewError("VALIDATION", "«"+title+"»: "+msg, 400)
}
