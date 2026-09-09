/**
 * Личная организация списков разделов: закрепление, ручной порядок и папки в
 * боковых панелях (ежедневники, реестры, формы, расписания, календари).
 *
 * Это не свойство сущности, а личный взгляд на список: в одной панели
 * соседствуют свои и расшаренные записи, и папку человек заводит под себя —
 * владельцу чужого ежедневника она не видна. Отсюда одно место на пользователя
 * (`/api/users/me/lists`, непрозрачный для сервера JSON) вместо таблицы папок в
 * каждом сервисе.
 *
 * Грузится ЛЕНИВО — первым разделом со списком, а не каркасом на старте;
 * localStorage — кэш ради первого кадра, сервер главнее. Запись отложенная:
 * перетаскивание не должно бить в бэкенд на каждый кадр.
 */
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getListPrefs, saveListPrefs } from '@/api/users.js'
import { storageGetJSON, storageSetJSON } from '@/utils/storage.js'
import { LOOSE_KEY, PINNED_KEY } from '@/utils/listOrganize.js'

const CACHE_KEY = 'gw_list_prefs'
const SAVE_DELAY = 600

function emptySection() {
  return { folders: [], collapsed: {}, pinned: [], folderOf: {}, order: [] }
}

function normalizeSection(raw) {
  const p = raw && typeof raw === 'object' ? raw : {}
  const folders = Array.isArray(p.folders)
    ? p.folders
      .filter((f) => f && (typeof f.id === 'string' || typeof f.id === 'number'))
      .map((f) => ({ id: String(f.id), name: String(f.name || 'Папка') }))
    : []
  const ids = new Set(folders.map((f) => f.id))
  const folderOf = {}
  if (p.folderOf && typeof p.folderOf === 'object') {
    for (const [key, fid] of Object.entries(p.folderOf)) {
      // Папку могли удалить на другом устройстве — привязка к ней не нужна.
      if (ids.has(String(fid))) folderOf[String(key)] = String(fid)
    }
  }
  /* Сворачиваются и «закреплённые» с «остальными» — их ключи не id папок,
     поэтому проверка «такая папка ещё есть» на них не распространяется. */
  const collapsed = {}
  if (p.collapsed && typeof p.collapsed === 'object') {
    for (const [key, on] of Object.entries(p.collapsed)) {
      const k = String(key)
      if (on && (ids.has(k) || k === PINNED_KEY || k === LOOSE_KEY)) collapsed[k] = true
    }
  }
  return {
    folders,
    collapsed,
    pinned: Array.isArray(p.pinned) ? [...new Set(p.pinned.map(String))] : [],
    folderOf,
    order: Array.isArray(p.order) ? [...new Set(p.order.map(String))] : [],
  }
}

function normalize(raw) {
  const p = raw && typeof raw === 'object' ? raw : {}
  const out = {}
  for (const [section, value] of Object.entries(p)) out[section] = normalizeSection(value)
  return out
}

export const useListPrefsStore = defineStore('listPrefs', () => {
  const prefs = ref(normalize(storageGetJSON(CACHE_KEY, null)))
  const loaded = ref(false)
  let loading = null
  let timer = null

  /** Настройки раздела: всегда объект — раздел не проверяет наличие ключа. */
  function section(key) {
    return prefs.value[key] || emptySection()
  }

  const folders = computed(() => (key) => section(key).folders)

  function mutate(key, patch) {
    const next = { ...section(key), ...patch }
    prefs.value = { ...prefs.value, [key]: next }
    scheduleSave()
    return next
  }

  function scheduleSave() {
    storageSetJSON(CACHE_KEY, prefs.value)
    clearTimeout(timer)
    timer = setTimeout(() => { saveListPrefs(prefs.value).catch(() => {}) }, SAVE_DELAY)
  }

  /** Разовая загрузка: параллельные вызовы разделов склеиваются в один запрос. */
  function load() {
    if (loaded.value) return Promise.resolve()
    if (loading) return loading
    loading = getListPrefs()
      .then((data) => {
        prefs.value = normalize(data?.prefs)
        storageSetJSON(CACHE_KEY, prefs.value)
      })
      .catch(() => { /* оффлайн — работаем на кэше */ })
      .finally(() => { loaded.value = true; loading = null })
    return loading
  }

  function isPinned(key, id) {
    return section(key).pinned.includes(String(id))
  }

  function togglePin(key, id) {
    const s = section(key)
    const sid = String(id)
    const pinned = s.pinned.includes(sid) ? s.pinned.filter((x) => x !== sid) : [...s.pinned, sid]
    // Закреплённый уходит из папки: он показан отдельной группой, и «числиться»
    // в двух местах сразу нельзя — открепив, человек ждёт его в общем списке.
    const folderOf = { ...s.folderOf }
    if (!s.pinned.includes(sid)) delete folderOf[sid]
    mutate(key, { pinned, folderOf })
  }

  /** Переложить в папку (folderId = '' — вынуть из папок). */
  function setFolder(key, id, folderId) {
    const s = section(key)
    const sid = String(id)
    const folderOf = { ...s.folderOf }
    if (folderId) folderOf[sid] = String(folderId)
    else delete folderOf[sid]
    mutate(key, { folderOf, pinned: s.pinned.filter((x) => x !== sid) })
  }

  /** Ручной порядок раздела — плоская последовательность ключей показа. */
  function setOrder(key, ids) {
    mutate(key, { order: ids.map(String) })
  }

  function addFolder(key, name = 'Новая папка') {
    const s = section(key)
    const id = `f${Date.now().toString(36)}${Math.random().toString(36).slice(2, 5)}`
    mutate(key, { folders: [...s.folders, { id, name }] })
    return id
  }

  function renameFolder(key, folderId, name) {
    const s = section(key)
    mutate(key, {
      folders: s.folders.map((f) => (f.id === String(folderId) ? { ...f, name: name || f.name } : f)),
    })
  }

  /** Удаление папки не трогает сами записи — они возвращаются в общий список. */
  function removeFolder(key, folderId) {
    const s = section(key)
    const fid = String(folderId)
    const folderOf = { ...s.folderOf }
    for (const [item, id] of Object.entries(folderOf)) if (id === fid) delete folderOf[item]
    const collapsed = { ...s.collapsed }
    delete collapsed[fid]
    mutate(key, { folders: s.folders.filter((f) => f.id !== fid), folderOf, collapsed })
  }

  function moveFolder(key, folderId, targetId, before = true) {
    const s = section(key)
    const fid = String(folderId)
    const rest = s.folders.filter((f) => f.id !== fid)
    const moving = s.folders.find((f) => f.id === fid)
    if (!moving) return
    const at = rest.findIndex((f) => f.id === String(targetId))
    if (at === -1) rest.push(moving)
    else rest.splice(before ? at : at + 1, 0, moving)
    mutate(key, { folders: rest })
  }

  function toggleCollapsed(key, folderId) {
    const s = section(key)
    const fid = String(folderId)
    const collapsed = { ...s.collapsed }
    if (collapsed[fid]) delete collapsed[fid]
    else collapsed[fid] = true
    mutate(key, { collapsed })
  }

  /* Забыть удалённую запись. Чистим ТОЛЬКО по явному удалению: список раздела
     показывает не всё сразу (вкладки «Мои» и «Поделились» — разные наборы), и
     чистка по «пропал из текущего списка» стёрла бы папки соседней вкладки. */
  function forget(key, id) {
    const s = section(key)
    const sid = String(id)
    const folderOf = { ...s.folderOf }
    delete folderOf[sid]
    mutate(key, {
      pinned: s.pinned.filter((x) => x !== sid),
      order: s.order.filter((x) => x !== sid),
      folderOf,
    })
  }

  function reset() {
    clearTimeout(timer)
    prefs.value = {}
    loaded.value = false
    loading = null
    storageSetJSON(CACHE_KEY, prefs.value)
  }

  return {
    prefs, loaded, folders,
    load, section, isPinned, togglePin, setFolder, setOrder,
    addFolder, renameFolder, removeFolder, moveFolder, toggleCollapsed, forget, reset,
  }
})
