export function formatHours(val) {
  if (val === null || val === undefined) return '0 мин'
  const totalMinutes = Math.round(val * 60)
  const h = Math.floor(totalMinutes / 60)
  const m = totalMinutes % 60
  if (h === 0) return `${m} мин`
  if (m === 0) return `${h} ч`
  return `${h} ч ${m} мин`
}
