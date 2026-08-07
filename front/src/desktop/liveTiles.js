/**
 * Живые плитки меню «Пуск» (в духе Metro): каждая плитка по очереди
 * показывает «грани» — короткие полезные сводки своего раздела.
 *
 * Здесь только чистое построение граней из уже собранных данных: сводки
 * `stores/liveTiles.js` (то, чего нет в памяти) и обычные сторы (непрочитанные,
 * питомец, активный юнит). Ни запросов, ни таймеров — их ведут стор и компонент.
 *
 * Грань: { key, value, label, tone? }
 *   value — крупная строка (число или короткая фраза), label — подпись,
 *   tone: 'alert' — тревожная (болезнь питомца, просроченный дедлайн).
 */

/** Грани раздела. ctx: { data, messenger, portal, pets, units, auth, companies }. */
export function tileFaces(appId, ctx) {
  const build = FACES[appId]
  if (!build) return []
  try {
    return build(ctx).filter(Boolean).slice(0, 4)
  } catch {
    // Плитка — украшение: любой сюрприз в данных гасит грани, но не меню.
    return []
  }
}

const FACES = {
  tasks: ({ data, units }) => {
    const out = []
    const summary = data.tasks
    if (summary?.total) out.push(face('count', summary.total, plural(summary.total, 'задача в работе', 'задачи в работе', 'задач в работе')))

    const next = (summary?.items || []).find((t) => t.deadline)
    if (next) {
      const overdue = new Date(next.deadline) < startOfToday()
      out.push(face('deadline', overdue ? 'Просрочено' : dayLabel(next.deadline), next.name, overdue ? 'alert' : null))
    }

    const unit = units?.activeUnit
    if (unit) out.push(face('unit', 'Идёт работа', unit.task_name || unit.name || 'Активный юнит'))
    return out
  },

  messenger: ({ data, messenger }) => {
    const out = []
    const unread = messenger?.totalUnread || 0
    if (unread) out.push(face('unread', unread, plural(unread, 'новое сообщение', 'новых сообщения', 'новых сообщений')))

    const conv = (messenger?.conversations || []).find((c) => c.unread_count)
      || (messenger?.conversations || [])[0]
    const text = conv?.last_message?.text?.trim()
    if (conv && text) out.push(face('last', convName(conv), text))

    const online = messenger?.onlineIds?.size || 0
    if (online) out.push(face('online', online, plural(online, 'коллега в сети', 'коллеги в сети', 'коллег в сети')))
    return out
  },

  portal: ({ data, portal }) => {
    const out = []
    const unread = portal?.unread || 0
    if (unread) out.push(face('unread', unread, plural(unread, 'новая публикация', 'новые публикации', 'новых публикаций')))

    const latest = data.portal?.latest
    if (latest) out.push(face('latest', 'Свежее', latest.title || firstLine(latest.body) || 'Публикация'))
    return out
  },

  pets: ({ pets }) => {
    const pet = pets?.pet
    if (!pet) return []
    const out = []

    if (pet.sick) {
      out.push(face('sick', 'Заболел', pet.runaway_in_days
        ? `Сбежит через ${pet.runaway_in_days} дн. — нужно лечение`
        : 'Нужно лечение', 'alert'))
    } else if (pet.adventure_until) {
      out.push(face('away', 'В походе', 'Вернётся с наградой'))
    }

    out.push(face('mood', pet.name || 'Грувик', pet.mood_title ? `Настроение: ${pet.mood_title.toLowerCase()}` : 'Ваш питомец'))
    if (pet.kudos != null) out.push(face('kudos', pet.kudos, plural(pet.kudos, 'кудос', 'кудоса', 'кудосов')))
    return out
  },

  notes: ({ data }) => {
    const n = data.notes
    if (!n) return []
    const out = [face('count', n.total, plural(n.total, 'заметка', 'заметки', 'заметок'))]
    if (n.latest) out.push(face('latest', 'Последняя', n.latest.title || 'Без названия'))
    return out
  },

  diaries: ({ data }) => {
    const a = data.diaries
    if (!a) return []
    if (!a.total) return [face('empty', 'Всё сделано', 'На сегодня дел не осталось')]

    const out = [face('count', a.total, plural(a.total, 'дело на сегодня', 'дела на сегодня', 'дел на сегодня'))]
    const next = a.items?.[0]
    if (next) out.push(face('next', next.start_min != null ? `в ${hhmm(next.start_min)}` : 'Дальше', next.title))
    return out
  },

  calendars: ({ data }) => {
    const a = data.calendars
    if (!a) return []
    if (!a.total) return [face('empty', 'Свободно', 'Событий на сегодня нет')]

    const out = [face('count', a.total, plural(a.total, 'событие сегодня', 'события сегодня', 'событий сегодня'))]
    const next = a.items?.[0]
    if (next) out.push(face('next', `в ${timeOf(next.event_at)}`, next.title))
    return out
  },

  boards: ({ data }) => {
    const b = data.boards
    if (!b) return []
    const out = [face('count', b.total, plural(b.total, 'доска', 'доски', 'досок'))]
    if (b.latest) out.push(face('latest', 'Последняя', b.latest.title || 'Без названия'))
    return out
  },

  drive: ({ data }) => {
    const d = data.drive
    if (!d) return []
    if (!d.total) return [face('empty', 'Диск пуст', 'Перетащите сюда файлы')]

    const out = [face('count', d.total, plural(d.total, 'недавний файл', 'недавних файла', 'недавних файлов'))]
    if (d.latest) out.push(face('latest', 'Последний', d.latest.name || 'Без названия'))
    return out
  },

  reminders: ({ data }) => {
    const r = data.reminders
    if (!r) return []
    if (!r.total) return [face('empty', 'Тишина', 'Ближайших напоминаний нет')]

    const out = [face('count', r.total, plural(r.total, 'напоминание', 'напоминания', 'напоминаний'))]
    const next = r.items?.[0]
    if (next) out.push(face('next', `в ${timeOf(next.remind_at)}`, next.title))
    return out
  },

  registries: ({ data }) => {
    const r = data.registries
    if (!r?.total) return []
    return [
      face('count', r.total, plural(r.total, 'реестр', 'реестра', 'реестров')),
      r.names?.length ? face('names', 'Справочники', r.names.join(' · ')) : null,
    ]
  },

  stats: ({ data }) => {
    const s = data.stats
    if (!s) return []
    return [
      face('week', `${hours(s.weekHours)} ч`, 'отработано за неделю'),
      face('today', `${hours(s.todayHours)} ч`, 'сегодня'),
      s.weekTasks ? face('tasks', s.weekTasks, plural(s.weekTasks, 'задача за неделю', 'задачи за неделю', 'задач за неделю')) : null,
    ]
  },

  companies: ({ data, auth }) => {
    const c = data.companies
    const out = []
    const active = auth?.claims?.company_name
    if (active) out.push(face('active', active, 'активная компания'))
    if (c?.total) out.push(face('count', c.total, plural(c.total, 'компания', 'компании', 'компаний')))
    return out
  },

  employees: ({ data, messenger }) => {
    const e = data.employees
    if (!e?.total) return []
    const out = [face('total', e.total, plural(e.total, 'сотрудник', 'сотрудника', 'сотрудников'))]

    // Онлайн считаем по своей компании, а не по всей платформе.
    const online = (e.ids || []).filter((id) => messenger?.onlineIds?.has(id)).length
    if (online) out.push(face('online', online, plural(online, 'в сети', 'в сети', 'в сети')))
    return out
  },

  users: ({ data }) => {
    const u = data.users
    if (!u) return []
    return [
      face('total', u.total, plural(u.total, 'пользователь', 'пользователя', 'пользователей')),
      face('active', u.active, 'активных на платформе'),
    ]
  },
}

function face(key, value, label, tone = null) {
  if (value == null || value === '') return null
  return { key, value: String(value), label: label || '', tone }
}

function convName(c) {
  if (c.is_dev_chat) return 'Техподдержка'
  return (c.is_group ? c.title : c.other_user?.fio) || 'Чат'
}

function firstLine(text) {
  return String(text || '').split('\n').find((l) => l.trim()) || ''
}

function startOfToday() {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  return d
}

/** «сегодня» / «завтра» / «до 3 авг» — короткая подпись срока. */
function dayLabel(iso) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return 'Срок'
  const days = Math.round((new Date(d).setHours(0, 0, 0, 0) - startOfToday()) / 86400000)
  if (days === 0) return 'Сегодня'
  if (days === 1) return 'Завтра'
  return `до ${d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })}`
}

function timeOf(iso) {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

/** Минуты от полуночи → ЧЧ:ММ (время записи ежедневника). */
function hhmm(min) {
  return `${String(Math.floor(min / 60)).padStart(2, '0')}:${String(min % 60).padStart(2, '0')}`
}

function hours(v) {
  const n = Number(v) || 0
  return Number.isInteger(n) ? String(n) : n.toFixed(1).replace('.', ',')
}

/** Русское склонение числительных: 1 задача / 2 задачи / 5 задач. */
export function plural(n, one, few, many) {
  const abs = Math.abs(n) % 100
  const last = abs % 10
  if (abs > 10 && abs < 20) return many
  if (last === 1) return one
  if (last >= 2 && last <= 4) return few
  return many
}
