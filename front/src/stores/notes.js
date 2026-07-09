import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@/api/notes.js'
import { useAuthStore } from '@/stores/auth.js'

// Заметки: плитки-стикеры владельца + группы-фильтры + вкладка «Поделились»
// (адресный шаринг). Скоуп по владельцу на сервере; события приходят в
// комнаты владельца и адресатов — чужие (шаренные) плитки отфильтровываются
// от владельческих по owner_id.
export const useNotesStore = defineStore('notes', () => {
  const notes = ref([])          // плитки текущей выборки (группа+поиск)
  const groups = ref([])
  const loading = ref(false)
  const loadingGroups = ref(false)
  const activeGroupId = ref(0)   // 0 — «Все»
  const showArchived = ref(false) // true — фильтр «Архив» (вместо групп)
  const showShared = ref(false)  // true — вкладка «Поделились» (чужие заметки)
  const search = ref('')

  let fetchSeq = 0
  let fetchCtrl = null

  const activeGroup = computed(() => groups.value.find((g) => g.id === activeGroupId.value) || null)
  const totalCount = computed(() =>
    activeGroupId.value === 0 && !showArchived.value && !showShared.value && !search.value
      ? notes.value.length : null)

  async function fetchGroups({ silent = false } = {}) {
    if (!silent) loadingGroups.value = true
    try {
      const data = await api.getGroups()
      groups.value = data.groups ?? []
      if (activeGroupId.value !== 0 && !groups.value.some((g) => g.id === activeGroupId.value)) {
        selectGroup(0)
      }
    } finally {
      if (!silent) loadingGroups.value = false
    }
  }

  async function fetchNotes({ silent = false } = {}) {
    const seq = ++fetchSeq
    fetchCtrl?.abort()
    fetchCtrl = new AbortController()
    if (!silent) loading.value = true
    try {
      const data = await api.getNotes(
        showShared.value
          ? { shared: '1', search: search.value }
          : {
              group_id: activeGroupId.value || '',
              search: search.value,
              archived: showArchived.value ? '1' : '',
            },
        { signal: fetchCtrl.signal },
      )
      if (seq !== fetchSeq) return
      notes.value = data.notes ?? []
    } catch (e) {
      if (e?.name !== 'AbortError' && e?.error !== 'ABORTED') throw e
    } finally {
      if (seq === fetchSeq) loading.value = false
    }
  }

  function selectGroup(id) {
    if (activeGroupId.value === id && !showArchived.value && !showShared.value) return
    activeGroupId.value = id
    showArchived.value = false
    showShared.value = false
    fetchNotes()
  }

  function selectArchive() {
    if (showArchived.value) return
    activeGroupId.value = 0
    showArchived.value = true
    showShared.value = false
    fetchNotes()
  }

  function selectShared() {
    if (showShared.value) return
    activeGroupId.value = 0
    showArchived.value = false
    showShared.value = true
    fetchNotes()
  }

  function setSearch(value) {
    search.value = value
    fetchNotes()
  }

  // ── Мутации ──
  async function createNote(title = '') {
    const n = await api.createNote(title)
    upsertNote(n)
    return n
  }

  async function importNote(file) {
    const n = await api.importNote(file)
    upsertNote(n)
    return n
  }

  async function removeNote(id) {
    await api.deleteNote(id)
    dropNote(id)
    fetchGroups({ silent: true })
  }

  // Архивирование/возврат — оптимистично: плитка сразу уходит из текущей
  // выборки (архив и основной список не пересекаются), при ошибке вернётся.
  async function setArchived(id, archived) {
    const prev = notes.value.find((n) => n.id === id)
    dropNote(id)
    try {
      await api.updateNote(id, { archived })
    } catch (e) {
      if (prev) upsertNote(prev)
      throw e
    }
  }

  // Закрепление: закреплённые всегда наверху списка (сортирует и сервер).
  async function setPinned(id, pinned) {
    const n = await api.updateNote(id, { pinned })
    upsertNote(n)
    sortTiles()
    return n
  }

  async function createGroup(name) {
    const g = await api.createGroup(name)
    upsertGroup(g)
    return g
  }

  async function renameGroup(id, name) {
    const g = await api.renameGroup(id, name)
    upsertGroup(g)
    return g
  }

  async function removeGroup(id) {
    await api.deleteGroup(id)
    groups.value = groups.value.filter((g) => g.id !== id)
    if (activeGroupId.value === id) selectGroup(0)
  }

  // ── Идемпотентные апдейты списка ──
  function upsertNote(payload) {
    const i = notes.value.findIndex((n) => n.id === payload.id)
    if (i === -1) notes.value.unshift(payload)
    else notes.value[i] = { ...notes.value[i], ...payload }
  }

  function dropNote(id) {
    notes.value = notes.value.filter((n) => n.id !== id)
  }

  function upsertGroup(payload) {
    const i = groups.value.findIndex((g) => g.id === payload.id)
    if (i === -1) groups.value.push(payload)
    else groups.value[i] = { ...groups.value[i], ...payload }
  }

  // Закреплённые первыми (как на сервере), затем updated_at DESC.
  function sortTiles() {
    notes.value = [...notes.value].sort((a, b) =>
      String(b.pinned_at || '').localeCompare(String(a.pinned_at || ''))
      || String(b.updated_at).localeCompare(String(a.updated_at)))
  }

  // ── Сокет-события (комнаты владельца и адресатов) ──
  function applyNoteSocket(kind, payload) {
    if (kind === 'deleted') {
      dropNote(payload.id)
      fetchGroups({ silent: true })
      return
    }
    // Вкладка «Поделились»: обновляем только уже видимые чужие плитки —
    // появление новых ведёт note_member:added, а свои заметки сюда не входят.
    if (showShared.value) {
      if (notes.value.some((n) => n.id === payload.id)) {
        upsertNote(payload)
        sortTiles()
      }
      return
    }
    // Владельческие вкладки: чужая (шаренная со мной) заметка сюда не попадает.
    const myId = useAuthStore().userId
    if (payload.owner_id && myId && payload.owner_id !== myId) return
    // created / updated: событие несёт плитку (без doc). Выборка с фильтром
    // группы/поиска на клиенте не повторяется — плитку не из текущей группы
    // (или не из текущей архивности) просто убираем из списка.
    const inGroup = activeGroupId.value === 0
      || (payload.group_ids || []).includes(activeGroupId.value)
    if (!!payload.archived !== showArchived.value || !inGroup) {
      dropNote(payload.id)
    } else if (!search.value) {
      upsertNote(payload)
      sortTiles()
    } else {
      // Активен серверный поиск — совпадение может измениться, перечитываем.
      fetchNotes({ silent: true })
    }
    fetchGroups({ silent: true })
  }

  // Адресный шаринг: заметка появилась/пропала во вкладке «Поделились».
  function applyMemberSocket(kind, payload) {
    if (kind === 'removed') {
      dropNote(payload.note_id)
      return
    }
    if (!showShared.value) return
    const tile = { ...payload.note, my_access: payload.can_edit ? 'edit' : 'view' }
    upsertNote(tile)
    sortTiles()
  }

  function applyGroupSocket(kind, payload) {
    if (kind === 'deleted') {
      groups.value = groups.value.filter((g) => g.id !== payload.id)
      if (activeGroupId.value === payload.id) selectGroup(0)
      return
    }
    upsertGroup(payload)
  }

  return {
    notes, groups, loading, loadingGroups, activeGroupId, activeGroup,
    showArchived, showShared, search, totalCount,
    fetchGroups, fetchNotes, selectGroup, selectArchive, selectShared, setSearch,
    createNote, importNote, removeNote, setArchived, setPinned,
    createGroup, renameGroup, removeGroup,
    upsertNote, applyNoteSocket, applyGroupSocket, applyMemberSocket,
  }
})
