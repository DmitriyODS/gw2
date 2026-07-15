// Разбор отсканированного/введённого кода спаривания устройств.
// QR кодирует URL вида `${origin}/link?code=ABC123`; вручную вводят сам код.

const CODE_RE = /^[A-Z2-9]{6}$/ // алфавит без похожих 0/O/1/I, длина 6

// normalizeLinkCode — верхний регистр, без пробелов/дефисов (ими код красиво
// показывают группами).
export function normalizeLinkCode(raw) {
  return String(raw || '')
    .toUpperCase()
    .replace(/[\s-]/g, '')
}

// extractLinkCode — вытащить код из отсканированной строки (URL или сам код).
// Возвращает нормализованный код или '' если это не наш код.
export function extractLinkCode(raw) {
  const str = String(raw || '').trim()
  if (!str) return ''
  // Строка-URL с ?code=...
  if (/code=/i.test(str)) {
    try {
      const url = new URL(str, window.location.origin)
      const code = normalizeLinkCode(url.searchParams.get('code'))
      return CODE_RE.test(code) ? code : ''
    } catch {
      /* не URL — попробуем как сырой код ниже */
    }
  }
  const code = normalizeLinkCode(str)
  return CODE_RE.test(code) ? code : ''
}
