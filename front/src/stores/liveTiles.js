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
import { getNotes } from '@/api/notes.js'
import { getBoards } from '@/api/boards.js'
import { getUpcoming as getUpcomingReminders } from '@/api/reminders.js'
import { getRegistries } from '@/api/registries.js'
import { getAgenda as getDiaryAgenda } from '@/api/diaries.js'
import { getAgenda as getCalendarAgenda } from '@/api/calendars.js'
import { getPosts } from '@/api/portal.js'
import { getStatsProfile } from '@/api/stats.js'
import { listMyCompanies, listCompanies } from '@/api/companies.js'
import { getUsers } from '@/api/users.js'
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

  calendars: async () => {
    const now = new Date()
    const tomorrow = new Date(now)
    tomorrow.setDate(tomorrow.getDate() + 1)
    return getCalendarAgenda(ymd(now), ymd(tomorrow), 3)
  },

  notes: async () => {
    const data = await getNotes({})
    const notes = data.notes ?? []
    return { total: notes.length, latest: notes[0] || null }
  },

  boards: async () => {
    const data = await getBoards({})
    const boards = data.boards ?? []
    return { total: boards.length, latest: boards[0] || null }
  },

  reminders: async () => {
    const data = await getUpcomingReminders(3)
    const items = data.items ?? []
    return { total: items.length, items }
  },

  registries: async () => {
    const items = (await getRegistries()) ?? []
    return { total: items.length, names: items.slice(0, 3).map((r) => r.name) }
  },

  portal: async () => {
    const data = await getPosts({ limit: 1 })
    const latest = [...(data.pinned ?? []), ...(data.posts ?? [])][0] || null
    return { latest }
  },

  stats: async () => {
    const today = ymd(new Date())
    const [week, day] = await Promise.all([
      getStatsProfile(ymd(startOfWeek(new Date())), today),
      getStatsProfile(today, today),
    ])
    return {
      weekHours: week?.total_hours ?? 0,
      weekTasks: week?.tasks_count ?? 0,
      todayHours: day?.total_hours ?? 0,
    }
  },

  companies: async () => {
    const auth = useAuthStore()
    const items = (await (auth.isSuperAdmin ? listCompanies() : listMyCompanies())) ?? []
    return { total: items.length, active: items.filter((c) => c.is_active !== false).length }
  },

  users: async () => {
    const items = (await getUsers()) ?? []
    return { total: items.length, active: items.filter((u) => u.is_active).length }
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
