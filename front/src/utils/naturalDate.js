/**
 * Разбор человеческих сроков в русской фразе: «завтра в 9», «через 20 минут»,
 * «в пятницу», «15 июля в 10:30», «каждый день в 9:00».
 *
 * Родня серверному разбору дат навыка Алисы (back-go/alice/internal/service/
 * dates.go), но со временем суток и повторами: здесь из фразы рождается
 * напоминание, а ему нужен точный момент, а не день.
 */

// Родительный падеж — так месяц и звучит во фразе («15 июля»).
const MONTHS = [
  'января', 'февраля', 'марта', 'апреля', 'мая', 'июня',
  'июля', 'августа', 'сентября', 'октября', 'ноября', 'декабря',
]

// 1..7 (пн..вс) — как в повторах напоминаний.
const WEEKDAYS = {
  понедельник: 1, вторник: 2, среду: 3, среда: 3, четверг: 4,
  пятницу: 5, пятница: 5, субботу: 6, суббота: 6, воскресенье: 7, воскресение: 7,
}

const MONTH_ALT = MONTHS.join('|')
const WEEKDAY_ALT = Object.keys(WEEKDAYS).join('|')

/* Границу слова `\b` не используем — в JS она считает по ASCII и после
   кириллицы не срабатывает. Слева — группа (^|\s) (её длина нужна, чтобы
   вырезать ровно найденное), справа — просмотр вперёд. */
const REPEATS = [
  { re: /(^|\s)(?:каждый день|ежедневно)(?=\s|$)/, kind: 'daily' },
  { re: /(^|\s)каждые (\d+) дн\S*(?=\s|$)/, kind: 'daily', interval: 2 },
  { re: /(^|\s)(?:по будням|по рабочим дням)(?=\s|$)/, kind: 'weekdays' },
  { re: new RegExp(`(^|\\s)кажд(?:ую|ый) (${WEEKDAY_ALT})(?=\\s|$)`), kind: 'weekly', weekday: 2 },
  { re: /(^|\s)(?:кажд(?:ую|ые) недел\S*|еженедельно)(?=\s|$)/, kind: 'weekly' },
  { re: /(^|\s)каждые (\d+) недел\S*(?=\s|$)/, kind: 'weekly', interval: 2 },
  { re: /(^|\s)(?:кажд(?:ый|ые) месяц\S*|ежемесячно)(?=\s|$)/, kind: 'monthly' },
  { re: /(^|\s)(?:кажд(?:ый|ые) год\S*|ежегодно)(?=\s|$)/, kind: 'yearly' },
]

const RE_THROUGH = /(^|\s)через\s+(полчаса|полтора часа|час|сутки|недел\S*|\d+)(?:\s+(минут\S*|мин|час\S*|дн\S*|день|недел\S*))?(?=\s|$)/
const RE_RELDAY = /(^|\s)(?:на |в )?(сегодня|завтра|послезавтра)(?=\s|$)/
const RE_WEEKDAY = new RegExp(`(^|\\s)(?:в|во|на)\\s+(${WEEKDAY_ALT})(?=\\s|$)`)
const RE_DAYMON = new RegExp(`(^|\\s)(?:на\\s+)?(\\d{1,2})\\s+(${MONTH_ALT})(?=\\s|$)`)
/* Месяц двузначный намеренно: иначе «версия 1.2» в тексте читалась бы как
   1 февраля. «15.07» и «1.08» — дата, «1.2» — просто число. */
const RE_DMY = /(^|\s)(?:на\s+)?(\d{1,2})\.(\d{2})(?:\.(\d{2,4}))?(?=\s|$)/
const RE_HM = /(^|\s)(?:в\s+)?(\d{1,2})[:.](\d{2})(?=\s|$)/
const RE_HOUR = /(^|\s)в\s+(\d{1,2})(?:\s+(утра|дня|вечера|ночи))?(?=\s|$)/
const RE_DAYPART = /(^|\s)(утром|днем|вечером|ночью|в полдень|в полночь)(?=\s|$)/

const DAYPART_HOUR = { утром: 9, днем: 13, вечером: 18, ночью: 22, 'в полдень': 12, 'в полночь': 0 }

/** Нижний регистр + ё→е: длина строки не меняется, индексы совпадают с исходной. */
const norm = (s) => s.toLowerCase().replace(/ё/g, 'е')

/**
 * extractWhen — вынуть срок из фразы.
 * Возвращает { at: Date|null, repeat: {kind, interval, days}|null, rest }.
 * Срок не распознан — фраза возвращается нетронутой (её целиком показываем
 * как название: пусть недостающее время задаст пользователь).
 */
export function extractWhen(input, now = new Date()) {
  const state = { rest: String(input || '').trim() }
  let repeat = null
  let day = null      // Date — только календарный день
  let time = null     // { h, m }
  let minutes = null  // «через N» — сдвиг от «сейчас»
  let weekday = null  // номер дня недели из повтора «каждую пятницу»

  const take = (re, apply) => {
    const m = re.exec(norm(state.rest))
    if (!m) return false
    if (apply(m) === false) return false
    const start = m.index + m[1].length
    state.rest = (state.rest.slice(0, start) + ' ' + state.rest.slice(m.index + m[0].length))
      .replace(/\s+/g, ' ')
      .trim()
    return true
  }

  for (const r of REPEATS) {
    const hit = take(r.re, (m) => {
      repeat = {
        kind: r.kind,
        interval: r.interval ? Number(m[r.interval]) || 1 : 1,
        days: [],
      }
      if (r.weekday) weekday = WEEKDAYS[m[r.weekday]]
    })
    if (hit) break
  }

  take(RE_THROUGH, (m) => {
    const word = m[2]
    if (word === 'полчаса') { minutes = 30; return }
    if (word === 'полтора часа') { minutes = 90; return }
    // «через 20» без единицы измерения — не срок, а часть названия.
    if (!m[3] && /^\d+$/.test(word)) return false
    const unit = m[3] || (word === 'сутки' ? 'дн' : word.startsWith('недел') ? 'недел' : 'час')
    const n = /^\d+$/.test(word) ? Number(word) : 1
    if (!Number.isFinite(n) || n <= 0) return false
    if (unit.startsWith('мин')) minutes = n
    else if (unit.startsWith('час')) minutes = n * 60
    else if (unit.startsWith('дн') || unit === 'день') minutes = n * 1440
    else if (unit.startsWith('недел')) minutes = n * 7 * 1440
    else return false
  })

  if (minutes == null) {
    const found = take(RE_RELDAY, (m) => {
      day = startOfDay(now)
      day.setDate(day.getDate() + { сегодня: 0, завтра: 1, послезавтра: 2 }[m[2]])
    })
      || take(RE_WEEKDAY, (m) => { day = nextWeekday(now, WEEKDAYS[m[2]]) })
      || take(RE_DAYMON, (m) => {
        const d = Number(m[2])
        const month = MONTHS.indexOf(m[3])
        if (d < 1 || d > 31) return false
        day = comingDate(now, month, d)
      })
      || take(RE_DMY, (m) => {
        const d = Number(m[2])
        const month = Number(m[3]) - 1
        if (d < 1 || d > 31 || month < 0 || month > 11) return false
        if (m[4]) {
          const y = Number(m[4])
          day = startOfDay(new Date(y < 100 ? 2000 + y : y, month, d))
          if (Number.isNaN(day.getTime())) return false
        } else {
          day = comingDate(now, month, d)
        }
      })
    // Повтор «каждую пятницу» сам задаёт ближайший день, если его не назвали.
    if (!found && weekday) day = nextWeekday(now, weekday)
  }

  take(RE_HM, (m) => {
    const h = Number(m[2])
    const min = Number(m[3])
    if (h > 23 || min > 59) return false
    time = { h, m: min }
  })
    || take(RE_HOUR, (m) => {
      const h = applyDaypart(Number(m[2]), m[3])
      if (h == null) return false
      time = { h, m: 0 }
    })
    || take(RE_DAYPART, (m) => { time = { h: DAYPART_HOUR[m[2]], m: 0 } })

  const at = combine(now, { minutes, day, time })
  if (!at) return { at: null, repeat: null, rest: String(input || '').trim() }

  if (repeat) {
    repeat.days = repeat.kind === 'weekly'
      ? [weekday || isoWeekday(at)]
      : []
  }
  return { at, repeat, rest: state.rest }
}

/**
 * humanWhen — человеческий срок для подписи: «сегодня в 14:30», «завтра в 9:00»,
 * «12 авг в 10:00».
 */
export function humanWhen(value, now = new Date()) {
  const at = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(at.getTime())) return ''
  const time = at.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  const days = Math.round((startOfDay(at) - startOfDay(now)) / 86_400_000)
  if (days === 0) return `сегодня в ${time}`
  if (days === 1) return `завтра в ${time}`
  if (days === 2) return `послезавтра в ${time}`
  return `${at.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })} в ${time}`
}

function startOfDay(d) {
  const out = new Date(d)
  out.setHours(0, 0, 0, 0)
  return out
}

/** Ближайший такой день недели вперёд (сегодняшний означает сегодня). */
function nextWeekday(now, iso) {
  const out = startOfDay(now)
  out.setDate(out.getDate() + (iso - isoWeekday(now) + 7) % 7)
  return out
}

const isoWeekday = (d) => d.getDay() || 7

/** День и месяц без года: прошедшая в этом году дата означает следующий год. */
function comingDate(now, month, dayOfMonth) {
  const out = startOfDay(new Date(now.getFullYear(), month, dayOfMonth))
  if (out < startOfDay(now)) out.setFullYear(out.getFullYear() + 1)
  return out
}

function applyDaypart(h, part) {
  if (h > 23) return null
  if (part === 'дня' || part === 'вечера') return h < 12 ? h + 12 : h
  if (part === 'ночи' || part === 'утра') return h % 12
  return h
}

function combine(now, { minutes, day, time }) {
  if (minutes != null) return new Date(now.getTime() + minutes * 60_000)
  if (!day && !time) return null
  const at = day ? new Date(day) : new Date(now)
  if (time) {
    at.setHours(time.h, time.m, 0, 0)
    // Время без дня, которое уже прошло, — про завтра («напомни в 8 утра»).
    if (!day && at <= now) at.setDate(at.getDate() + 1)
    return at
  }
  at.setHours(9, 0, 0, 0)
  // День без времени, чьё утро уже позади, не должен срабатывать сразу.
  if (at <= now) return roundUp(new Date(now.getTime() + 3_600_000))
  return at
}

/** Округление вверх до 5 минут — как в быстрых сроках диалога напоминания. */
function roundUp(d) {
  d.setMinutes(Math.ceil(d.getMinutes() / 5) * 5, 0, 0)
  return d
}
