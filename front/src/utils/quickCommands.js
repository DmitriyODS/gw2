/**
 * Разбор быстрых команд строки поиска рабочего стола.
 *
 * «создай задачу институт что-то там» → { kind: 'task', title: 'Институт что-то там' }
 * Название — всё, что после существительного; кавычки снимаем, первую букву
 * поднимаем в верхний регистр (диктуют голосом и печатают строчными).
 *
 * Отдельные формы:
 *   • напоминание — из названия вынимается срок («завтра в 9», «через час»);
 *   • сообщение — адресат и текст разбираются не здесь: имя во фразе стоит в
 *     дательном падеже («напиши Васе»), сопоставить его можно только со
 *     списком живых собеседников (см. utils/recipients.js).
 */
import { extractWhen } from './naturalDate.js'

const VERB = '(?:созда(?:й|йте|ть|м)|добав(?:ь|ьте|ить|лю)|запиш(?:и|ите)|нов(?:ая|ую|ый|ое)|create|add|new)'

const KINDS = [
  { kind: 'task', noun: '(?:задач(?:у|а|и|ку)|таск|task)' },
  { kind: 'note', noun: '(?:заметк(?:у|а|и|ой)|заметочку|note)' },
  { kind: 'board', noun: '(?:доск(?:у|а|и)|board)' },
  { kind: 'reminder', noun: '(?:напоминани(?:е|я|й)|напоминалк(?:у|а|и)|будильник|reminder)' },
]

/* Разделитель между существительным и названием: пробел, двоеточие или тире.
   Границу слова `\b` не используем — в JS она считает по ASCII и после
   кириллицы не срабатывает. */
const RULES = KINDS.map((k) => ({
  kind: k.kind,
  re: new RegExp(`^\\s*${VERB}\\s+${k.noun}(?:[\\s:—–-]+(.*))?$`, 'i'),
}))

// «напомни мне позвонить в банк завтра в 9» — своя форма без существительного.
const RE_REMIND = /^\s*напомн(?:и|ите|ить)(?:\s+мне)?(?:[\s:—–-]+(.*))?$/i

// «напиши Васе созвон в 15:00» — адресата и текст отделяет utils/recipients.js.
const RE_MESSAGE = /^\s*(?:напиши(?:те)?|написать|отправ(?:ь|ьте|ить)|сообщи(?:те)?|write|msg|dm)\s+(.+)$/i

export function parseQuickCommand(input, now = new Date()) {
  const text = String(input || '')

  const msg = RE_MESSAGE.exec(text)
  if (msg) return { kind: 'message', rest: msg[1].trim() }

  for (const rule of RULES) {
    const m = rule.re.exec(text)
    if (m) return build(rule.kind, m[1], now)
  }

  const remind = RE_REMIND.exec(text)
  if (remind) return build('reminder', remind[1], now)

  return null
}

function build(kind, raw, now) {
  if (kind !== 'reminder') return { kind, title: cleanTitle(raw) }
  const { at, repeat, rest } = extractWhen(String(raw || ''), now)
  return { kind, title: cleanTitle(rest), at, repeat }
}

function cleanTitle(raw) {
  const t = String(raw || '').trim().replace(/^["'«“](.*)["'»”]$/s, '$1').trim()
  return t ? t[0].toLocaleUpperCase('ru') + t.slice(1) : ''
}
