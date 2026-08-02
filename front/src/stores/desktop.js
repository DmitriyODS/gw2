/**
 * Оконный менеджер рабочего стола: список открытых окон, их геометрия,
 * z-порядок, свёрнутость и «маршрут» каждого окна.
 *
 * Окно — это раздел приложения, открытый по пути роутера (`path`). Один раздел
 * можно открыть много раз: у каждого окна свой путь, своя история переходов и
 * своя геометрия. Компонент раздела окно берёт из router.resolve(path) —
 * маршруты остаются единственным источником истины (см. WindowContent.vue).
 */
import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import router from '@/router/index.js'
import { appById, appForPath } from '@/desktop/apps.js'
import { cascadeRect, clampPosition, clampSize, rectForZone, refitRect } from '@/desktop/geometry.js'
import { storageGetJSON, storageSetJSON } from '@/utils/storage.js'
import { logSection } from '@/utils/activityLog.js'

const STATE_KEY = 'gw_desktop_session'

function minOf(app) {
  return { w: app?.min?.[0] ?? 380, h: app?.min?.[1] ?? 280 }
}

export const useDesktopStore = defineStore('desktop', () => {
  const windows = ref([])
  const focusedId = ref(null)
  /* Предел одновременно открытых разделов (0 — без предела). Нужен мобильному
     каркасу: там разделы тоже остаются смонтированными, а памяти у телефона
     меньше — самый давний по последнему обращению экран закрывается сам. */
  const limit = ref(0)
  const startOpen = ref(false)
  // Меню «Пуск» раскрыто во весь экран: панель задач на это время прячется —
  // экран занимает только меню, пока не выберут раздел. Флаг рантаймовый
  // (личная настройка «всегда во весь экран» живёт в desktopPrefs).
  const startFull = ref(false)
  const notifOpen = ref(false)
  // Всплывающая панель Hola (поиск, команды, чат) — поверх стола, вне окон.
  const holaOpen = ref(false)
  // Подсветка зоны прилипания во время перетаскивания окна.
  const snapPreview = ref(null)
  // Рабочая область (экран минус панель задач) — задаёт DesktopShell.
  const area = reactive({ x: 0, y: 0, w: 1280, h: 720 })
  // Весь экран: развёрнутое окно занимает его целиком, без полей и без панели
  // задач — как обычное одноэкранное приложение.
  const screen = reactive({ x: 0, y: 0, w: 1280, h: 800 })
  // Фактическая геометрия панели задач: она центрирована и уже экрана, а меню
  // «Пуск» и центр уведомлений выравниваются по её краям.
  const taskbarRect = reactive({ x: 0, y: 0, w: 0, h: 0 })
  // Центр кнопки уведомлений по горизонтали — центр панели уведомлений.
  const bellCenter = ref(0)

  let seq = 1
  let zTop = 10

  const focused = computed(() => windows.value.find((w) => w.id === focusedId.value) || null)
  const hasWindows = computed(() => windows.value.length > 0)

  /* Полноэкранный режим: активное окно развёрнуто. Панель задач в нём прячется
     (выезжает обратно при подведении указателя к нижнему краю), а сворачивание,
     закрытие или возврат размера окна возвращают её сразу. */
  const fullscreen = computed(() => !!focused.value && !focused.value.minimized && focused.value.mode === 'max')
  const taskbarPeek = ref(false)

  /** Прямоугольник зоны прилипания: «во весь экран» — это весь экран. */
  function zoneRect(zone) {
    return rectForZone(zone, zone === 'max' ? screen : area)
  }

  /* ── Служебное ─────────────────────────────────────────────── */

  function persist() {
    storageSetJSON(STATE_KEY, {
      windows: windows.value.map((w) => ({
        appId: w.appId, path: w.path, x: w.x, y: w.y, w: w.w, h: w.h,
        mode: w.mode, snap: w.snap, minimized: w.minimized,
      })),
      focusedIndex: windows.value.findIndex((w) => w.id === focusedId.value),
    })
  }

  function makeWindow(app, path, rect) {
    return reactive({
      id: `w${seq++}`,
      appId: app.id,
      path,
      history: [path],
      hi: 0,
      x: rect.x, y: rect.y, w: rect.w, h: rect.h,
      // Геометрия до разворота/прилипания — чтобы вернуть окно как было.
      restore: null,
      mode: 'normal', // 'normal' | 'max' | 'snap'
      snap: null,
      minimized: false,
      z: ++zTop,
    })
  }

  /* ── Открытие и навигация ──────────────────────────────────── */

  /**
   * Открывает раздел по пути. По умолчанию переиспользует уже открытое окно
   * этого раздела (как док настольной ОС); newWindow — всегда новое окно.
   */
  function open(path, { newWindow = false } = {}) {
    const resolved = router.resolve(path)
    const app = appForPath(resolved.path)
    if (!app) return null

    // Раздел открыт пользователем — помечаем в ленте «Недавние разделы».
    // Восстановление сессии окна не создаёт через open(), так что перезагрузка
    // страницы список не перетасовывает.
    logSection(app.id)

    if (!newWindow) {
      const existing = windows.value.find((w) => w.appId === app.id)
      if (existing) {
        navigate(existing.id, resolved.fullPath)
        restore(existing.id)
        focus(existing.id)
        return existing
      }
    }

    const size = { w: app.size?.[0] ?? 1000, h: app.size?.[1] ?? 700 }
    const win = makeWindow(app, resolved.fullPath, cascadeRect(windows.value.length, size, area))
    windows.value.push(win)
    focus(win.id)
    enforceLimit(win.id)
    persist()
    return win
  }

  /** Закрывает самые давние окна, пока их больше предела (см. limit). */
  function enforceLimit(keepId = null) {
    if (!limit.value) return
    while (windows.value.length > limit.value) {
      // Порядок z — очерёдность последнего обращения: наименьший и уходит.
      const victim = [...windows.value]
        .filter((w) => w.id !== keepId)
        .sort((a, b) => a.z - b.z)[0]
      if (!victim) return
      close(victim.id)
    }
  }

  function openApp(appId, opts) {
    const app = appById(appId)
    return app ? open(app.path, opts) : null
  }

  /** Переход внутри окна (как переход по ссылке в браузере) с историей. */
  function navigate(id, path, { replace = false } = {}) {
    const win = byId(id)
    if (!win || win.path === path) return
    if (replace) {
      win.history[win.hi] = path
    } else {
      win.history = win.history.slice(0, win.hi + 1)
      win.history.push(path)
      win.hi = win.history.length - 1
    }
    win.path = path
    // Раздел мог смениться (например, /notes/5 → /notes): окно следует за ним.
    const app = appForPath(router.resolve(path).path)
    if (app) win.appId = app.id
    persist()
  }

  function back(id) {
    const win = byId(id)
    if (!win || win.hi <= 0) return
    win.hi -= 1
    win.path = win.history[win.hi]
    persist()
  }

  function canGoBack(id) {
    const win = byId(id)
    return !!win && win.hi > 0
  }

  /* ── Состояние окон ────────────────────────────────────────── */

  function byId(id) {
    return windows.value.find((w) => w.id === id) || null
  }

  /* Слои окон живут в диапазоне ниже панели задач (900) и диалогов PrimeVue
     (~1100): модалка раздела обязана перекрывать панель, а окно — нет.
     Поэтому счётчик не растёт бесконечно: дойдя до потолка, порядок
     пересобирается с нуля, сохраняя взаимное перекрытие. */
  function normalizeZ() {
    const ordered = [...windows.value].sort((a, b) => a.z - b.z)
    ordered.forEach((w, i) => { w.z = 10 + i })
    zTop = 10 + ordered.length
  }

  function focus(id) {
    const win = byId(id)
    if (!win) return
    if (focusedId.value !== id || win.z < zTop) win.z = ++zTop
    focusedId.value = id
    startOpen.value = false
    notifOpen.value = false
    if (zTop > 800) normalizeZ()
  }

  function close(id) {
    const i = windows.value.findIndex((w) => w.id === id)
    if (i < 0) return
    windows.value.splice(i, 1)
    if (focusedId.value === id) {
      const top = [...windows.value].filter((w) => !w.minimized).sort((a, b) => b.z - a.z)[0]
      focusedId.value = top?.id ?? null
    }
    persist()
  }

  function minimize(id) {
    const win = byId(id)
    if (!win) return
    win.minimized = true
    if (focusedId.value === id) {
      const top = [...windows.value].filter((w) => !w.minimized).sort((a, b) => b.z - a.z)[0]
      focusedId.value = top?.id ?? null
    }
    persist()
  }

  function restore(id) {
    const win = byId(id)
    if (!win) return
    win.minimized = false
  }

  /** Клик по кнопке в панели задач: сфокусированное окно сворачивается. */
  function toggleFromTaskbar(id) {
    const win = byId(id)
    if (!win) return
    if (win.minimized) { restore(id); focus(id); return }
    if (focusedId.value === id) minimize(id)
    else focus(id)
  }

  function maximize(id) {
    const win = byId(id)
    if (!win || win.mode === 'max') return
    win.restore = win.mode === 'normal' ? { x: win.x, y: win.y, w: win.w, h: win.h } : win.restore
    Object.assign(win, zoneRect('max'))
    win.mode = 'max'
    win.snap = null
    persist()
  }

  /** Возврат к «нормальной» геометрии из развёрнутого/прижатого состояния. */
  function unmaximize(id) {
    const win = byId(id)
    if (!win || win.mode === 'normal') return
    const app = appById(win.appId)
    const rect = win.restore || { x: win.x, y: win.y, w: Math.round(area.w * 0.6), h: Math.round(area.h * 0.7) }
    Object.assign(win, refitRect(rect, area, minOf(app)))
    win.mode = 'normal'
    win.snap = null
    win.restore = null
    persist()
  }

  function toggleMaximize(id) {
    const win = byId(id)
    if (!win) return
    if (win.mode === 'normal') maximize(id)
    else unmaximize(id)
  }

  /** Прилипание к зоне экрана (половина/четверть/во весь экран). */
  function snapTo(id, zone) {
    const win = byId(id)
    const rect = zoneRect(zone)
    if (!win || !rect) return
    if (win.mode === 'normal') win.restore = { x: win.x, y: win.y, w: win.w, h: win.h }
    Object.assign(win, rect)
    win.mode = zone === 'max' ? 'max' : 'snap'
    win.snap = zone === 'max' ? null : zone
    persist()
  }

  /** Свободная геометрия (перетаскивание/изменение размера) — снимает прилипание. */
  function setRect(id, rect, { commit = false } = {}) {
    const win = byId(id)
    if (!win) return
    const app = appById(win.appId)
    Object.assign(win, clampSize({ ...rect }, area, minOf(app)))
    win.mode = 'normal'
    win.snap = null
    win.restore = null
    if (commit) persist()
  }

  function setPosition(id, x, y, { commit = false } = {}) {
    const win = byId(id)
    if (!win) return
    Object.assign(win, clampPosition({ x, y, w: win.w, h: win.h }, area))
    if (commit) persist()
  }

  /* ── Рабочая область и восстановление сессии ───────────────── */

  function setScreen(rect) {
    Object.assign(screen, rect)
  }

  function setArea(rect) {
    const changed = area.x !== rect.x || area.y !== rect.y || area.w !== rect.w || area.h !== rect.h
    Object.assign(area, rect)
    if (!changed) return
    for (const win of windows.value) {
      if (win.mode === 'max') Object.assign(win, zoneRect('max'))
      else if (win.mode === 'snap' && win.snap) Object.assign(win, rectForZone(win.snap, area))
      else Object.assign(win, refitRect({ x: win.x, y: win.y, w: win.w, h: win.h }, area, minOf(appById(win.appId))))
    }
  }

  /**
   * Восстанавливает окна прошлой сессии. Разделы, недоступные текущему
   * пользователю (сменилась компания/роль), пропускаем — canOpen проверяет
   * вызывающая сторона через фильтр available.
   */
  function restoreSession(isAvailable) {
    /* Каркас перемонтируется при смене раскладки (изменение размера окна через
       границу мобильного каркаса) и зовёт boot() заново, а стор переживает это
       вместе с окнами. Восстанавливаем ТОЛЬКО пустой стол — иначе каждая такая
       смена удваивала бы открытые разделы. */
    if (windows.value.length) return true

    const saved = storageGetJSON(STATE_KEY, null)
    if (!saved?.windows?.length) return false
    for (const s of saved.windows) {
      const app = appById(s.appId)
      if (!app || !isAvailable(app)) continue
      const win = makeWindow(app, s.path || app.path, refitRect({ x: s.x, y: s.y, w: s.w, h: s.h }, area, minOf(app)))
      win.mode = s.mode === 'max' || s.mode === 'snap' ? s.mode : 'normal'
      win.snap = s.snap || null
      win.minimized = !!s.minimized
      if (win.mode === 'max') Object.assign(win, zoneRect('max'))
      else if (win.mode === 'snap' && win.snap) Object.assign(win, rectForZone(win.snap, area))
      windows.value.push(win)
    }
    const target = windows.value[saved.focusedIndex] || [...windows.value].reverse().find((w) => !w.minimized)
    focusedId.value = target?.id ?? null
    enforceLimit(focusedId.value)
    return windows.value.length > 0
  }

  function closeAll() {
    windows.value = []
    focusedId.value = null
    snapPreview.value = null
    // Всплывающие слои чужого сеанса тоже не должны пережить смену пользователя.
    startOpen.value = false
    notifOpen.value = false
    holaOpen.value = false
    persist()
  }

  return {
    windows, focusedId, focused, hasWindows, limit, startOpen, startFull, notifOpen, holaOpen, snapPreview,
    area, screen, taskbarRect, bellCenter, fullscreen, taskbarPeek, zoneRect,
    byId, open, openApp, navigate, back, canGoBack,
    focus, close, minimize, restore, toggleFromTaskbar,
    maximize, unmaximize, toggleMaximize, snapTo, setRect, setPosition,
    setArea, setScreen, restoreSession, closeAll,
  }
})
