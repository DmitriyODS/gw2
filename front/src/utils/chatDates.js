function startOfDay(d) {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate())
}

export function dayLabel(iso, now = new Date()) {
  const d = new Date(iso)
  const diffDays = Math.round((startOfDay(now) - startOfDay(d)) / 86400000)
  if (diffDays === 0) return 'Сегодня'
  if (diffDays === 1) return 'Вчера'
  const opts = { day: 'numeric', month: 'long' }
  if (d.getFullYear() !== now.getFullYear()) opts.year = 'numeric'
  return d.toLocaleDateString('ru-RU', opts).replace(/\s*г\.$/, '')
}

export function groupMessagesByDay(messages, now = new Date()) {
  const groups = []
  let cur = null
  for (const m of messages) {
    const d = new Date(m.created_at)
    const key = `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
    if (!cur || cur.key !== key) {
      cur = { key, label: dayLabel(m.created_at, now), items: [] }
      groups.push(cur)
    }
    cur.items.push(m)
  }
  return groups
}
