/* Перевод «день ⇄ строка YYYY-MM-DD» для полей с общим DatePicker: он держит
   Date, а сервер почти везде ждёт день строкой. Считаем по ЛОКАЛЬНЫМ частям —
   toISOString даёт UTC-день, и восточнее Гринвича дата уезжает на сутки назад
   (выбрали 5 марта, ушло 4-е). */

export function dayString(date) {
  const d = date instanceof Date ? date : date ? new Date(date) : null
  if (!d || isNaN(d.getTime())) return ''
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function parseDay(value) {
  const [y, m, d] = String(value || '').split('-').map(Number)
  return y && m && d ? new Date(y, m - 1, d) : null
}
