import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@/api/diaries.js'
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

export const useDiariesStore = defineStore('diaries', () => {
  const tab = ref('mine')            // 'mine' | 'shared'
  const diaries = ref([])            // ежедневники активной вкладки
  const loadingList = ref(false)
  const selectedId = ref(null)

  const subtab = ref('active')       // 'active' | 'archive'
  const entries = ref([])            // активные записи текущего диапазона
  const archive = ref([])            // выполненные (вкладка «Архив»)
  const dayDone = ref([])            // выполненные за день (вид «День» делится на активные/архив)
  const loadingEntries = ref(false)

  const view = ref('week')           // по умолчанию — неделя
  const cursor = ref(startOfDay(new Date()))
  const search = ref('')

  let fetchSeq = 0
  let fetchCtrl = null

  const selected = computed(() => diaries.value.find((d) => d.id === selectedId.value) || null)
  // Read-only: чужой ежедневник (вкладка «Поделились»).
  const readonly = computed(() => tab.value === 'shared' || selected.value?.shared === true)

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
    const from = startOfWeek(startOfMonth(base))
    return { from, to: addDays(from, 42) }
  })

  const entriesByDay = computed(() => {
    const map = {}
    for (const e of entries.value) (map[e.entry_date] ||= []).push(e)
    return map
  })

  function myId() { return useAuthStore().userId ?? useAuthStore().user?.id ?? null }

  async function fetchDiaries() {
    loadingList.value = true
    try {
      const data = await api.getDiaries(tab.value)
      diaries.value = data.diaries ?? []
      if (selectedId.value && !diaries.value.some((d) => d.id === selectedId.value)) {
        selectedId.value = null
      }
    } finally {
      loadingList.value = false
    }
  }

  function setTab(v) {
    if (tab.value === v) return
    tab.value = v
    selectedId.value = null
    entries.value = []
    archive.value = []
    fetchDiaries()
  }

  function select(id) {
    if (selectedId.value === id) return
    selectedId.value = id
    search.value = ''
    subtab.value = 'active'
    entries.value = []
    archive.value = []
    if (id != null) fetchEntries()
  }

  function setSubtab(v) {
    if (subtab.value === v) return
    subtab.value = v
    fetchEntries()
  }

  function setView(v) {
    if (view.value === v) return
    view.value = v
    if (subtab.value === 'active') fetchEntries()
  }

  function setCursor(date) { cursor.value = startOfDay(date); fetchEntries() }

  function step(dir) {
    const base = cursor.value
    if (view.value === 'day') cursor.value = addDays(base, dir)
    else if (view.value === 'week') cursor.value = addDays(base, dir * 7)
    else { const x = startOfMonth(base); x.setMonth(x.getMonth() + dir); cursor.value = x }
    fetchEntries()
  }

  function today() { setCursor(new Date()) }

  function setSearch(value) { search.value = value; fetchEntries() }

  async function fetchEntries({ silent = false } = {}) {
    if (selectedId.value == null) return
    const seq = ++fetchSeq
    fetchCtrl?.abort()
    fetchCtrl = new AbortController()
    if (!silent) loadingEntries.value = true
    try {
      const id = selectedId.value
      let data
      if (subtab.value === 'archive') {
        data = await api.getEntries(id, { archived: 1, search: search.value }, { signal: fetchCtrl.signal })
        if (seq !== fetchSeq) return
        archive.value = data.items ?? []
      } else {
        const { from, to } = range.value
        data = await api.getEntries(id, { from: dayKey(from), to: dayKey(to), search: search.value }, { signal: fetchCtrl.signal })
        if (seq !== fetchSeq) return
        entries.value = data.items ?? []
        // В виде «День» дополнительно тянем выполненные за этот день — день
        // делится на активные и архив прямо в списке.
        if (view.value === 'day') {
          const done = await api.getEntries(id, { archived: 1, from: dayKey(from), to: dayKey(to), search: search.value }, { signal: fetchCtrl.signal })
          if (seq !== fetchSeq) return
          dayDone.value = done.items ?? []
        } else {
          dayDone.value = []
        }
      }
    } catch (e) {
      if (e?.name !== 'AbortError' && e?.error !== 'ABORTED') throw e
    } finally {
      if (seq === fetchSeq) loadingEntries.value = false
    }
  }

  // ── Ежедневники (мутации) ──
  async function createDiary(name) {
    const d = await api.createDiary(name)
    if (tab.value === 'mine') diaries.value.push(d)
    return d
  }
  async function renameDiary(id, name) {
    const d = await api.updateDiary(id, name)
    const i = diaries.value.findIndex((x) => x.id === id)
    if (i !== -1) diaries.value[i] = { ...diaries.value[i], ...d }
    return d
  }
  async function removeDiary(id) {
    await api.deleteDiary(id)
    diaries.value = diaries.value.filter((d) => d.id !== id)
    if (selectedId.value === id) select(null)
  }

  // ── Записи (мутации) ──
  async function createEntry(body) {
    const e = await api.createEntry(selectedId.value, body)
    if (subtab.value === 'active') await fetchEntries({ silent: true })
    return e
  }
  async function updateEntry(entryId, body) {
    const e = await api.updateEntry(selectedId.value, entryId, body)
    await fetchEntries({ silent: true })
    return e
  }
  async function toggleDone(entryId, done) {
    await api.setEntryDone(selectedId.value, entryId, done)
    await fetchEntries({ silent: true })
  }
  async function linkTask(entryId, taskId) {
    const e = await api.linkEntryTask(selectedId.value, entryId, taskId)
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

  // ── Сокет-события (адресованы владельцу и адресатам) ──
  function applyDiarySocket(kind, payload) {
    if (kind === 'deleted' || kind === 'unshared') {
      diaries.value = diaries.value.filter((d) => d.id !== payload.id)
      if (selectedId.value === payload.id) select(null)
      return
    }
    if (kind === 'shared') {
      // Чужой ежедневник открыли мне — он в «Поделились».
      if (tab.value === 'shared') upsertDiary({ ...payload, shared: true })
      return
    }
    // created / updated
    const mine = payload.owner_id === myId()
    const belongs = (mine && tab.value === 'mine') || (!mine && tab.value === 'shared')
    if (!belongs) return
    upsertDiary(payload)
  }

  function upsertDiary(payload) {
    const i = diaries.value.findIndex((d) => d.id === payload.id)
    const d = { id: payload.id, owner_id: payload.owner_id, name: payload.name, position: payload.position,
      shared: !!payload.shared, owner_name: payload.owner_name, owner_avatar: payload.owner_avatar }
    if (i === -1) diaries.value.push(d)
    else diaries.value[i] = { ...diaries.value[i], ...d }
  }

  function applyEntrySocket(payload) {
    if (payload?.diary_id !== selectedId.value) return
    fetchEntries({ silent: true })
  }

  return {
    tab, diaries, loadingList, selectedId, selected, readonly,
    subtab, entries, archive, dayDone, loadingEntries, entriesByDay,
    view, cursor, search, range,
    fetchDiaries, setTab, select, setSubtab, setView, setCursor, step, today, setSearch, fetchEntries,
    createDiary, renameDiary, removeDiary,
    createEntry, updateEntry, toggleDone, linkTask, deleteEntry, bulkDelete,
    applyDiarySocket, applyEntrySocket,
  }
})
