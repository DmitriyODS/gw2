/* Цикличность расписания — ЗЕРКАЛО back-go/schedule/internal/domain/cycle.go.

   Занятие говорит, когда оно идёт, одним из двух взаимоисключающих способов:
     1. номера недель цикла (`weeks`; пусто — каждую неделю) — прямое обобщение
        числителя и знаменателя;
     2. своё правило (`repeat_every` + `repeat_from`, необязательный
        `repeat_until`) — «каждые N недель с такой-то даты», мимо цикла.

   Расчёт идёт по UTC-полуночи: перевод часов не должен сдвигать границу недели.
   Меняете правило здесь — меняйте и в Go, тестовые кейсы у них общие. */

export const MAX_CYCLE_WEEKS = 4

/** Дни недели, начиная с понедельника (0 — понедельник). */
export const WEEKDAYS = [
  { full: 'Понедельник', short: 'Пн' },
  { full: 'Вторник', short: 'Вт' },
  { full: 'Среда', short: 'Ср' },
  { full: 'Четверг', short: 'Чт' },
  { full: 'Пятница', short: 'Пт' },
  { full: 'Суббота', short: 'Сб' },
  { full: 'Воскресенье', short: 'Вс' },
]
const WEEK_MS = 7 * 24 * 3600 * 1000

/** Дата (Date | 'YYYY-MM-DD') → UTC-полночь того же дня. */
export function utcDay(value) {
  if (typeof value === 'string') {
    const [y, m, d] = value.split('-').map(Number)
    return new Date(Date.UTC(y, (m || 1) - 1, d || 1))
  }
  const d = value instanceof Date ? value : new Date(value)
  return new Date(Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()))
}

/** Понедельник недели, в которую попадает дата (UTC-полночь). */
export function mondayOf(value) {
  const d = utcDay(value)
  d.setUTCDate(d.getUTCDate() - ((d.getUTCDay() + 6) % 7))
  return d
}

/** День недели: 0 — понедельник … 6 — воскресенье. */
export function weekdayOf(value) {
  return (utcDay(value).getUTCDay() + 6) % 7
}

/** 'YYYY-MM-DD' для дат, которыми обменивается сервер. */
export function dateKey(value) {
  return utcDay(value).toISOString().slice(0, 10)
}

/** Дата плюс N дней (в UTC). */
export function addDays(value, days) {
  const d = utcDay(value)
  d.setUTCDate(d.getUTCDate() + days)
  return d
}

function weeksBetween(from, to) {
  return Math.round((mondayOf(to) - mondayOf(from)) / WEEK_MS)
}

/** Номер недели цикла (1..cycleWeeks) для даты; считается в обе стороны от якоря. */
export function weekIndex(anchor, date, cycleWeeks) {
  const n = cycleWeeks > 0 ? cycleWeeks : 1
  const diff = weeksBetween(anchor, date)
  return ((diff % n) + n) % n + 1
}

/** Название недели цикла: своё, если задано, иначе «Неделя N». */
export function weekLabel(schedule, index) {
  const own = (schedule?.week_labels || [])[index - 1]
  return (own || '').trim() || `Неделя ${index}`
}

/** Идёт ли занятие на неделе, в которую попадает дата. */
export function occursOn(schedule, item, date) {
  const monday = mondayOf(date)
  if (item.repeat_every > 0) {
    if (!item.repeat_from) return false
    const from = mondayOf(item.repeat_from)
    if (monday < from) return false
    if (item.repeat_until && monday > mondayOf(item.repeat_until)) return false
    return weeksBetween(from, monday) % item.repeat_every === 0
  }
  const weeks = item.weeks || []
  if (!weeks.length) return true
  return weeks.includes(weekIndex(schedule.cycle_anchor, monday, schedule.cycle_weeks))
}

/** Занятия расписания на конкретный день (день недели + неделя цикла). */
export function itemsOfDay(schedule, items, date) {
  const weekday = weekdayOf(date)
  return (items || [])
    .filter((it) => it.weekday === weekday && occursOn(schedule, it, date))
    .sort((a, b) => a.start_min - b.start_min || a.end_min - b.end_min)
}

/** Короткая подпись повтора занятия для карточки и списка. */
export function repeatLabel(schedule, item) {
  if (item.repeat_every > 0) {
    return item.repeat_every === 1 ? 'Каждую неделю' : `Каждые ${item.repeat_every} нед.`
  }
  const weeks = item.weeks || []
  if (!weeks.length || weeks.length === schedule.cycle_weeks) return 'Каждую неделю'
  return weeks.map((w) => weekLabel(schedule, w)).join(', ')
}

/** Минуты от полуночи → «ЧЧ:ММ». */
export function hhmm(min) {
  const m = Math.max(0, Math.round(min))
  return `${String(Math.floor(m / 60)).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`
}

/** «ЧЧ:ММ» → минуты от полуночи (мусор даёт 0). */
export function toMinutes(value) {
  const [h, m] = String(value || '').split(':').map(Number)
  if (!Number.isFinite(h) || !Number.isFinite(m)) return 0
  return Math.min(24 * 60, Math.max(0, h * 60 + m))
}

/** Длительность по-человечески: «1 ч 25», «40 мин». */
export function duration(min) {
  const h = Math.floor(min / 60)
  const m = min % 60
  if (!h) return `${m} мин`
  return m ? `${h} ч ${String(m).padStart(2, '0')}` : `${h} ч`
}
