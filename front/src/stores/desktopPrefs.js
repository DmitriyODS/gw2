/**
 * Личные настройки рабочего стола: закреплённые в панели задач разделы,
 * разделы меню «Пуск» (свои группы, переименования, состав, порядок, свёрнутость),
 * размер плиток и обои.
 *
 * Живут на сервере (`/api/users/me/desktop`), поэтому переезжают между
 * устройствами; localStorage — только кэш для мгновенного первого кадра,
 * сервер всегда главнее. Запись отложенная: щелчки по размеру плиток не
 * должны бить в бэкенд на каждый клик.
 */
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getDesktopPrefs, saveDesktopPrefs } from '@/api/users.js'
import { storageGetJSON, storageSetJSON } from '@/utils/storage.js'

const CACHE_KEY = 'gw_desktop_prefs'
const SAVE_DELAY = 700
// Сколько последних картинок обоев помним для быстрого возврата.
const WALLPAPER_HISTORY = 10

function empty() {
  return {
    pinned: [],
    tiles: {},
    order: {},
    // Свои разделы меню «Пуск», переименования встроенных, куда перенесена
    // плитка и какие разделы свёрнуты.
    groups: [],
    labels: {},
    appGroup: {},
    collapsed: {},
    wallpaper: null,
    // Последние загруженные картинки обоев — чтобы вернуться к прошлым.
    wallpapers: [],
    // Живые плитки меню «Пуск» (сводки разделов) — включены по умолчанию.
    liveTiles: true,
    // Плитки, у которых сводки выключены поимённо ({ [appId]: false }).
    // Храним только исключения: по умолчанию живая плитка включена.
    tileLive: {},
    // Где висит панель задач: снизу (как было), сверху, слева или справа.
    taskbarSide: 'bottom',
    // Меню «Пуск» всегда открывается во весь экран (иначе — обычная панель,
    // а полный экран включается кнопкой в самом меню).
    startFullscreen: false,
  }
}

// Допустимые стороны панели задач: значение приходит с сервера и из чужих
// версий клиента, поэтому проверяем.
export const TASKBAR_SIDES = ['bottom', 'top', 'left', 'right']

function normalize(raw) {
  const p = raw && typeof raw === 'object' ? raw : {}
  return {
    pinned: Array.isArray(p.pinned) ? p.pinned.filter((id) => typeof id === 'string') : [],
    tiles: p.tiles && typeof p.tiles === 'object' ? { ...p.tiles } : {},
    order: p.order && typeof p.order === 'object' ? { ...p.order } : {},
    groups: Array.isArray(p.groups)
      ? p.groups.filter((g) => g && typeof g.key === 'string').map((g) => ({ key: g.key, label: String(g.label || 'Раздел') }))
      : [],
    labels: p.labels && typeof p.labels === 'object' ? { ...p.labels } : {},
    appGroup: p.appGroup && typeof p.appGroup === 'object' ? { ...p.appGroup } : {},
    collapsed: p.collapsed && typeof p.collapsed === 'object' ? { ...p.collapsed } : {},
    wallpaper: p.wallpaper && typeof p.wallpaper === 'object' ? p.wallpaper : null,
    wallpapers: Array.isArray(p.wallpapers)
      ? p.wallpapers.filter((u) => typeof u === 'string').slice(0, WALLPAPER_HISTORY)
      : [],
    liveTiles: p.liveTiles !== false,
    tileLive: p.tileLive && typeof p.tileLive === 'object' ? { ...p.tileLive } : {},
    taskbarSide: TASKBAR_SIDES.includes(p.taskbarSide) ? p.taskbarSide : 'bottom',
    startFullscreen: p.startFullscreen === true,
  }
}

export const useDesktopPrefsStore = defineStore('desktopPrefs', () => {
  const prefs = ref(normalize(storageGetJSON(CACHE_KEY, null)))
  const loaded = ref(false)
  let timer = null

  const pinned = computed(() => prefs.value.pinned)
  const order = computed(() => prefs.value.order)
  const wallpaper = computed(() => prefs.value.wallpaper)
  const wallpapers = computed(() => prefs.value.wallpapers)
  const liveTiles = computed(() => prefs.value.liveTiles)
  const taskbarSide = computed(() => prefs.value.taskbarSide)
  const startFullscreen = computed(() => prefs.value.startFullscreen)
  // Вертикальная панель (слева/справа) — другая раскладка кнопок и другие
  // якоря всплывающих панелей.
  const taskbarVertical = computed(() => taskbarSide.value === 'left' || taskbarSide.value === 'right')
  // Раскладка меню «Пуск» одним объектом — её целиком принимает menuGroups().
  const layout = computed(() => ({
    groups: prefs.value.groups,
    labels: prefs.value.labels,
    appGroup: prefs.value.appGroup,
    order: prefs.value.order,
  }))
  const customized = computed(() =>
    prefs.value.groups.length > 0 || Object.keys(prefs.value.appGroup).length > 0)

  function tileSize(appId, fallback = 'square') {
    return prefs.value.tiles[appId] || fallback
  }

  function isPinned(appId) {
    return prefs.value.pinned.includes(appId)
  }

  async function load() {
    try {
      const data = await getDesktopPrefs()
      prefs.value = normalize(data?.prefs)
      storageSetJSON(CACHE_KEY, prefs.value)
    } catch {
      // Оффлайн/ошибка — остаёмся на кэше, пользователь работает как обычно.
    } finally {
      loaded.value = true
    }
  }

  function scheduleSave() {
    storageSetJSON(CACHE_KEY, prefs.value)
    clearTimeout(timer)
    timer = setTimeout(() => {
      saveDesktopPrefs(prefs.value).catch(() => {})
    }, SAVE_DELAY)
  }

  function setTileSize(appId, size) {
    prefs.value.tiles = { ...prefs.value.tiles, [appId]: size }
    scheduleSave()
  }

  /** Порядок плиток внутри раздела меню «Пуск» (перетаскивание). */
  function setGroupOrder(groupKey, ids) {
    prefs.value.order = { ...prefs.value.order, [groupKey]: [...ids] }
    scheduleSave()
  }

  /** Перенос плитки в другой раздел вместе с новым порядком этого раздела. */
  function moveTileToGroup(appId, groupKey, ids) {
    prefs.value.appGroup = { ...prefs.value.appGroup, [appId]: groupKey }
    prefs.value.order = { ...prefs.value.order, [groupKey]: [...ids] }
    scheduleSave()
  }

  function addGroup(label) {
    const key = `g${Date.now().toString(36)}`
    prefs.value.groups = [...prefs.value.groups, { key, label: label || 'Новый раздел' }]
    scheduleSave()
    return key
  }

  function renameGroup(groupKey, label) {
    const own = prefs.value.groups.find((g) => g.key === groupKey)
    if (own) prefs.value.groups = prefs.value.groups.map((g) => (g.key === groupKey ? { ...g, label } : g))
    else prefs.value.labels = { ...prefs.value.labels, [groupKey]: label }
    scheduleSave()
  }

  /* Удаляем только свой раздел: его плитки возвращаются в родные разделы —
     снимаем переносы, а не прячем сами разделы. */
  function removeGroup(groupKey) {
    if (!prefs.value.groups.some((g) => g.key === groupKey)) return
    prefs.value.groups = prefs.value.groups.filter((g) => g.key !== groupKey)
    const appGroup = { ...prefs.value.appGroup }
    for (const [appId, key] of Object.entries(appGroup)) if (key === groupKey) delete appGroup[appId]
    prefs.value.appGroup = appGroup
    const order = { ...prefs.value.order }
    delete order[groupKey]
    prefs.value.order = order
    const collapsed = { ...prefs.value.collapsed }
    delete collapsed[groupKey]
    prefs.value.collapsed = collapsed
    scheduleSave()
  }

  function isCollapsed(groupKey) {
    return !!prefs.value.collapsed[groupKey]
  }

  function toggleCollapsed(groupKey) {
    prefs.value.collapsed = { ...prefs.value.collapsed, [groupKey]: !prefs.value.collapsed[groupKey] }
    scheduleSave()
  }

  function pin(appId) {
    if (isPinned(appId)) return
    prefs.value.pinned = [...prefs.value.pinned, appId]
    scheduleSave()
  }

  function unpin(appId) {
    prefs.value.pinned = prefs.value.pinned.filter((id) => id !== appId)
    scheduleSave()
  }

  function togglePin(appId) {
    isPinned(appId) ? unpin(appId) : pin(appId)
  }

  /** Порядок закреплённых разделов на панели задач (перетаскивание кнопок). */
  function setPinnedOrder(ids) {
    prefs.value.pinned = ids.filter((id) => isPinned(id))
    scheduleSave()
  }

  /** Живые плитки «Пуска»: выключенные показывают обычные иконки, а сводки
      разделов вообще не запрашиваются. */
  function setLiveTiles(on) {
    prefs.value.liveTiles = !!on
    scheduleSave()
  }

  /** Живая ли плитка КОНКРЕТНОГО раздела (общий тумблер важнее личного). */
  function isTileLive(appId) {
    return prefs.value.liveTiles && prefs.value.tileLive[appId] !== false
  }

  /* Сводки одной плитки. Выключенная не только показывает обычный значок, но и
     не попадает в опрос источников — лишних запросов не будет. */
  function setTileLive(appId, on) {
    const next = { ...prefs.value.tileLive }
    if (on) delete next[appId]
    else next[appId] = false
    prefs.value.tileLive = next
    scheduleSave()
  }

  function toggleTileLive(appId) {
    setTileLive(appId, prefs.value.tileLive[appId] === false)
  }

  /** Сторона панели задач: bottom | top | left | right. */
  function setTaskbarSide(side) {
    if (!TASKBAR_SIDES.includes(side)) return
    prefs.value.taskbarSide = side
    scheduleSave()
  }

  /** Открывать ли меню «Пуск» сразу во весь экран. */
  function setStartFullscreen(on) {
    prefs.value.startFullscreen = !!on
    scheduleSave()
  }

  function setWallpaper(recipe) {
    prefs.value.wallpaper = recipe ? { ...recipe } : null
    const url = recipe?.image?.url
    if (url) {
      prefs.value.wallpapers = [url, ...prefs.value.wallpapers.filter((u) => u !== url)]
        .slice(0, WALLPAPER_HISTORY)
    }
    scheduleSave()
  }

  function forgetWallpaper(url) {
    prefs.value.wallpapers = prefs.value.wallpapers.filter((u) => u !== url)
    scheduleSave()
  }

  function reset() {
    clearTimeout(timer)
    prefs.value = empty()
    loaded.value = false
    storageSetJSON(CACHE_KEY, prefs.value)
  }

  return {
    prefs, loaded, pinned, order, wallpaper, wallpapers, liveTiles, layout, customized,
    taskbarSide, taskbarVertical, startFullscreen, setTaskbarSide, setStartFullscreen,
    tileSize, isPinned, load, setTileSize, setGroupOrder, moveTileToGroup,
    addGroup, renameGroup, removeGroup, isCollapsed, toggleCollapsed,
    pin, unpin, togglePin, setPinnedOrder, setLiveTiles, isTileLive, setTileLive, toggleTileLive,
    setWallpaper, forgetWallpaper, reset,
  }
})
