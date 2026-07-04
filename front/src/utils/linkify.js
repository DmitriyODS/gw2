/**
 * Разбивает текст сообщения на сегменты: обычный текст и ссылки.
 * Используется в MessageBubble, чтобы рендерить URL как активные <a>.
 *
 * Поддерживаем http(s)://… и www.… (последнему подставляем https://).
 * Хвостовая пунктуация (.,;:!?) и закрывающие скобки/кавычки не съедаются
 * в ссылку — частый случай «зайди на https://site.ru.» или «(см. http://x)».
 */
const URL_RE = /((?:https?:\/\/|www\.)[^\s<]+)/gi
// Хвостовые символы, которые не должны попадать в ссылку. Закрывающие скобки
// тут НЕ перечислены — их разбираем отдельно, с учётом баланса.
const TRAILING_RE = /[.,;:!?'"»…]$/

export function linkifyParts(text) {
  if (!text) return []
  const parts = []
  let last = 0
  let m
  URL_RE.lastIndex = 0
  while ((m = URL_RE.exec(text)) !== null) {
    let raw = m[0]
    let offset = m.index
    // Откусываем хвостовую пунктуацию обратно в обычный текст. Закрывающую
    // скобку/квадратную скобку отрезаем, только если она непарная (нет
    // соответствующей открывающей в ссылке) — иначе рвём валидные URL вида
    // …/Foo_(bar) (Wikipedia).
    let tail = ''
    for (;;) {
      const ch = raw.slice(-1)
      if (TRAILING_RE.test(ch)) { tail = ch + tail; raw = raw.slice(0, -1); continue }
      if (ch === ')' && (raw.match(/\)/g)?.length || 0) > (raw.match(/\(/g)?.length || 0)) {
        tail = ch + tail; raw = raw.slice(0, -1); continue
      }
      if (ch === ']' && (raw.match(/]/g)?.length || 0) > (raw.match(/\[/g)?.length || 0)) {
        tail = ch + tail; raw = raw.slice(0, -1); continue
      }
      break
    }
    if (!raw) {
      continue
    }
    if (offset > last) {
      parts.push({ type: 'text', value: text.slice(last, offset) })
    }
    const href = raw.startsWith('www.') ? `https://${raw}` : raw
    parts.push({ type: 'link', value: raw, href })
    last = offset + raw.length
    if (tail) {
      parts.push({ type: 'text', value: tail })
      last += tail.length
    }
  }
  if (last < text.length) {
    parts.push({ type: 'text', value: text.slice(last) })
  }
  return parts
}
