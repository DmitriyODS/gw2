/**
 * Разбивает текст сообщения на сегменты: обычный текст и ссылки.
 * Используется в MessageBubble, чтобы рендерить URL как активные <a>.
 *
 * Поддерживаем http(s)://… и www.… (последнему подставляем https://).
 * Хвостовая пунктуация (.,;:!?) и закрывающие скобки/кавычки не съедаются
 * в ссылку — частый случай «зайди на https://site.ru.» или «(см. http://x)».
 */
const URL_RE = /((?:https?:\/\/|www\.)[^\s<]+)/gi
const TRAILING_RE = /[.,;:!?)\]}'"»…]+$/

export function linkifyParts(text) {
  if (!text) return []
  const parts = []
  let last = 0
  let m
  URL_RE.lastIndex = 0
  while ((m = URL_RE.exec(text)) !== null) {
    let raw = m[0]
    let offset = m.index
    // Откусываем хвостовую пунктуацию обратно в обычный текст.
    const trailing = raw.match(TRAILING_RE)
    let tail = ''
    if (trailing) {
      // Не трогаем закрывающую скобку, если в самой ссылке есть открывающая
      // (например, ссылки на Wikipedia вида …(disambiguation)).
      tail = trailing[0]
      raw = raw.slice(0, raw.length - tail.length)
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
