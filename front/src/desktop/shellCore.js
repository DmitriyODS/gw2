/**
 * Общее ядро каркасов-«ОС»: настольного (окна рабочего стола) и мобильного
 * (экраны разделов поверх стартового экрана). Оба открывают разделы одинаково —
 * через стор рабочего стола, — поэтому всё, что не про раскладку, живёт здесь:
 * права на раздел, обои, сводки живых плиток, синхронизация адресной строки с
 * активным разделом и чистка при выходе из системы.
 *
 * Раскладка (геометрия окон, панель задач, жесты) остаётся в самих каркасах —
 * она у них разная.
 */
import { computed, onBeforeUnmount, watch } from 'vue'
import { useRoute } from 'vue-router'
import router from '@/router/index.js'
import { useAuthStore } from '@/stores/auth.js'
import { useDesktopStore } from '@/stores/desktop.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { useLiveTilesStore } from '@/stores/liveTiles.js'
import { useActivityStore } from '@/stores/activity.js'
import { usePermission } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { clearNotificationJournal } from '@/composables/useDesktopNotifications.js'
import { appForPath, menuGroups } from '@/desktop/apps.js'
import { TASKBAR_MARGIN, shellActive, taskbarHeight, taskbarMargin } from '@/desktop/layout.js'
import { isBlankRecipe, normalizeRecipe } from '@/utils/chatBackgrounds.js'
import { defaultWallpaperRecipe } from '@/utils/wallpapers.js'

// Как часто освежаем сводки живых плиток.
const LIVE_PULSE = 60_000

/**
 * @param {object} opts
 * @param {() => string} opts.activePath — путь активного раздела для адресной
 *   строки ('/home', когда на переднем плане ничего нет).
 * @param {number} opts.barHeight — толщина панели задач этого каркаса.
 * @param {number} [opts.barMargin] — отступ панели от кромки экрана (0 —
 *   панель прижата вплотную, как на телефоне).
 * @param {number} [opts.limit] — предел одновременно открытых разделов.
 * @param {'replace'|'push'} [opts.navigate] — как каркас пишет свой адрес.
 *   Рабочий стол — replace (переключение окон историю браузера не засоряет);
 *   мобильный каркас — push, и тогда системная кнопка «назад» ходит по
 *   разделам сама, без перехвата жестов и popstate.
 * @param {() => void} [opts.onHome] — адрес стал «/home» (браузерное «назад»
 *   до стартового экрана).
 * @param {'desktop'|'mobile'} [opts.platform] — чья раскладка «Пуска» у этого
 *   каркаса (desktopPrefs хранит стол и мобилу раздельно).
 */
export function useShellCore({
  activePath,
  barHeight,
  barMargin = TASKBAR_MARGIN,
  limit = 0,
  navigate = 'replace',
  onHome = null,
  platform = 'desktop',
}) {
  const route = useRoute()
  const auth = useAuthStore()
  const desktop = useDesktopStore()
  const prefs = useDesktopPrefsStore()
  const live = useLiveTilesStore()
  const activity = useActivityStore()
  const { isSuperAdmin, hasActiveCompany } = usePermission()
  const { settings } = useCompanySettings()

  let livePulse = null

  function isAvailable(app) {
    return !!app && app.available({
      hasCompany: hasActiveCompany(),
      isSuperAdmin: isSuperAdmin(),
      settings: settings.value,
    })
  }

  // Обои: своя настройка пользователя, иначе — встроенный комплект. Пустой
  // рецепт (всё снято руками) оставляет каркас мягким волнам из токенов темы.
  const wallpaper = computed(() => {
    const recipe = normalizeRecipe(prefs.wallpaper) || defaultWallpaperRecipe()
    return isBlankRecipe(recipe) ? null : recipe
  })

  /* ── Живые плитки ───────────────────────────────────────────
     Сводки тянем заранее и освежаем по таймеру, поэтому стартовый экран
     открывается уже с данными. В скрытой вкладке не опрашиваем. */
  /* Опрашиваем только те плитки, которые реально показывают сводку: выключенная
     поимённо живая плитка не должна стоить ни одного запроса. */
  const liveAppIds = computed(() => menuGroups({
    hasCompany: hasActiveCompany(),
    isSuperAdmin: isSuperAdmin(),
    settings: settings.value,
  }, prefs.layout(platform))
    .flatMap((g) => g.items.map((a) => a.id))
    .filter((id) => prefs.isTileLive(id)))

  function pulseLiveTiles({ force = false } = {}) {
    if (document.hidden || !auth.user || !prefs.liveTiles) return
    live.refresh(liveAppIds.value, { force }).catch(() => {})
  }

  function onVisibility() {
    if (!document.hidden) pulseLiveTiles()
  }

  /* ── Синхронизация адреса и разделов ─────────────────────────
     Адресная строка отражает АКТИВНЫЙ раздел: deep-link и клик по системному
     уведомлению открывают нужный раздел, а переключение — обновляет URL.
     Обе стороны идемпотентны: совпадающий путь ничего не делает. */
  function openForPath(fullPath) {
    const resolved = router.resolve(fullPath)
    // У Hola своего окна нет: адрес /hola (ссылка, закладка) открывает панель
    // ассистента, а каркас возвращает себе адрес стартового экрана.
    if (resolved.path === '/hola') {
      desktop.holaOpen = true
      router.replace('/home').catch(() => {})
      return
    }
    if (resolved.path === '/home') {
      onHome?.()
      return
    }
    const app = appForPath(resolved.path)
    if (!app || !isAvailable(app)) return
    desktop.open(fullPath)
  }

  watch(() => route.fullPath, (fullPath) => {
    if (activePath() === fullPath) return
    openForPath(fullPath)
  })

  watch(activePath, (path) => {
    if (route.fullPath === path) return
    router[navigate](path).catch(() => {})
  })

  // Выход из системы — каркас чистим: разделы, сводки живых плиток и журнал
  // уведомлений следующего пользователя не должны наследоваться от прежнего.
  watch(() => auth.user, (user) => {
    if (user) {
      pulseLiveTiles({ force: true })
      return
    }
    desktop.closeAll()
    prefs.reset()
    live.reset()
    activity.reset()
    clearNotificationJournal()
  })

  // Сменилась активная компания — прежние сводки уже не про неё.
  watch(() => auth.companyId, () => { live.reset(); pulseLiveTiles({ force: true }) })

  /** Запуск каркаса: вызывать из onMounted ПОСЛЕ расчёта его геометрии. */
  function boot() {
    shellActive.value = true
    taskbarHeight.value = barHeight
    taskbarMargin.value = barMargin
    desktop.limit = limit
    // Личные настройки (закреплённые разделы, размеры плиток, обои) приезжают
    // с сервера — каркас одинаков на всех устройствах пользователя.
    prefs.load()
    // Сессия переживает перезагрузку; поверх неё открывается раздел из адреса
    // (deep-link важнее сохранённого состояния).
    desktop.restoreSession(isAvailable)
    openForPath(route.fullPath)

    pulseLiveTiles()
    livePulse = setInterval(() => pulseLiveTiles({ force: true }), LIVE_PULSE)
    document.addEventListener('visibilitychange', onVisibility)
  }

  onBeforeUnmount(() => {
    clearInterval(livePulse)
    document.removeEventListener('visibilitychange', onVisibility)
    shellActive.value = false
  })

  return { isAvailable, wallpaper, pulseLiveTiles, openForPath, boot }
}
