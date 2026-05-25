/* Форматирование статуса присутствия для мессенджера. */

function timeStr(d) {
  return d.toLocaleTimeString('ru', { hour: '2-digit', minute: '2-digit' })
}

/* Точная дата+время последнего захода. Примеры:
   «был(а) в сети сегодня в 14:32», «… вчера в 09:05», «… 23.05.2026 в 18:40».
   Пол неизвестен — пишем нейтральное «был(а)». */
export function formatLastSeen(iso) {
  if (!iso) return 'не в сети'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return 'не в сети'
  const now = new Date()
  const sameDay = d.toDateString() === now.toDateString()
  if (sameDay) return `был(а) в сети сегодня в ${timeStr(d)}`
  const yest = new Date(now)
  yest.setDate(now.getDate() - 1)
  if (d.toDateString() === yest.toDateString()) return `был(а) в сети вчера в ${timeStr(d)}`
  const date = d.toLocaleDateString('ru', { day: '2-digit', month: '2-digit', year: 'numeric' })
  return `был(а) в сети ${date} в ${timeStr(d)}`
}
