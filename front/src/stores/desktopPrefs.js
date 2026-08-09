/**
 * Личные настройки рабочего стола: закреплённые в панели задач разделы,
 * разделы меню «Пуск» (свои группы, переименования, состав, порядок, свёрнутость),
 * размер плиток и обои.
 *
 * Раскладка «Пуска» (пункты layouts.*) хранится ОТДЕЛЬНО для стола, мобилы и
 * планшета — это разные экраны с разной геометрией, и человек расставляет
 * плитки на них независимо: широкая на телефоне не обязана быть широкой на
 * столе. Обои, живые плитки и панель задач — общие, это не про раскладку.
 * САМ выбор каркаса сюда не относится: он про устройство, а не про человека,
 * и живёт локально (см. composables/useShellMode.js).
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

export const PLATFORMS = ['desktop', 'mobile', 'tablet']

function emptyLayout() {
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
  }
}

function empty() {
  return {
    layouts: { desktop: emptyLayout(), mobile: emptyLayout(), tablet: emptyLayout() },
    wallpaper: null,
    // Обои экрана блокировки — свой рецепт: запертый экран человек видит
    // чаще, чем рабочий стол, и оформляет его отдельно.
    lockWallpaper: null,
    // Последние загруженные картинки обоев — чтобы вернуться к прошлым.
    wallpapers: [],
    // Живые плитки меню «Пуск» (сводки разделов) — включены по умолчанию.
    liveTiles: true,
    // Плитки, у которых сводки выключены поимённо ({ [appId]: false }).
    // Храним только исключения: по умолчанию живая плитка включена.
    tileLive: {},
    // Где висит панель задач: снизу (как было), сверху, слева или справа.
    // Только про стол — на телефоне панель всегда снизу, своей настройки нет.
    taskbarSide: 'bottom',
    // Меню «Пуск» всегда открывается во весь экран (иначе — обычная панель,
    // а полный экран включается кнопкой в самом меню). Тоже только про стол.
    startFullscreen: false,
  }
}

// Допустимые стороны панели задач: значение приходит с сервера и из чужих
// версий клиента, поэтому проверяем.
export const TASKBAR_SIDES = ['bottom', 'top', 'left', 'right']

function normalizeLayout(raw) {
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
  }
}

function normalize(raw) {
  const p = raw && typeof raw === 'object' ? raw : {}
  // До разделения раскладка хранилась одним плоским объектом на оба каркаса.
  // Если своих layouts ещё нет — прежние данные были рабочим столом (там жила
  // основная настройка), мобильный «Пуск» начинает с чистого листа и человек
  // расставляет его отдельно.
  const legacy = p.layouts && typeof p.layouts === 'object' ? null : p
  return {
    layouts: {
      desktop: normalizeLayout(legacy || p.layouts.desktop),
      mobile: normalizeLayout(legacy ? null : p.layouts.mobile),
      tablet: normalizeLayout(legacy ? null : p.layouts.tablet),
    },
    wallpaper: p.wallpaper && typeof p.wallpaper === 'object' ? p.wallpaper : null,
    lockWallpaper: p.lockWallpaper && typeof p.lockWallpaper === 'object' ? p.lockWallpaper : null,
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

  const wallpaper = computed(() => prefs.value.wallpaper)
  const lockWallpaper = computed(() => prefs.value.lockWallpaper)
  const wallpapers = computed(() => prefs.value.wallpapers)
  const liveTiles = computed(() => prefs.value.liveTiles)
  const taskbarSide = computed(() => prefs.value.taskbarSide)
  const startFullscreen = computed(() => prefs.value.startFullscreen)
  // Вертикальная панель (слева/справа) — другая раскладка кнопок и другие
  // якоря всплывающих панелей.
  const taskbarVertical = computed(() => taskbarSide.value === 'left' || taskbarSide.value === 'right')

  function layoutState(platform) {
    return prefs.value.layouts[platform] || prefs.value.layouts.desktop
  }

  // Раскладка меню «Пуск» одним объектом — её целиком принимает menuGroups().
  function layout(platform) {
    const l = layoutState(platform)
    return { groups: l.groups, labels: l.labels, appGroup: l.appGroup, order: l.order }
  }

  function customized(platform) {
    const l = layoutState(platform)
    return l.groups.length > 0 || Object.keys(l.appGroup).length > 0
  }

  function tileSize(platform, appId, fallback = 'square') {
    return layoutState(platform).tiles[appId] || fallback
  }

  function isPinned(platform, appId) {
    return layoutState(platform).pinned.includes(appId)
  }

  function pinnedList(platform) {
    return layoutState(platform).pinned
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

  function setTileSize(platform, appId, size) {
    const l = layoutState(platform)
    l.tiles = { ...l.tiles, [appId]: size }
    scheduleSave()
  }

  /** Порядок плиток внутри раздела меню «Пуск» (перетаскивание). */
  function setGroupOrder(platform, groupKey, ids) {
    const l = layoutState(platform)
    l.order = { ...l.order, [groupKey]: [...ids] }
    scheduleSave()
  }

  /** Перенос плитки в другой раздел вместе с новым порядком этого раздела. */
  function moveTileToGroup(platform, appId, groupKey, ids) {
    const l = layoutState(platform)
    l.appGroup = { ...l.appGroup, [appId]: groupKey }
    l.order = { ...l.order, [groupKey]: [...ids] }
    scheduleSave()
  }

  function addGroup(platform, label) {
    const l = layoutState(platform)
    const key = `g${Date.now().toString(36)}`
    l.groups = [...l.groups, { key, label: label || 'Новый раздел' }]
    scheduleSave()
    return key
  }

  function renameGroup(platform, groupKey, label) {
    const l = layoutState(platform)
    const own = l.groups.find((g) => g.key === groupKey)
    if (own) l.groups = l.groups.map((g) => (g.key === groupKey ? { ...g, label } : g))
    else l.labels = { ...l.labels, [groupKey]: label }
    scheduleSave()
  }

  /* Удаляем только свой раздел: его плитки возвращаются в родные разделы —
     снимаем переносы, а не прячем сами разделы. */
  function removeGroup(platform, groupKey) {
    const l = layoutState(platform)
    if (!l.groups.some((g) => g.key === groupKey)) return
    l.groups = l.groups.filter((g) => g.key !== groupKey)
    const appGroup = { ...l.appGroup }
    for (const [appId, key] of Object.entries(appGroup)) if (key === groupKey) delete appGroup[appId]
    l.appGroup = appGroup
    const order = { ...l.order }
    delete order[groupKey]
    l.order = order
    const collapsed = { ...l.collapsed }
    delete collapsed[groupKey]
    l.collapsed = collapsed
    scheduleSave()
  }

  function isCollapsed(platform, groupKey) {
    return !!layoutState(platform).collapsed[groupKey]
  }

  function toggleCollapsed(platform, groupKey) {
    const l = layoutState(platform)
    l.collapsed = { ...l.collapsed, [groupKey]: !l.collapsed[groupKey] }
    scheduleSave()
  }

  function pin(platform, appId) {
    if (isPinned(platform, appId)) return
    const l = layoutState(platform)
    l.pinned = [...l.pinned, appId]
    scheduleSave()
  }

  function unpin(platform, appId) {
    const l = layoutState(platform)
    l.pinned = l.pinned.filter((id) => id !== appId)
    scheduleSave()
  }

  function togglePin(platform, appId) {
    isPinned(platform, appId) ? unpin(platform, appId) : pin(platform, appId)
  }

  /** Порядок закреплённых разделов на панели задач (перетаскивание кнопок). */
  function setPinnedOrder(platform, ids) {
    const l = layoutState(platform)
    l.pinned = ids.filter((id) => isPinned(platform, id))
    scheduleSave()
  }

  /** Живые плитки «Пуска»: выключенные показывают обычные иконки, а сводки
      разделов вообще не запрашиваются. Общий тумблер — раскладка своя, а
      какие данные показывать, человек решает одинаково на всех устройствах. */
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

  /** Сторона панели задач: bottom | top | left | right. Только рабочий стол. */
  function setTaskbarSide(side) {
    if (!TASKBAR_SIDES.includes(side)) return
    prefs.value.taskbarSide = side
    scheduleSave()
  }

  /** Открывать ли меню «Пуск» сразу во весь экран. Только рабочий стол. */
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

  // Обои запертого экрана. История картинок общая с рабочим столом: человек
  // выбирает из одного набора, а показываются они в разных местах.
  function setLockWallpaper(recipe) {
    prefs.value.lockWallpaper = recipe ? { ...recipe } : null
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
    prefs, loaded, wallpaper, wallpapers, liveTiles, customized,
    taskbarSide, taskbarVertical, startFullscreen, setTaskbarSide, setStartFullscreen,
    layout, tileSize, isPinned, pinnedList, load, setTileSize, setGroupOrder, moveTileToGroup,
    addGroup, renameGroup, removeGroup, isCollapsed, toggleCollapsed,
    pin, unpin, togglePin, setPinnedOrder, setLiveTiles, isTileLive, setTileLive, toggleTileLive,
    setWallpaper, setLockWallpaper, lockWallpaper, forgetWallpaper, reset,
  }
})
