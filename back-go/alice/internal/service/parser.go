// Интент-парсер русских голосовых команд. Работает по request.command
// (Диалоги уже отдают его в нижнем регистре без пунктуации), без LLM —
// у вебхука жёсткий бюджет времени ответа.
package service

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/alice/internal/domain"
)

// Intent — распознанная команда (см. domain.Intent).
type Intent = domain.Intent

// normalize — нижний регистр, ё→е, схлопнутые пробелы (command Диалогов уже
// без пунктуации; normalize нужен и для наших сравнений названий).
func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "ё", "е")
	return strings.Join(strings.Fields(s), " ")
}

var (
	reHelp       = regexp.MustCompile(`^(помощь|справка|что ты умеешь|помоги|как пользоваться)$`)
	reYes        = regexp.MustCompile(`^(да|ага|угу|конечно|давай|подтверждаю|верно|точно|подтвердить)$`)
	reNo         = regexp.MustCompile(`^(нет|отмена|отмени|не надо|не нужно|передумал|передумала)$`)
	reTaskCreate = regexp.MustCompile(`^(?:добавь|создай|заведи|поставь|создать|добавить|новая) (?:новую )?задач[ау] (.+)$`)
	reTaskClose  = regexp.MustCompile(`^(?:закрой|заверши|выполни|закончи|закрыть|завершить) задачу (.+)$`)
	reTaskList   = regexp.MustCompile(`^(?:мои задачи|список задач|задачи|какие (?:у меня )?задачи.*|что у меня по задачам)$`)
	reUnitStart  = regexp.MustCompile(`^(?:начни|начать|запусти|запустить|стартуй|начинай) (?:работу|работать|юнит)(?: (?:над|по))?(?: задач\S*)? (.+)$`)
	reUnitStop   = regexp.MustCompile(`^(?:останови|заверши|закончи|прекрати|стоп)(?: (?:работу|юнит))?$`)
	reUnitStatus = regexp.MustCompile(`^(?:что (?:сейчас )?в работе|статус(?: работы| юнита)?|какой юнит(?: активен)?)$`)

	reDiaryCreate = regexp.MustCompile(`^(?:создай|заведи|добавь|новый) ежедневник (.+)$`)
	reDiaryList   = regexp.MustCompile(`^(?:что у меня|какие планы|планы|мои планы|расписание|мои дела|дела|что в ежедневнике|что запланировано)(?: в ежедневнике)?( на .+)?$`)
	reDiaryDone   = regexp.MustCompile(`^(?:отметь|вычеркни|отметить) (.+)$`)
	reDiaryDone2  = regexp.MustCompile(`^(?:выполни|сделал|сделала|заверши) (?:дело|запись|пункт) (.+)$`)
	reDoneSuffix  = regexp.MustCompile(` (?:выполненн?(?:ым|ой|ое)|сделанн?(?:ым|ой|ое)|как выполнено|выполнено)$`)
	reDiaryMove   = regexp.MustCompile(`^перенеси (.+)$`)
	reDiaryDelete = regexp.MustCompile(`^удали (?:запись|дело|пункт) (.+)$`)

	reNoteAppend   = regexp.MustCompile(`^(?:допиши|добавь) (?:в заметку|к заметке) (.+)$`)
	reNoteCreate   = regexp.MustCompile(`^(?:создай|добавь|заведи|новая) заметку (.+)$`)
	reNoteRead     = regexp.MustCompile(`^(?:прочитай|прочти|зачитай|открой|покажи) заметку (.+)$`)
	reNoteDelete   = regexp.MustCompile(`^удали заметку (.+)$`)
	reFolderCreate = regexp.MustCompile(`^(?:создай|добавь|заведи) папку (.+)$`)

	reDiaryAdd = regexp.MustCompile(`^(?:добавь|запиши|внеси|запланируй) (?:в ежедневник )?(.+)$`)
)

// splitBody — отделить тело от названия: «… с текстом <тело>» / «… текст <тело>».
func splitBody(s string) (title, text string) {
	for _, sep := range []string{" с текстом ", " со словами ", " текст "} {
		if i := strings.Index(s, sep); i > 0 {
			return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+len(sep):])
		}
	}
	return s, ""
}

// Parse — интент из нормализованной команды. now — «сейчас» в таймзоне
// пользователя (для дат «завтра»/«в пятницу»).
func Parse(command string, now time.Time) Intent {
	c := normalize(command)
	if c == "" {
		return Intent{Kind: "greet"}
	}
	switch {
	case reHelp.MatchString(c):
		return Intent{Kind: "help"}
	case reYes.MatchString(c):
		return Intent{Kind: "yes"}
	case reNo.MatchString(c):
		return Intent{Kind: "no"}
	}

	if m := reTaskCreate.FindStringSubmatch(c); m != nil {
		return Intent{Kind: "task_create", Title: m[1]}
	}
	if m := reTaskClose.FindStringSubmatch(c); m != nil {
		return Intent{Kind: "task_close", Title: m[1]}
	}
	if reTaskList.MatchString(c) {
		return Intent{Kind: "task_list"}
	}
	if m := reUnitStart.FindStringSubmatch(c); m != nil {
		return Intent{Kind: "unit_start", Title: m[1]}
	}
	if reUnitStop.MatchString(c) {
		return Intent{Kind: "unit_stop"}
	}
	if reUnitStatus.MatchString(c) {
		return Intent{Kind: "unit_status"}
	}

	if m := reDiaryCreate.FindStringSubmatch(c); m != nil {
		return Intent{Kind: "diary_create", Title: m[1]}
	}
	if reDiaryList.MatchString(c) {
		date, _, _ := ExtractDate(c, now)
		return Intent{Kind: "diary_list", Date: date}
	}
	if m := reDiaryDelete.FindStringSubmatch(c); m != nil {
		return Intent{Kind: "diary_delete", Title: m[1]}
	}
	if m := reDiaryMove.FindStringSubmatch(c); m != nil {
		date, cleaned, ok := ExtractDate(m[1], now)
		it := Intent{Kind: "diary_move", Title: cleaned}
		if ok {
			it.Date = date
		}
		return it
	}
	if m := reDiaryDone.FindStringSubmatch(c); m != nil {
		return Intent{Kind: "diary_done", Title: strings.TrimSpace(reDoneSuffix.ReplaceAllString(m[1], ""))}
	}
	if m := reDiaryDone2.FindStringSubmatch(c); m != nil {
		return Intent{Kind: "diary_done", Title: strings.TrimSpace(reDoneSuffix.ReplaceAllString(m[1], ""))}
	}

	if m := reNoteAppend.FindStringSubmatch(c); m != nil {
		title, text := splitBody(m[1])
		return Intent{Kind: "note_append", Title: title, Text: text}
	}
	if m := reNoteCreate.FindStringSubmatch(c); m != nil {
		title, text := splitBody(m[1])
		return Intent{Kind: "note_create", Title: title, Text: text}
	}
	if m := reNoteRead.FindStringSubmatch(c); m != nil {
		return Intent{Kind: "note_read", Title: m[1]}
	}
	if m := reNoteDelete.FindStringSubmatch(c); m != nil {
		return Intent{Kind: "note_delete", Title: m[1]}
	}
	if m := reFolderCreate.FindStringSubmatch(c); m != nil {
		return Intent{Kind: "folder_create", Title: m[1]}
	}

	// Общий «добавь/запиши …» — запись ежедневника (после всех специфичных).
	if m := reDiaryAdd.FindStringSubmatch(c); m != nil {
		date, cleaned, ok := ExtractDate(m[1], now)
		it := Intent{Kind: "diary_add", Title: cleaned}
		if ok {
			it.Date = date
		}
		return it
	}

	return Intent{Kind: "unknown"}
}

// ── Разбор ответа на уточняющий вопрос ──

var numberWords = map[string]int{
	"один": 1, "первый": 1, "первая": 1, "первую": 1, "первое": 1,
	"два": 2, "второй": 2, "вторая": 2, "вторую": 2, "второе": 2,
	"три": 3, "третий": 3, "третья": 3, "третью": 3, "третье": 3,
	"четыре": 4, "четвертый": 4, "четвертая": 4, "четвертую": 4,
	"пять": 5, "пятый": 5, "пятая": 5, "пятую": 5,
}

// ParseChoiceIndex — номер варианта из ответа («2», «второй», «вариант два»);
// 0 — номера нет.
func ParseChoiceIndex(answer string) int {
	for _, w := range strings.Fields(normalize(answer)) {
		if n, err := strconv.Atoi(w); err == nil && n > 0 {
			return n
		}
		if n, ok := numberWords[w]; ok {
			return n
		}
	}
	return 0
}

// wordsMatch — все слова needle встречаются в hay (подстрочно); пустой
// needle — false.
func wordsMatch(needle, hay string) bool {
	needle, hay = normalize(needle), normalize(hay)
	if needle == "" {
		return false
	}
	for _, w := range strings.Fields(needle) {
		if !strings.Contains(hay, w) {
			return false
		}
	}
	return true
}
