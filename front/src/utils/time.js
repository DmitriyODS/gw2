/**
 * «только что» / «12 мин назад» / «3 ч назад» / «вчера» / дата — короткая
 * подпись давности для лент (последние действия рабочего стола).
 */
export function timeAgo(value) {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  const sec = Math.round((Date.now() - d.getTime()) / 1000)
  if (sec < 60) return 'только что'
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min} мин назад`
  const hours = Math.floor(min / 60)
  if (hours < 24) return `${hours} ч назад`
  const days = Math.floor(hours / 24)
  if (days === 1) return 'вчера'
  if (days < 7) return `${days} дн. назад`
  return d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })
}

export function formatHours(val) {
  if (val === null || val === undefined) return '0 мин'
  const totalMinutes = Math.round(val * 60)
  const h = Math.floor(totalMinutes / 60)
  const m = totalMinutes % 60
  if (h === 0) return `${m} мин`
  if (m === 0) return `${h} ч`
  return `${h} ч ${m} мин`
}
