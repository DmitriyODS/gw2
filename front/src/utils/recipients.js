/**
 * Кому адресована фраза «напиши Васе созвон в 15:00».
 *
 * Имя во фразе стоит в дательном падеже («Васе», «Ивану», «Марии»), а в базе
 * лежит именительный — сравниваем по основе: отбрасываем окончание и
 * сопоставляем началом слова. Поэтому «васе» находит и «Василия», и
 * «Васнецову»: несколько кандидатов — не беда, их показывает список, выбирает
 * человек.
 */

const MAX_NAME_WORDS = 3

const norm = (s) => String(s || '').toLowerCase().replace(/ё/g, 'е')

/** Основа слова: «Васе» → «вас», «Ивану» → «иван», «Ли» → «ли». */
export function nameStem(word) {
  const w = norm(word).replace(/[^\p{L}\p{N}._-]/gu, '')
  return w.length >= 4 ? w.replace(/[аеиоуыэюяйьъ]$/u, '') : w
}

/** Основы всех слов имени; пустые и односимвольные отбрасываем. */
export function nameStems(name) {
  return norm(name).split(/[\s,]+/).map(nameStem).filter((s) => s.length >= 2)
}

/** Совпадает ли адресат: каждая основа запроса начинает какое-то его слово. */
export function matchesRecipient(names, stems) {
  if (!stems.length) return false
  const words = names
    .flatMap((n) => norm(n).split(/[\s._-]+/))
    .filter(Boolean)
  return stems.every((s) => words.some((w) => w.startsWith(s)))
}

/**
 * Варианты «где кончается имя и начинается текст». Явный разделитель (:/,)
 * главнее — иначе перебираем первые слова, длинные имена первыми:
 * «Иванову Петру привет» опознаётся как двусловное имя, а «Васе привет
 * Петрову» — как односложное.
 */
export function recipientSplits(rest) {
  const s = String(rest || '').trim()
  if (!s) return []
  const out = []

  const explicit = /^([^:,]+)[:,]\s*([\s\S]*)$/.exec(s)
  if (explicit && explicit[1].trim().split(/\s+/).length <= MAX_NAME_WORDS) {
    out.push({ name: explicit[1].trim(), text: explicit[2].trim() })
  }

  const words = s.split(/\s+/)
  for (let n = Math.min(MAX_NAME_WORDS, words.length); n >= 1; n--) {
    out.push({ name: words.slice(0, n).join(' '), text: words.slice(n).join(' ') })
  }
  return out
}

/**
 * Разобрать фразу по списку известных собеседников.
 * pool — [{ names: [ФИО, логин, …], … }]; возвращает первый разбор, у которого
 * нашлись адресаты (порядок pool сохраняется — свои диалоги идут раньше).
 */
export function resolveRecipients(rest, pool) {
  for (const split of recipientSplits(rest)) {
    const stems = nameStems(split.name)
    if (!stems.length) continue
    const matches = pool.filter((p) => matchesRecipient(p.names || [], stems))
    if (matches.length) return { ...split, matches }
  }
  return { name: recipientSplits(rest)[0]?.name || '', text: '', matches: [] }
}

/** Основа первого слова — с ней ищем адресата на сервере (ILIKE по началу). */
export function searchStem(rest) {
  return nameStems(String(rest || '').split(/\s+/)[0] || '')[0] || ''
}
