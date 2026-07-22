import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@/api/calendars.js'
import { useAuthStore } from '@/stores/auth.js'

// ── Хелперы дат (неделя начинается с понедельника) ──
function startOfDay(d) { const x = new Date(d); x.setHours(0, 0, 0, 0); return x }
function addDays(d, n) { const x = new Date(d); x.setDate(x.getDate() + n); return x }
function startOfWeek(d) { const x = startOfDay(d); const wd = (x.getDay() + 6) % 7; return addDays(x, -wd) }
function startOfMonth(d) { const x = startOfDay(d); x.setDate(1); return x }

// dayKey — локальный ключ дня 'YYYY-MM-DD' (без UTC-сдвига).
export function dayKey(d) {
  const x = new Date(d)
  const pad = (n) => String(n).padStart(2, '0')
  return `${x.getFullYear()}-${pad(x.getMonth() + 1)}-${pad(x.getDate())}`
}

export const useCalendarsStore = defineStore('calendars', () => {
  const calendars = ref([])          // [{id, name, fields:[...]}]
  const loadingList = ref(false)
  const selectedId = ref(null)

  const entries = ref([])
  const loadingEntries = ref(false)

  const view = ref('month')          // 'month' | 'week' | 'day'
  const cursor = ref(startOfDay(new Date())) // опорная дата периода
  const search = ref('')

  let fetchSeq = 0
  let fetchCtrl = null

  const selected = computed(() => calendars.value.find((c) => c.id === selectedId.value) || null)

  // ── Видимый диапазон [from, to) по режиму ──
  const range = computed(() => {
    const base = cursor.value
    if (view.value === 'day') {
      const from = startOfDay(base)
      return { from, to: addDays(from, 1) }
    }
    if (view.value === 'week') {
      const from = startOfWeek(base)
      return { from, to: addDays(from, 7) }
    }
    // month — сетка 6 недель от начала недели, содержащей 1-е число.
    const from = startOfWeek(startOfMonth(base))
    return { from, to: addDays(from, 42) }
  })

  // entriesByDay — записи, сгруппированные по дню (для плиток).
  const entriesByDay = computed(() => {
    const map = {}
    for (const e of entries.value) {
      const k = dayKey(e.event_at)
      ;(map[k] ||= []).push(e)
    }
    return map
  })

  function myCompanyId() { return useAuthStore().companyId ?? null }
  function isMine(companyId) {
    const mine = myCompanyId()
    return companyId == null || mine == null || companyId === mine
  }

  function normalizeCal(c) {
    return { ...c, fields: Array.isArray(c?.fields) ? c.fields : [] }
  }

  async function fetchCalendars() {
    loadingList.value = true
    try {
      const data = await api.getCalendars()
      calendars.value = (data.calendars ?? data ?? []).map(normalizeCal)
      if (selectedId.value && !calendars.value.some((c) => c.id === selectedId.value)) {
        selectedId.value = null
      }
    } finally {
      loadingList.value = false
    }
  }

  function select(id) {
    if (selectedId.value === id) return
    selectedId.value = id
    search.value = ''
    entries.value = []
    if (id != null) fetchEntries()
  }

  function setView(v) {
    if (view.value === v) return
    view.value = v
    fetchEntries()
  }

  function setCursor(date) {
    cursor.value = startOfDay(date)
    fetchEntries()
  }

  // Шаг назад/вперёд по текущему режиму.
  function step(dir) {
    const base = cursor.value
    if (view.value === 'day') cursor.value = addDays(base, dir)
    else if (view.value === 'week') cursor.value = addDays(base, dir * 7)
    else {
      const x = startOfMonth(base)
      x.setMonth(x.getMonth() + dir)
      cursor.value = x
    }
    fetchEntries()
  }

  function today() { setCursor(new Date()) }

  function setSearch(value) {
    search.value = value
    fetchEntries()
  }

  async function fetchEntries({ silent = false } = {}) {
    if (selectedId.value == null) return
    const seq = ++fetchSeq
    fetchCtrl?.abort()
    fetchCtrl = new AbortController()
    if (!silent) loadingEntries.value = true
    try {
      const { from, to } = range.value
      const data = await api.getEntries(
        selectedId.value,
        { from: from.toISOString(), to: to.toISOString(), search: search.value },
        { signal: fetchCtrl.signal },
      )
      if (seq !== fetchSeq) return
      entries.value = data.items ?? []
    } catch (e) {
      if (e?.name !== 'AbortError') throw e
    } finally {
      if (seq === fetchSeq) loadingEntries.value = false
    }
  }

  async function createEntry(eventAt, data) {
    await api.createEntry(selectedId.value, eventAt, data)
    await fetchEntries({ silent: true })
  }

  async function updateEntry(entryId, eventAt, data) {
    const e = await api.updateEntry(selectedId.value, entryId, eventAt, data)
    await fetchEntries({ silent: true })
    return e
  }

  async function deleteEntry(entryId) {
    await api.deleteEntry(selectedId.value, entryId)
    await fetchEntries({ silent: true })
  }

  async function bulkDelete(ids) {
    await api.bulkDeleteEntries(selectedId.value, ids)
    await fetchEntries({ silent: true })
  }

  // ── Сокет-события ──
  function applyCalendarSocket(kind, payload) {
    if (!isMine(payload?.company_id)) return
    if (kind === 'deleted') {
      calendars.value = calendars.value.filter((c) => c.id !== payload.id)
      if (selectedId.value === payload.id) select(null)
      return
    }
    const i = calendars.value.findIndex((c) => c.id === payload.id)
    const cal = normalizeCal({ id: payload.id, name: payload.name, position: payload.position, fields: payload.fields })
    if (i === -1) calendars.value.push(cal)
    else calendars.value[i] = { ...calendars.value[i], ...cal }
    if (kind === 'updated' && selectedId.value === payload.id) fetchEntries({ silent: true })
  }

  function applyEntrySocket(_kind, payload) {
    if (!isMine(payload?.company_id)) return
    if (payload?.calendar_id !== selectedId.value) return
    // Чужие мутации проще отразить перечиткой текущего диапазона.
    fetchEntries({ silent: true })
  }

  // Смена активной компании: календари company-scoped — сбрасываем список,
  // выбор и записи прежней компании и грузим заново под новую.
  async function reloadForCompany() {
    selectedId.value = null
    entries.value = []
    search.value = ''
    calendars.value = []
    await fetchCalendars()
  }

  return {
    calendars, loadingList, selectedId, selected,
    entries, loadingEntries, entriesByDay,
    view, cursor, search, range,
    fetchCalendars, select, setView, setCursor, step, today, setSearch, reloadForCompany,
    fetchEntries, createEntry, updateEntry, deleteEntry, bulkDelete,
    applyCalendarSocket, applyEntrySocket,
  }
})
