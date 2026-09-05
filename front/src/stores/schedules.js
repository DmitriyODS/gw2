import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@/api/schedules.js'
import { logActivity } from '@/utils/activityLog.js'
import { addDays, dateKey, mondayOf, weekIndex, weekLabel } from '@/utils/scheduleCycle.js'

/* Раздел «Расписание».

   Расписание приезжает ЦЕЛИКОМ (настройки, категории, поля и все занятия) —
   их десятки, и лишний запрос на каждую неделю дороже, чем весь набор разом.
   По неделям фильтрует клиент: правило повтора у него зеркальное
   (utils/scheduleCycle.js ≡ domain/cycle.go).

   Порог окна — личная настройка УСТРОЙСТВА поверх умолчания расписания: гость
   по ссылке чужой gap_min тронуть не может, а подкрутить шкалу под себя должен. */

const GAP_STORAGE_KEY = 'gw_schedule_gap'

function readGaps() {
  try {
    return JSON.parse(localStorage.getItem(GAP_STORAGE_KEY) || '{}') || {}
  } catch {
    return {}
  }
}

function writeGaps(map) {
  try {
    localStorage.setItem(GAP_STORAGE_KEY, JSON.stringify(map))
  } catch { /* приватный режим — переживём без запоминания */ }
}

export const useSchedulesStore = defineStore('schedules', () => {
  const tab = ref('mine')             // 'mine' | 'shared'
  const schedules = ref([])
  const loadingList = ref(false)
  const selectedId = ref(null)

  const items = ref([])
  const canEdit = ref(false)
  const loadingItems = ref(false)

  // Понедельник показываемой недели и выбранный день (телефон и вид «день»).
  const monday = ref(mondayOf(new Date()))
  const selectedWeekday = ref((new Date().getDay() + 6) % 7)
  // Режим конструктора: клик по пустому месту шкалы заводит занятие.
  const editing = ref(false)
  const gapOverrides = ref(readGaps())

  const selected = computed(() => schedules.value.find((s) => s.id === selectedId.value) || null)
  const readonly = computed(() => !canEdit.value)

  // Порог окна: личная подкрутка сильнее умолчания расписания.
  const gapMin = computed(() => {
    const own = gapOverrides.value[selectedId.value]
    return own > 0 ? own : (selected.value?.gap_min || 20)
  })

  const cycleWeek = computed(() => {
    if (!selected.value) return 1
    return weekIndex(selected.value.cycle_anchor, monday.value, selected.value.cycle_weeks)
  })

  const cycleWeekLabel = computed(() =>
    selected.value ? weekLabel(selected.value, cycleWeek.value) : '')

  const categoryById = computed(() => {
    const map = {}
    for (const c of selected.value?.categories || []) map[c.id] = c
    return map
  })

  function setGap(value) {
    if (!selectedId.value) return
    const next = Math.min(240, Math.max(5, Math.round(value)))
    gapOverrides.value = { ...gapOverrides.value, [selectedId.value]: next }
    writeGaps(gapOverrides.value)
  }

  async function fetchSchedules({ silent = false } = {}) {
    if (!silent) loadingList.value = true
    try {
      const data = await api.getSchedules(tab.value)
      schedules.value = data.schedules ?? []
      if (selectedId.value && !schedules.value.some((s) => s.id === selectedId.value)) {
        selectedId.value = null
        items.value = []
      }
    } finally {
      if (!silent) loadingList.value = false
    }
  }

  async function setTab(value) {
    if (tab.value === value) return
    tab.value = value
    selectedId.value = null
    items.value = []
    await fetchSchedules()
  }

  // applyView — разложить ответ сервера: расписание, занятия и право правки.
  function applyView(data) {
    const schedule = data.schedule
    if (!schedule) return null
    const idx = schedules.value.findIndex((s) => s.id === schedule.id)
    if (idx >= 0) schedules.value[idx] = schedule
    else schedules.value.push(schedule)
    if (selectedId.value === schedule.id || !selectedId.value) {
      selectedId.value = schedule.id
      items.value = data.items ?? []
      canEdit.value = data.can_edit !== false
    }
    return schedule
  }

  async function select(id, { force = false } = {}) {
    if (!force && selectedId.value === id && items.value.length) return
    selectedId.value = id
    loadingItems.value = true
    try {
      applyView(await api.getSchedule(id))
    } finally {
      loadingItems.value = false
    }
  }

  async function reload() {
    if (selectedId.value) await select(selectedId.value, { force: true })
  }

  // ── Навигация по неделям ──
  function stepWeek(delta) { monday.value = addDays(monday.value, delta * 7) }
  function today() {
    monday.value = mondayOf(new Date())
    selectedWeekday.value = (new Date().getDay() + 6) % 7
  }
  /** Дата дня недели показываемой недели. */
  function dateOf(weekday) { return addDays(monday.value, weekday) }

  // ── Расписания ──
  async function createSchedule(body) {
    const data = await api.createSchedule(body)
    const schedule = data.schedule
    schedules.value.push(schedule)
    selectedId.value = schedule.id
    items.value = data.items ?? []
    canEdit.value = true
    logActivity({ type: 'schedule', id: schedule.id, title: schedule.name, path: `/schedule?id=${schedule.id}` })
    return schedule
  }

  async function updateSchedule(id, body) {
    const data = await api.updateSchedule(id, body)
    applyView(data)
    return data
  }

  async function removeSchedule(id) {
    await api.deleteSchedule(id)
    schedules.value = schedules.value.filter((s) => s.id !== id)
    if (selectedId.value === id) {
      selectedId.value = null
      items.value = []
    }
  }

  // ── Занятия ──
  // Оптимизма здесь нет намеренно: занятие возвращается сервером уже
  // нормализованным (недели подрезаны под цикл, чужая категория отброшена), и
  // показывать «своё» до ответа значило бы показывать неправду.
  function upsertItem(item) {
    const idx = items.value.findIndex((i) => i.id === item.id)
    if (idx >= 0) items.value[idx] = item
    else items.value.push(item)
  }

  async function createItem(body) {
    const item = await api.createItem(selectedId.value, body)
    upsertItem(item)
    return item
  }

  async function updateItem(itemId, body) {
    const item = await api.updateItem(selectedId.value, itemId, body)
    upsertItem(item)
    return item
  }

  async function removeItem(itemId) {
    await api.deleteItem(selectedId.value, itemId)
    items.value = items.value.filter((i) => i.id !== itemId)
  }

  // ── Категории и поля ──
  async function createCategory(body) {
    const category = await api.createCategory(selectedId.value, body)
    if (selected.value) selected.value.categories = [...(selected.value.categories || []), category]
    return category
  }

  async function updateCategory(categoryId, body) {
    const category = await api.updateCategory(selectedId.value, categoryId, body)
    if (selected.value) {
      selected.value.categories = (selected.value.categories || [])
        .map((c) => (c.id === categoryId ? category : c))
    }
    return category
  }

  async function removeCategory(categoryId) {
    await api.deleteCategory(selectedId.value, categoryId)
    if (selected.value) {
      selected.value.categories = (selected.value.categories || []).filter((c) => c.id !== categoryId)
    }
    // Занятия категорию теряют, но остаются — перечитываем их снимок.
    items.value = items.value.map((i) => (i.category_id === categoryId ? { ...i, category_id: null } : i))
  }

  async function saveFields(fields) {
    applyView(await api.replaceFields(selectedId.value, fields))
  }

  // ── Сокет-события (адресуются аудитории поимённо) ──
  function applyScheduleSocket(kind, payload) {
    if (!payload) return
    const id = payload.id
    if (kind === 'deleted' || kind === 'unshared') {
      schedules.value = schedules.value.filter((s) => s.id !== id)
      if (selectedId.value === id) {
        selectedId.value = null
        items.value = []
      }
      return
    }
    const idx = schedules.value.findIndex((s) => s.id === id)
    if (idx >= 0) {
      schedules.value[idx] = { ...schedules.value[idx], ...payload }
      if (selectedId.value === id) reload()
      return
    }
    // Расписание появилось у нас впервые (им поделились) — список перечитываем:
    // снимок события структуры не несёт.
    fetchSchedules({ silent: true })
  }

  function applyItemSocket(kind, payload) {
    if (!payload || payload.schedule_id !== selectedId.value) return
    if (kind === 'deleted') {
      items.value = items.value.filter((i) => i.id !== payload.id)
      return
    }
    if (payload.item) upsertItem(payload.item)
  }

  function reset() {
    schedules.value = []
    items.value = []
    selectedId.value = null
    canEdit.value = false
    editing.value = false
  }

  return {
    tab, schedules, loadingList, selectedId, items, loadingItems, canEdit, editing,
    monday, selectedWeekday,
    selected, readonly, gapMin, cycleWeek, cycleWeekLabel, categoryById,
    setTab, fetchSchedules, select, reload, setGap, applyView,
    stepWeek, today, dateOf, dateKey,
    createSchedule, updateSchedule, removeSchedule,
    createItem, updateItem, removeItem,
    createCategory, updateCategory, removeCategory, saveFields,
    applyScheduleSocket, applyItemSocket, reset,
  }
})
