import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@/api/notes.js'
import { useAuthStore } from '@/stores/auth.js'

// Единое окно заметок: иерархические папки (свои + расшаренные мне) слева/в
// проводнике, теги-метки, плитки заметок. Скоуп по владельцу на сервере;
// эффективный доступ (шары, расшаренные папки-предки) считает сервер, клиент
// лишь отражает my_access. Два режима отображения: hierarchy | explorer.
export const useNotesStore = defineStore('notes', () => {
  const viewMode = ref(localStorage.getItem('gw_notes_view') || 'hierarchy')

  const folders = ref([])        // свои папки (плоско, parent_id)
  const sharedRoots = ref([])    // расшаренные мне «корни»
  const tags = ref([])
  const notes = ref([])          // плитки текущей выборки

  const loading = ref(false)
  const loadingFolders = ref(false)

  // Текущая выборка. activeFolderId: null — корень/«Все»; number — папка.
  const activeFolderId = ref(null)
  const showArchived = ref(false)   // фильтр «Архив»
  const showShared = ref(false)     // агрегат «Поделились со мной»
  const showAllFlat = ref(false)    // «Все заметки» плоским списком (проводник)
  const activeTagIds = ref([])
  const search = ref('')

  // Проводник: путь хлебных крошек + подпапки текущего расположения.
  const path = ref([])              // [{id, name, owner_id, my_access}]
  const browseChildren = ref([])    // папки-плитки текущего расположения (explorer)

  // Мультивыбор в проводнике: множества id.
  const selectedNoteIds = ref(new Set())
  const selectedFolderIds = ref(new Set())

  let fetchSeq = 0
  let fetchCtrl = null

  const myId = () => useAuthStore().userId

  // ── Производные ──
  const folderById = computed(() => {
    const m = new Map()
    for (const f of folders.value) m.set(f.id, f)
    for (const f of sharedRoots.value) m.set(f.id, f)
    return m
  })

  // Дерево своих папок (для сайдбара иерархии).
  const folderTree = computed(() => buildTree(folders.value, null))

  function buildTree(list, parentId) {
    return list
      .filter((f) => (f.parent_id ?? null) === parentId)
      .sort((a, b) => (a.position - b.position) || a.name.localeCompare(b.name))
      .map((f) => ({ ...f, children: buildTree(list, f.id) }))
  }

  const childrenOf = (parentId) =>
    folders.value.filter((f) => (f.parent_id ?? null) === (parentId ?? null))

  const activeFolder = computed(() => (activeFolderId.value ? folderById.value.get(activeFolderId.value) : null))
  const isSharedContext = computed(() =>
    !!activeFolder.value && activeFolder.value.owner_id !== myId())

  const hasSelection = computed(() => selectedNoteIds.value.size + selectedFolderIds.value.size > 0)
  const selectionCount = computed(() => selectedNoteIds.value.size + selectedFolderIds.value.size)

  // ── Загрузка ──
  async function fetchFolders({ silent = false } = {}) {
    if (!silent) loadingFolders.value = true
    try {
      const data = await api.getFolders()
      folders.value = data.folders ?? []
      sharedRoots.value = data.shared ?? []
      // Активная папка исчезла — вернёмся в корень.
      if (activeFolderId.value && !folderById.value.get(activeFolderId.value)) {
        selectAll()
      } else if (viewMode.value === 'explorer' && !showShared.value && !showArchived.value) {
        // Папки подгрузились — обновим плитки-папки текущего расположения.
        fetchBrowseChildren()
      }
    } finally {
      if (!silent) loadingFolders.value = false
    }
  }

  async function fetchTags() {
    const data = await api.getTags()
    tags.value = data.tags ?? []
  }

  function notesParams() {
    if (showShared.value) return { shared: '1', search: search.value }
    const p = { search: search.value }
    if (showArchived.value) p.archived = '1'
    if (activeTagIds.value.length) p.tag_ids = activeTagIds.value.join(',')
    // При активном поиске фильтр по папке снимается — ищем ГЛОБАЛЬНО (по всем
    // папкам и подпапкам). Иначе: hierarchy без папки — все, explorer в корне — корень.
    if (!search.value && !showAllFlat.value) {
      if (activeFolderId.value) p.folder_id = activeFolderId.value
      else if (viewMode.value === 'explorer' && !showArchived.value) p.folder_id = 'root'
    }
    return p
  }

  // Верхний уровень («Все»/корень) и любой ПОИСК — тут в общую кучу
  // подмешиваются и расшаренные мне заметки (с пометкой). НЕ подмешиваем при
  // фильтре по тегам (теги личные — у чужих заметок их нет) и в конкретной
  // папке/архиве без поиска.
  const includeShared = () =>
    !showShared.value && !showArchived.value && !activeTagIds.value.length
    && (!activeFolderId.value || !!search.value)

  async function fetchNotes({ silent = false } = {}) {
    const seq = ++fetchSeq
    fetchCtrl?.abort()
    fetchCtrl = new AbortController()
    if (!silent) loading.value = true
    const opt = { signal: fetchCtrl.signal }
    try {
      const [own, shared] = await Promise.all([
        api.getNotes(notesParams(), opt),
        includeShared() ? api.getNotes({ shared: '1', search: search.value }, opt) : Promise.resolve({ notes: [] }),
      ])
      if (seq !== fetchSeq) return
      notes.value = mergeNotes(own.notes ?? [], shared.notes ?? [])
    } catch (e) {
      if (e?.name !== 'AbortError' && e?.error !== 'ABORTED') throw e
    } finally {
      if (seq === fetchSeq) loading.value = false
    }
  }

  // Свои + расшаренные без дублей (свои имеют приоритет), закреплённые/свежие выше.
  function mergeNotes(own, shared) {
    const seen = new Set(own.map((n) => n.id))
    const list = [...own, ...shared.filter((n) => !seen.has(n.id))]
    return list.sort((a, b) =>
      String(b.pinned_at || '').localeCompare(String(a.pinned_at || ''))
      || String(b.updated_at).localeCompare(String(a.updated_at)))
  }

  // Подпапки текущего расположения проводника: свои — из дерева, чужие — с сервера.
  async function fetchBrowseChildren() {
    if (!activeFolderId.value) {
      browseChildren.value = [...childrenOf(null), ...sharedRoots.value]
      return
    }
    if (!isSharedContext.value) {
      browseChildren.value = childrenOf(activeFolderId.value)
      return
    }
    try {
      const data = await api.getFolderChildren(activeFolderId.value)
      browseChildren.value = data.folders ?? []
    } catch {
      browseChildren.value = []
    }
  }

  // ── Навигация ──
  function clearSelection() {
    selectedNoteIds.value = new Set()
    selectedFolderIds.value = new Set()
  }

  function refresh() {
    fetchNotes()
    if (viewMode.value === 'explorer') fetchBrowseChildren()
  }

  function selectAll() {
    activeFolderId.value = null
    showArchived.value = false
    showShared.value = false
    showAllFlat.value = false
    path.value = []
    clearSelection()
    refresh()
  }

  // «Все заметки» плоским списком (особая группировка проводника — все заметки
  // из всех папок сразу, + расшаренные).
  function selectAllFlat() {
    activeFolderId.value = null
    showArchived.value = false
    showShared.value = false
    showAllFlat.value = true
    path.value = []
    clearSelection()
    fetchNotes()
  }

  function selectFolder(id) {
    activeFolderId.value = id
    showArchived.value = false
    showShared.value = false
    showAllFlat.value = false
    clearSelection()
    rebuildPath(id)
    refresh()
  }

  function selectShared() {
    activeFolderId.value = null
    showArchived.value = false
    showShared.value = true
    showAllFlat.value = false
    path.value = []
    clearSelection()
    // В агрегате «Поделились» подпапки-плитки — расшаренные корни.
    browseChildren.value = [...sharedRoots.value]
    fetchNotes()
  }

  function selectArchive() {
    activeFolderId.value = null
    showArchived.value = true
    showShared.value = false
    showAllFlat.value = false
    path.value = []
    clearSelection()
    fetchNotes()
    if (viewMode.value === 'explorer') browseChildren.value = []
  }

  // Открыть папку в проводнике (двойной клик).
  function openFolder(folder) {
    activeFolderId.value = folder.id
    showArchived.value = false
    showShared.value = false
    showAllFlat.value = false
    clearSelection()
    rebuildPath(folder.id)
    refresh()
  }

  function navigateTo(index) {
    // index -1 — корень; иначе элемент пути.
    if (index < 0) { selectAll(); return }
    const target = path.value[index]
    if (target) openFolder(target)
  }

  // Восстановить путь до папки id по своему дереву (для крошек и подсветки).
  function rebuildPath(id) {
    const chain = []
    let cur = folderById.value.get(id)
    // Для чужого поддерева цепочку целиком не построить — показываем хотя бы саму папку.
    while (cur) {
      chain.unshift({ id: cur.id, name: cur.name, owner_id: cur.owner_id, my_access: cur.my_access })
      cur = cur.parent_id ? folderById.value.get(cur.parent_id) : null
    }
    path.value = chain
  }

  function setViewMode(mode) {
    if (viewMode.value === mode) return
    viewMode.value = mode
    localStorage.setItem('gw_notes_view', mode)
    clearSelection()
    refresh()
  }

  function setSearch(v) {
    search.value = v
    fetchNotes()
  }

  function toggleTag(id) {
    const i = activeTagIds.value.indexOf(id)
    if (i === -1) activeTagIds.value = [...activeTagIds.value, id]
    else activeTagIds.value = activeTagIds.value.filter((t) => t !== id)
    fetchNotes()
  }

  function clearTags() {
    if (!activeTagIds.value.length) return
    activeTagIds.value = []
    fetchNotes()
  }

  // ── Выделение (проводник) ──
  function toggleNoteSelect(id, additive) {
    const s = new Set(additive ? selectedNoteIds.value : [])
    if (!additive) selectedFolderIds.value = new Set()
    if (s.has(id)) s.delete(id)
    else s.add(id)
    selectedNoteIds.value = s
  }

  function toggleFolderSelect(id, additive) {
    const s = new Set(additive ? selectedFolderIds.value : [])
    if (!additive) selectedNoteIds.value = new Set()
    if (s.has(id)) s.delete(id)
    else s.add(id)
    selectedFolderIds.value = s
  }

  // ── Мутации: заметки ──
  async function createNote(title = '', folderOverride = undefined) {
    const folderId = folderOverride !== undefined
      ? folderOverride
      : (activeFolderId.value && !isSharedContext.value ? activeFolderId.value : null)
    const n = await api.createNote(title, folderId)
    upsertNote(n)
    return n
  }

  async function importNote(file) {
    const folderId = activeFolderId.value && !isSharedContext.value ? activeFolderId.value : null
    const n = await api.importNote(file, folderId)
    upsertNote(n)
    return n
  }

  async function removeNote(id) {
    await api.deleteNote(id)
    dropNote(id)
    fetchFolders({ silent: true })
  }

  async function copyNote(id) {
    const n = await api.copyNote(id)
    upsertNote(n)
    return n
  }

  async function moveNote(id, folderId) {
    await api.moveNote(id, folderId)
    // Заметка ушла из текущей выборки, если та привязана к другой папке.
    if (activeFolderId.value || viewMode.value === 'explorer') dropNote(id)
    fetchFolders({ silent: true })
  }

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

  async function setPinned(id, pinned) {
    const n = await api.updateNote(id, { pinned })
    upsertNote(n)
    sortTiles()
    return n
  }

  async function setNoteColor(id, color) {
    const n = await api.updateNote(id, { color })
    upsertNote(n)
    return n
  }

  // ── Мутации: папки ──
  async function createFolder(name, parentId = null, color = '') {
    const f = await api.createFolder(name, parentId, color)
    upsertFolder(f)
    return f
  }

  async function renameFolder(id, name, color) {
    const f = await api.updateFolder(id, { name, color })
    upsertFolder(f)
    return f
  }

  async function moveFolder(id, parentId) {
    const f = await api.moveFolder(id, parentId)
    upsertFolder(f)
    if (viewMode.value === 'explorer') fetchBrowseChildren()
    return f
  }

  async function copyFolder(id) {
    const f = await api.copyFolder(id)
    upsertFolder(f)
    if (viewMode.value === 'explorer') fetchBrowseChildren()
    return f
  }

  async function removeFolder(id) {
    await api.deleteFolder(id)
    folders.value = folders.value.filter((f) => f.id !== id)
    if (activeFolderId.value === id) selectAll()
    else refresh()
    fetchFolders({ silent: true })
  }

  // ── Мутации: теги ──
  // upsertTag — идемпотентно: сокет note_tag:created мог успеть добавить тег
  // ДО того, как вернётся HTTP-ответ createTag; вслепую пушить нельзя (дубль).
  function upsertTag(payload) {
    const i = tags.value.findIndex((t) => t.id === payload.id)
    if (i === -1) tags.value = [...tags.value, payload]
    else tags.value[i] = { ...tags.value[i], ...payload }
  }

  async function createTag(name, color = '') {
    const t = await api.createTag(name, color)
    upsertTag(t)
    return t
  }

  async function renameTag(id, name, color) {
    const t = await api.updateTag(id, { name, color })
    const i = tags.value.findIndex((x) => x.id === id)
    if (i !== -1) tags.value[i] = { ...tags.value[i], ...t }
    return t
  }

  async function removeTag(id) {
    await api.deleteTag(id)
    tags.value = tags.value.filter((t) => t.id !== id)
    activeTagIds.value = activeTagIds.value.filter((t) => t !== id)
    refresh()
  }

  async function setNoteTags(id, tagIds) {
    const n = await api.setNoteTags(id, tagIds)
    upsertNote(n)
    fetchTags()
    return n
  }

  // ── Идемпотентные апдейты ──
  function upsertNote(payload) {
    // Плитка не из текущей выборки — не добавляем (события чужих папок/архива).
    const i = notes.value.findIndex((n) => n.id === payload.id)
    if (i === -1) notes.value = [payload, ...notes.value]
    else notes.value[i] = { ...notes.value[i], ...payload }
  }

  function dropNote(id) {
    notes.value = notes.value.filter((n) => n.id !== id)
    selectedNoteIds.value.delete(id)
  }

  function upsertFolder(payload) {
    const i = folders.value.findIndex((f) => f.id === payload.id)
    if (i === -1) folders.value = [...folders.value, payload]
    else folders.value[i] = { ...folders.value[i], ...payload }
  }

  function sortTiles() {
    notes.value = [...notes.value].sort((a, b) =>
      String(b.pinned_at || '').localeCompare(String(a.pinned_at || ''))
      || String(b.updated_at).localeCompare(String(a.updated_at)))
  }

  // ── Сокеты ──
  function belongsHere(payload) {
    const mine = !payload.owner_id || payload.owner_id === myId()
    if (showShared.value) return !mine
    // Расшаренная мне заметка — в общей куче верхнего уровня (с пометкой).
    if (!mine) return includeShared()
    if (!!payload.archived !== showArchived.value) return false
    // Фильтр по папке (если выбрана и это не иерархия-«Все»).
    if (activeFolderId.value) {
      return (payload.folder_id ?? null) === activeFolderId.value
    }
    if (viewMode.value === 'explorer' && !showArchived.value) {
      return (payload.folder_id ?? null) === null
    }
    return true
  }

  function applyNoteSocket(kind, payload) {
    if (kind === 'deleted') { dropNote(payload.id); fetchFolders({ silent: true }); return }
    if (search.value) { fetchNotes({ silent: true }); return }
    if (belongsHere(payload)) { upsertNote(payload); sortTiles() }
    else dropNote(payload.id)
    fetchFolders({ silent: true })
  }

  function applyFolderSocket(kind, payload) {
    if (kind === 'deleted') {
      folders.value = folders.value.filter((f) => f.id !== payload.id)
      if (activeFolderId.value === payload.id) selectAll()
      return
    }
    if (payload.owner_id === myId()) upsertFolder(payload)
    if (viewMode.value === 'explorer') fetchBrowseChildren()
  }

  function applyTagSocket(kind, payload) {
    if (kind === 'deleted') {
      tags.value = tags.value.filter((t) => t.id !== payload.id)
      return
    }
    upsertTag(payload)
  }

  // Заметка/папка появилась или пропала в «Поделились со мной».
  function applyShareSocket() {
    fetchFolders({ silent: true })
    if (showShared.value || includeShared()) fetchNotes({ silent: true })
  }

  return {
    // state
    viewMode, folders, sharedRoots, tags, notes, loading, loadingFolders,
    activeFolderId, showArchived, showShared, showAllFlat, activeTagIds, search,
    path, browseChildren, selectedNoteIds, selectedFolderIds,
    // computed
    folderById, folderTree, activeFolder, isSharedContext, hasSelection, selectionCount,
    childrenOf,
    // load / navigate
    fetchFolders, fetchTags, fetchNotes, fetchBrowseChildren,
    selectAll, selectAllFlat, selectFolder, selectShared, selectArchive, openFolder, navigateTo,
    setViewMode, setSearch, toggleTag, clearTags, clearSelection,
    toggleNoteSelect, toggleFolderSelect,
    // mutations
    createNote, importNote, removeNote, copyNote, moveNote, setArchived, setPinned, setNoteColor,
    createFolder, renameFolder, moveFolder, copyFolder, removeFolder,
    createTag, renameTag, removeTag, setNoteTags,
    upsertNote, upsertFolder,
    // sockets
    applyNoteSocket, applyFolderSocket, applyTagSocket, applyShareSocket,
  }
})
