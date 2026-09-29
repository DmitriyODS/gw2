/**
 * Данные живых плиток меню «Пуск».
 *
 * Всё, что уже есть в сторах (непрочитанные чаты, портал, питомец, активный
 * юнит), плитки читают напрямую — сюда попадает только то, чего в памяти нет.
 * Сводки лёгкие (счётчики и по паре строк), тянутся ПАРАЛЛЕЛЬНО и лишь для
 * разделов, доступных пользователю, а результат живёт TTL секунд — открывать
 * «Пуск» подряд можно без единого запроса.
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getTasks } from '@/api/tasks.js'
import { getNotesSummary } from '@/api/notes.js'
import { getBoardsSummary } from '@/api/boards.js'
import { getDriveSummary } from '@/api/drive.js'
import { getUpcoming as getUpcomingReminders } from '@/api/reminders.js'
import { getRegistriesSummary } from '@/api/registries.js'
import { getFormsSummary } from '@/api/forms.js'
import { getAgenda as getDiaryAgenda } from '@/api/diaries.js'
import { getAgenda as getScheduleAgenda } from '@/api/schedules.js'
import { getAgenda as getCalendarAgenda } from '@/api/calendars.js'
import { getPosts } from '@/api/portal.js'
import { getStatsSummary } from '@/api/stats.js'
import { listMyCompanies, listCompanies } from '@/api/companies.js'
import { getDirectorySummary, getUsersSummary } from '@/api/users.js'
import { useAuthStore } from '@/stores/auth.js'

const TTL = 60_000

// Границы периодов считаем в зоне пользователя: сервер её не знает.
function ymd(d) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function startOfWeek(d) {
  const x = new Date(d)
  x.setHours(0, 0, 0, 0)
  // Понедельник — начало недели (getDay: воскресенье = 0).
  x.setDate(x.getDate() - ((x.getDay() + 6) % 7))
  return x
}

// Источники сводок: id раздела → загрузчик. Раздела нет в списке доступных —
// загрузчик не вызывается вовсе.
const SOURCES = {
  tasks: async () => {
    const userId = useAuthStore().userId
    const data = await getTasks({
      tab: 'active', responsible_id: userId, sort: 'deadline', page: 1, per_page: 3,
    })
    return { total: data.total ?? 0, items: data.tasks ?? data.items ?? [] }
  },

  diaries: async () => {
    const today = ymd(new Date())
    return getDiaryAgenda(today, today, 3)
  },

  schedule: async () => {
    /* День и текущую минуту считает КЛИЕНТ: у расписаний свои зоны, а «сейчас»
       у человека одно — сервер его не додумывает. */
    const now = new Date()
    return getScheduleAgenda(ymd(now), now.getHours() * 60 + now.getMinutes())
  },

  calendars: async () => {
    /* Календарям нужен момент времени в формате RFC3339, а не дата: границы
       периода считаются в зоне клиента (сервер её не знает). С голыми датами
       запрос отвечал 400, и плитка молча оставалась пустой. */
    const from = new Date()
    const to = new Date(from)
    to.setDate(to.getDate() + 1)
    return getCalendarAgenda(from.toISOString(), to.toISOString(), 3)
  },

  /* Сводки считает сервер (счётчик и последний элемент в SQL): раньше ради
     числа на плитке раз в минуту тянулся весь список раздела. */
  notes: () => getNotesSummary(),

  boards: () => getBoardsSummary(),

  drive: () => getDriveSummary(),

  reminders: async () => {
    const data = await getUpcomingReminders(3)
    const items = data.items ?? []
    return { total: items.length, items }
  },

  forms: () => getFormsSummary(),

  registries: () => getRegistriesSummary(),

  portal: async () => {
    const data = await getPosts({ limit: 1 })
    const latest = [...(data.pinned ?? []), ...(data.posts ?? [])][0] || null
    return { latest }
  },

  stats: async () => {
    const data = await getStatsSummary(ymd(startOfWeek(new Date())), ymd(new Date()))
    return {
      weekHours: data?.week_hours ?? 0,
      weekTasks: data?.week_tasks ?? 0,
      todayHours: data?.today_hours ?? 0,
    }
  },

  companies: async () => {
    const auth = useAuthStore()
    const items = (await (auth.isSuperAdmin ? listCompanies() : listMyCompanies())) ?? []
    return { total: items.length, active: items.filter((c) => c.is_active !== false).length }
  },

  users: () => getUsersSummary(),

  /* Сотрудники активной компании: число и id — «сколько в сети» плитка
     считает сама, пересекая их с presence (шлюз отдаёт онлайн только круга
     пользователя). Супер-админу — активные пользователи платформы. */
  employees: async () => {
    const auth = useAuthStore()
    if (!auth.isSuperAdmin) return getDirectorySummary()
    const { active } = await getUsersSummary()
    return { total: active, ids: [], allOnline: true }
  },
}

export const useLiveTilesStore = defineStore('liveTiles', () => {
  const data = ref({})
  const loading = ref(false)
  let fetchedAt = 0
  let fetchedKey = ''
  let inflight = null

  /**
   * Подтянуть сводки для перечисленных разделов.
   * Свежие данные (моложе TTL и по тому же набору разделов) не перезапрашиваем;
   * force — обход кэша (периодическое обновление, смена компании).
   * Рабочий стол зовёт это заранее и по таймеру, поэтому к открытию меню
   * «Пуск» плитки уже живые — ждать загрузки пользователю не приходится.
   */
  function refresh(appIds = [], { force = false } = {}) {
    const ids = appIds.filter((id) => SOURCES[id])
    const key = ids.join(',')
    if (!force && key === fetchedKey && Date.now() - fetchedAt < TTL) return Promise.resolve()
    // Запрос уже в пути — второй вызов ждёт его, а не шлёт свою пачку.
    if (inflight) return inflight

    loading.value = true
    inflight = Promise.allSettled(ids.map((id) => SOURCES[id]())).then((results) => {
      const next = { ...data.value }
      results.forEach((r, i) => {
        // Упавший источник не гасит остальные плитки: у раздела просто не
        // появится живая грань (или останется прошлая).
        if (r.status === 'fulfilled') next[ids[i]] = r.value
      })
      data.value = next
      fetchedAt = Date.now()
      fetchedKey = key
    }).finally(() => {
      loading.value = false
      inflight = null
    })
    return inflight
  }

  function reset() {
    data.value = {}
    fetchedAt = 0
    fetchedKey = ''
  }

  return { data, loading, refresh, reset }
})
