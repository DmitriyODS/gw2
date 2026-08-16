package domain

import "strconv"

/* Маршрут по разделам.

   Форма ведёт отвечающего не подряд, а по ветвлению: выбранный вариант может
   увести на другой раздел или сразу к отправке. Значит и «обязательный вопрос»
   имеет смысл только для разделов, через которые человек РЕАЛЬНО прошёл, —
   иначе форма отвергала бы законный ответ из-за вопросов, которых он не видел.

   Маршрут считает СЕРВЕР (эта функция), клиент повторяет её для навигации
   (front/src/utils/formFlow.js) — держать в паре. */

/*
Visible — выполнено ли условие показа: на вопрос-источник дан один из

	ожидаемых ответов. Пустой список ожидаемых означает «любой непустой ответ»
	(частый случай: «покажи, если на предыдущий вопрос вообще ответили»).
	Значением бывает и список — тогда достаточно одного совпадения.
*/
func Visible(sourceID int64, want []string, answers map[string]any) bool {
	got := answers[QuestionID(sourceID)]
	if !Filled(got) {
		return false
	}
	if len(want) == 0 {
		return true
	}
	chosen := List(got)
	if len(chosen) == 0 {
		chosen = []string{Text(got)}
	}
	for _, w := range want {
		for _, c := range chosen {
			if c == w {
				return true
			}
		}
	}
	return false
}

// QuestionVisible — показывать ли вопрос при таких ответах.
func QuestionVisible(q Question, answers map[string]any) bool {
	sourceID, want, ok := q.VisibleIf()
	if !ok {
		return true
	}
	return Visible(sourceID, want, answers)
}

// SectionVisible — показывать ли раздел при таких ответах.
func SectionVisible(s Section, answers map[string]any) bool {
	if s.VisibleQuestionID == nil || *s.VisibleQuestionID <= 0 {
		return true
	}
	return Visible(*s.VisibleQuestionID, s.VisibleValues, answers)
}

// VisibleQuestions — вопросы раздела, прошедшие своё условие показа.
func VisibleQuestions(s Section, answers map[string]any) []Question {
	out := make([]Question, 0, len(s.Questions))
	for _, q := range s.Questions {
		if QuestionVisible(q, answers) {
			out = append(out, q)
		}
	}
	return out
}

// VisitedSections — разделы, которые проходит отвечающий с такими ответами, в
// порядке прохождения. Скрытый условием раздел пропускается, повторно
// встреченный обрывает маршрут: составитель формы волен закольцевать переходы,
// но заполнение обязано завершаться.
func VisitedSections(sections []Section, answers map[string]any) []Section {
	if len(sections) == 0 {
		return nil
	}
	byID := make(map[int64]int, len(sections))
	for i, s := range sections {
		byID[s.ID] = i
	}

	out := make([]Section, 0, len(sections))
	seen := make(map[int64]bool, len(sections))
	for i := 0; i >= 0 && i < len(sections); {
		section := sections[i]
		if seen[section.ID] {
			break
		}
		seen[section.ID] = true
		// Скрытый условием раздел пропускаем, но переход берём его же: он
		// остаётся частью маршрута, просто ничего не показывает.
		if SectionVisible(section, answers) {
			out = append(out, section)
		}

		switch target := nextTarget(section, answers); target {
		case NextSubmit:
			return out
		case "":
			i++
		default:
			id, err := strconv.ParseInt(target, 10, 64)
			if err != nil {
				i++
				continue
			}
			next, ok := byID[id]
			if !ok {
				i++
				continue
			}
			i = next
		}
	}
	return out
}

/*
nextTarget — куда ведёт раздел при таких ответах.

	Приоритет у вопроса: если в разделе отвечен вопрос с переходом по варианту,
	решает он (последний по порядку — так же, как в исходном образце). Иначе
	действует переход самого раздела.
*/
func nextTarget(section Section, answers map[string]any) string {
	target := ""
	for _, q := range section.Questions {
		if !Branching(q.Type) {
			continue
		}
		chosen := stringOf(answers[QuestionID(q.ID)])
		if chosen == "" {
			continue
		}
		if t := q.Target(chosen); t != "" {
			target = t
		}
	}
	if target != "" {
		return target
	}
	switch section.NextAction {
	case NextSubmit:
		return NextSubmit
	case NextSection:
		if section.NextSectionID != nil {
			return strconv.FormatInt(*section.NextSectionID, 10)
		}
	}
	return ""
}

/* Переход, записанный ПОЗИЦИЕЙ раздела ("#2").

   Так ветвление приезжает при сохранении структуры: у только что добавленного
   раздела id ещё нет. Репозиторий переводит позиции в id той же транзакцией, а
   наружу переходы всегда уходят уже идентификаторами. */

func TargetIndex(i int) string { return "#" + strconv.Itoa(i) }

// ParseTargetIndex — позиция раздела из такой записи; ok=false — это не позиция.
func ParseTargetIndex(s string) (int, bool) {
	if len(s) < 2 || s[0] != '#' {
		return 0, false
	}
	i, err := strconv.Atoi(s[1:])
	if err != nil || i < 0 {
		return 0, false
	}
	return i, true
}

// MissingRequired — первый незаполненный обязательный вопрос пройденного
// маршрута ("" — всё на месте). Возвращает название: человеку нужно знать, что
// именно он пропустил.
func MissingRequired(sections []Section, answers map[string]any) string {
	for _, section := range VisitedSections(sections, answers) {
		for _, q := range VisibleQuestions(section, answers) {
			if !q.Required || !Answerable(q.Type) {
				continue
			}
			if !Filled(answers[QuestionID(q.ID)]) {
				title := q.Title
				if title == "" {
					title = "Вопрос без названия"
				}
				return title
			}
		}
	}
	return ""
}

// AnsweredQuestions — вопросы, которые отвечающий реально видел при таких
// ответах: пройденный маршрут разделов и внутри них — прошедшие условие показа.
// По ним же чистятся значения скрытых вопросов перед сохранением.
func AnsweredQuestions(sections []Section, answers map[string]any) []Question {
	out := []Question{}
	for _, section := range VisitedSections(sections, answers) {
		out = append(out, VisibleQuestions(section, answers)...)
	}
	return out
}

// AllQuestions — плоский список вопросов формы в порядке разделов (выгрузка,
// сводка, поиск).
func AllQuestions(sections []Section) []Question {
	out := []Question{}
	for _, s := range sections {
		out = append(out, s.Questions...)
	}
	return out
}
