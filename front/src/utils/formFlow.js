/* Маршрут по разделам формы.

   Форма ведёт отвечающего не подряд, а по ветвлению: выбранный вариант может
   увести на другой раздел или сразу к отправке. Здесь — та же логика, что и на
   сервере (back-go/forms/internal/domain/flow.go): клиент по ней показывает
   страницы и считает прогресс, сервер — решает, какие обязательные вопросы
   спрашивать. Держать в паре.

   Повторно встреченный раздел обрывает маршрут: составитель волен закольцевать
   переходы, но заполнение обязано завершаться. */

import { isBranching, isFilled, visibleCondition } from '@/utils/formFields.js'

/* Условное отображение: раздел или вопрос выводится, только если на
   вопрос-источник дан один из ожидаемых ответов (пустой список — «любой
   непустой ответ»). Зеркало domain.Visible. */
function conditionMet(sourceId, want = [], answers = {}) {
  const got = answers[String(sourceId)]
  if (!isFilled(got)) return false
  if (!want.length) return true
  const chosen = Array.isArray(got) ? got : [got]
  return want.some((w) => chosen.includes(w))
}

export function questionVisible(question, answers) {
  const cond = visibleCondition(question)
  return !cond || conditionMet(cond.questionId, cond.values, answers)
}

export function sectionVisible(section, answers) {
  if (!section?.visible_question_id) return true
  return conditionMet(section.visible_question_id, section.visible_values || [], answers)
}

// visibleQuestions — вопросы раздела, прошедшие своё условие показа.
export function visibleQuestions(section, answers) {
  return (section?.questions || []).filter((q) => questionVisible(q, answers))
}

// visitedSections — разделы, которые проходит отвечающий с такими ответами.
export function visitedSections(sections = [], answers = {}) {
  if (!sections.length) return []
  const byId = new Map(sections.map((s, i) => [String(s.id), i]))
  const out = []
  const seen = new Set()

  let i = 0
  while (i >= 0 && i < sections.length) {
    const section = sections[i]
    if (seen.has(section.id)) break
    seen.add(section.id)
    // Скрытый условием раздел пропускаем, но переход берём его же: он остаётся
    // частью маршрута, просто ничего не показывает.
    if (sectionVisible(section, answers)) out.push(section)

    const target = nextTarget(section, answers)
    if (target === 'submit') return out
    if (!target) {
      i += 1
      continue
    }
    const next = byId.get(String(target))
    i = next == null ? i + 1 : next
  }
  return out
}

/* nextTarget — куда ведёт раздел при таких ответах. Приоритет у вопроса: если
   отвечен вопрос с переходом по варианту, решает он (последний по порядку);
   иначе действует переход самого раздела. */
function nextTarget(section, answers) {
  let target = ''
  for (const q of section.questions || []) {
    if (!isBranching(q.type)) continue
    const chosen = answers[String(q.id)]
    if (!chosen) continue
    const t = q.config?.targets?.[chosen]
    if (t && t !== 'next') target = String(t)
  }
  if (target) return target
  if (section.next_action === 'submit') return 'submit'
  if (section.next_action === 'section' && section.next_section_id) {
    return String(section.next_section_id)
  }
  return ''
}

// nextSectionIndex — индекс следующего раздела в исходном массиве (-1 —
// дальше отправка формы).
export function nextSectionIndex(sections, current, answers) {
  const section = sections[current]
  if (!section) return -1
  const target = nextTarget(section, answers)
  if (target === 'submit') return -1
  if (!target) return current + 1 < sections.length ? current + 1 : -1
  const idx = sections.findIndex((s) => String(s.id) === String(target))
  return idx === -1 ? (current + 1 < sections.length ? current + 1 : -1) : idx
}

// allQuestions — плоский список вопросов формы в порядке разделов.
export function allQuestions(sections = []) {
  return sections.flatMap((s) => s.questions || [])
}
