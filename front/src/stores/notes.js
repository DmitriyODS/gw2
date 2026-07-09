import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@/api/notes.js'

// Заметки: плитки-стикеры владельца + группы-фильтры. Все данные личные
// (скоуп по владельцу на сервере), сокет-события приходят только в комнату
// владельца — фильтровать по компании не нужно.
export const useNotesStore = defineStore('notes', () => {
  const notes = ref([])          // плитки текущей выборки (группа+поиск)
  const groups = ref([])
  const loading = ref(false)
  const loadingGroups = ref(false)
  const activeGroupId = ref(0)   // 0 — «Все»
  const showArchived = ref(false) // true — фильтр «Архив» (вместо групп)
  const search = ref('')

  let fetchSeq = 0
  let fetchCtrl = null

  const activeGroup = computed(() => groups.value.find((g) => g.id === activeGroupId.value) || null)
  const totalCount = computed(() =>
    activeGroupId.value === 0 && !showArchived.value && !search.value ? notes.value.length : null)

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
        {
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
    if (activeGroupId.value === id && !showArchived.value) return
    activeGroupId.value = id
    showArchived.value = false
    fetchNotes()
  }

  function selectArchive() {
    if (showArchived.value) return
    activeGroupId.value = 0
    showArchived.value = true
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

  // ── Сокет-события (только комната владельца) ──
  function applyNoteSocket(kind, payload) {
    if (kind === 'deleted') {
      dropNote(payload.id)
      fetchGroups({ silent: true })
      return
    }
    // created / updated: событие несёт плитку (без doc). Выборка с фильтром
    // группы/поиска на клиенте не повторяется — плитку не из текущей группы
    // (или не из текущей архивности) просто убираем из списка.
    const inGroup = activeGroupId.value === 0
      || (payload.group_ids || []).includes(activeGroupId.value)
    if (!!payload.archived !== showArchived.value || !inGroup) {
      dropNote(payload.id)
    } else if (!search.value) {
      upsertNote(payload)
      // Сортировка updated_at DESC — обновлённая заметка всплывает наверх.
      notes.value = [...notes.value].sort((a, b) =>
        String(b.updated_at).localeCompare(String(a.updated_at)))
    } else {
      // Активен серверный поиск — совпадение может измениться, перечитываем.
      fetchNotes({ silent: true })
    }
    fetchGroups({ silent: true })
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
    notes, groups, loading, loadingGroups, activeGroupId, activeGroup, showArchived, search, totalCount,
    fetchGroups, fetchNotes, selectGroup, selectArchive, setSearch,
    createNote, importNote, removeNote, setArchived,
    createGroup, renameGroup, removeGroup,
    upsertNote, applyNoteSocket, applyGroupSocket,
  }
})
