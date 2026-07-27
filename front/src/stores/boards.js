import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@/api/boards.js'
import { useAuthStore } from '@/stores/auth.js'
import { logActivity } from '@/utils/activityLog.js'

/* Раздел «Доски»: иерархические папки (свои + расшаренные мне) и плитки досок. Скоуп по владельцу на сервере; эффективный доступ (шары,
   расшаренные папки-предки) считает сервер, клиент лишь отражает my_access.

   Все изменения списка идут через upsert по id: сокет-событие приходит раньше
   HTTP-ответа, поэтому слепой push дублировал бы плитку. */
export const useBoardsStore = defineStore('boards', () => {
  const folders = ref([])       // свои папки (плоско, parent_id)
  const sharedRoots = ref([])   // расшаренные мне «корни»
  const boards = ref([])        // плитки текущей выборки

  const loading = ref(false)
  const loadingFolders = ref(false)

  const activeFolderId = ref(null)  // null — корень
  const showArchived = ref(false)
  const showShared = ref(false)     // «Поделились со мной»
  const search = ref('')

  let fetchSeq = 0

  const myId = () => useAuthStore().userId

  const folderById = computed(() => {
    const m = new Map()
    for (const f of folders.value) m.set(f.id, f)
    for (const f of sharedRoots.value) m.set(f.id, f)
    return m
  })

  const folderTree = computed(() => buildTree(folders.value, null))

  function buildTree(list, parentId) {
    return list
      .filter((f) => (f.parent_id ?? null) === parentId)
      .sort((a, b) => (a.position - b.position) || a.name.localeCompare(b.name))
      .map((f) => ({ ...f, children: buildTree(list, f.id) }))
  }

  const activeFolder = computed(() =>
    (activeFolderId.value ? folderById.value.get(activeFolderId.value) : null))

  // Папка чужая — правит её содержимое только тот, кому дали can_edit.
  const isSharedContext = computed(() =>
    !!activeFolder.value && activeFolder.value.owner_id !== myId())

  const boardById = (id) => boards.value.find((b) => b.id === id) || null

  // Путь до активной папки (хлебные крошки).
  const path = computed(() => {
    const out = []
    let cur = activeFolder.value
    const guard = new Set()
    while (cur && !guard.has(cur.id)) {
      guard.add(cur.id)
      out.unshift(cur)
      cur = cur.parent_id ? folderById.value.get(cur.parent_id) : null
    }
    return out
  })

  // ── Загрузка ──
  async function fetchFolders({ silent = false } = {}) {
    if (!silent) loadingFolders.value = true
    try {
      const data = await api.getFolders()
      folders.value = data.folders ?? []
      sharedRoots.value = data.shared ?? []
    } finally {
      loadingFolders.value = false
    }
  }

  /** Плитки текущей выборки. Поиск всегда глобальный: фильтр папки снимается. */
  async function fetchBoards({ silent = false } = {}) {
    const seq = ++fetchSeq
    if (!silent) loading.value = true
    try {
      const params = {}
      if (search.value.trim()) {
        params.search = search.value.trim()
      } else if (showShared.value) {
        params.shared = '1'
      } else {
        params.folder_id = activeFolderId.value ?? 'root'
      }
      if (showArchived.value && !showShared.value) params.archived = '1'

      const data = await api.getBoards(params)
      if (seq !== fetchSeq) return // ответ устаревшего запроса
      boards.value = data.boards ?? []
    } finally {
      if (seq === fetchSeq) loading.value = false
    }
  }

  async function refresh() {
    await Promise.all([fetchFolders({ silent: true }), fetchBoards({ silent: true })])
  }

  // ── Навигация ──
  function openFolder(id) {
    activeFolderId.value = id ?? null
    showShared.value = false
    showArchived.value = false
    search.value = ''
    return fetchBoards()
  }

  function openShared() {
    showShared.value = true
    activeFolderId.value = null
    showArchived.value = false
    return fetchBoards()
  }

  function openArchive() {
    showArchived.value = true
    showShared.value = false
    activeFolderId.value = null
    return fetchBoards()
  }

  // ── Доски ──
  async function createBoard(title = 'Новая доска') {
    const created = await api.createBoard(title, activeFolderId.value)
    upsertBoard(created)
    logActivity({ kind: 'board', id: created.id, title: created.title || 'Доска' })
    return created
  }

  async function updateBoard(id, body) {
    const updated = await api.updateBoard(id, body)
    syncBoard(updated)
    return updated
  }

  async function removeBoard(id) {
    await api.deleteBoard(id)
    dropBoard(id)
  }

  async function moveBoard(id, folderId) {
    const moved = await api.moveBoard(id, folderId)
    syncBoard(moved)
    return moved
  }

  async function copyBoard(id) {
    const copy = await api.copyBoard(id)
    upsertBoard(copy)
    return copy
  }

  const togglePinned = (b) => updateBoard(b.id, { pinned: !b.pinned_at })
  const toggleArchived = (b) => updateBoard(b.id, { archived: !b.archived })
  const setColor = (b, color) => updateBoard(b.id, { color })

  // ── Папки ──
  async function createFolder(name, parentId = activeFolderId.value, color = '') {
    const created = await api.createFolder(name, parentId, color)
    upsertFolder(created)
    return created
  }

  async function updateFolder(id, body) {
    upsertFolder(await api.updateFolder(id, body))
  }

  async function moveFolder(id, parentId) {
    upsertFolder(await api.moveFolder(id, parentId))
  }

  async function removeFolder(id) {
    await api.deleteFolder(id)
    folders.value = folders.value.filter((f) => f.id !== id)
    if (activeFolderId.value === id) await openFolder(null)
    else await fetchFolders({ silent: true })
  }

  // ── Применение сокет-событий ──
  function applyBoardSocket(kind, payload) {
    if (!payload?.id) return
    if (kind === 'deleted') {
      dropBoard(payload.id)
      return
    }
    syncBoard(payload)
  }

  function applyFolderSocket(kind, payload) {
    if (!payload?.id) return
    if (kind === 'deleted') {
      folders.value = folders.value.filter((f) => f.id !== payload.id)
      sharedRoots.value = sharedRoots.value.filter((f) => f.id !== payload.id)
      return
    }
    upsertFolder(payload)
  }

  /** Появился/пропал доступ ко мне — перечитываем дерево и текущий список. */
  function applyShareSocket() {
    return refresh()
  }

  // ── Внутреннее ──
  function inCurrentScope(b) {
    if (!b) return false
    if (search.value.trim()) return true
    if (showShared.value) return b.owner_id !== myId()
    if (!!b.archived !== showArchived.value) return false
    return (b.folder_id ?? null) === (activeFolderId.value ?? null)
  }

  /* Привести список к текущей выборке: доска, ушедшая в архив (или вернувшаяся
     из него, уехавшая в другую папку), не должна оставаться на экране — иначе
     перенос виден только после перезагрузки страницы. */
  function syncBoard(b) {
    if (!b?.id) return
    if (inCurrentScope(b)) upsertBoard(b)
    else dropBoard(b.id)
  }

  function upsertBoard(b) {
    if (!b?.id) return
    const idx = boards.value.findIndex((x) => x.id === b.id)
    if (idx >= 0) boards.value[idx] = { ...boards.value[idx], ...b }
    else boards.value.push(b)
  }

  function dropBoard(id) {
    boards.value = boards.value.filter((b) => b.id !== id)
  }

  function upsertFolder(f) {
    if (!f?.id) return
    const list = f.owner_id === myId() ? folders : sharedRoots
    const idx = list.value.findIndex((x) => x.id === f.id)
    if (idx >= 0) list.value[idx] = { ...list.value[idx], ...f }
    else list.value.push(f)
  }

  function reset() {
    folders.value = []
    sharedRoots.value = []
    boards.value = []
    activeFolderId.value = null
    showArchived.value = false
    showShared.value = false
    search.value = ''
  }

  return {
    // состояние
    folders, sharedRoots, boards, loading, loadingFolders,
    activeFolderId, showArchived, showShared, search,
    // производное
    folderById, folderTree, activeFolder, isSharedContext, path, boardById,
    // загрузка и навигация
    fetchFolders, fetchBoards, refresh,
    openFolder, openShared, openArchive,
    // доски
    createBoard, updateBoard, removeBoard, moveBoard, copyBoard,
    togglePinned, toggleArchived, setColor,
    // папки
    createFolder, updateFolder, moveFolder, removeFolder,
    // сокеты
    applyBoardSocket, applyFolderSocket, applyShareSocket, syncBoard,
    reset,
  }
})
