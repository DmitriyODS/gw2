/**
 * Безопасный href из пользовательских данных: Vue и шаблонные строки адрес
 * не проверяют, и `javascript:…` в поле «ссылка» исполнился бы по клику у
 * любого, кто откроет запись.
 *
 * http(s)/mailto/tel и внутренний путь (`/…`) — как есть, адрес без схемы
 * («site.ru», «site.ru:8080/x») — с https://, любая другая схема — пустая
 * строка. Сервер отбивает такие значения при записи (`records.SafeLink`), здесь
 * — второй рубеж для старых данных.
 */
const SCHEME_RE = /^([a-z][a-z0-9+.-]*):(?![0-9])/i
const ALLOWED = new Set(['http', 'https', 'mailto', 'tel'])

export function safeHref(raw) {
  if (raw == null) return ''
  const url = String(raw).trim()
  if (!url) return ''
  // Браузер пропускает пробелы и управляющие символы внутри схемы
  // («java\tscript:»), поэтому схему ищем в строке без них.
  const compact = [...url].filter((ch) => ch.charCodeAt(0) > 32 && ch.charCodeAt(0) !== 127).join('')
  const m = SCHEME_RE.exec(compact)
  if (m) return ALLOWED.has(m[1].toLowerCase()) ? url : ''
  if (compact.startsWith('/')) return url
  return 'https://' + url
}
