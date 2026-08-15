<template>
  <!-- Панель задач мобильного каркаса: «Пуск» слева, открытые и закреплённые
       разделы дальше. Юнит и уведомления живут в панели статусов — она видна
       и на стартовом экране; Hola вызывается свайпом от правой кромки, своей
       кнопки у неё нет (лента разделов забирала место у самой себя). -->
  <footer ref="barEl" class="mbar">
    <button
      class="mb-start"
      type="button"
      :class="{ active: startActive }"
      title="Пуск"
      aria-label="Пуск"
      @click="toggleStart"
    >
      <Logo :size="32" />
    </button>

    <div v-if="buttons.length" class="mb-apps">
      <button
        v-for="item in buttons"
        :key="item.key"
        class="mb-app"
        type="button"
        :class="{ active: isActive(item), shortcut: !item.win }"
        :title="item.title"
        :aria-label="item.title"
        @click="onClick(item)"
        @pointerdown="longPress.start(item, $event)"
        @pointermove="longPress.move($event)"
        @pointerup="longPress.cancel()"
        @pointercancel="longPress.cancel()"
        @contextmenu.prevent="openMenu(item, $event)"
      >
        <span class="material-symbols-outlined">{{ item.icon }}</span>
      </button>
    </div>

    <ContextMenu
      :visible="menu.open"
      :x="menu.x"
      :y="menu.y"
      :items="menuItems"
      @select="onMenuSelect"
      @close="menu.open = false"
    />
  </footer>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import router from '@/router/index.js'
import { useDesktopStore } from '@/stores/desktop.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { useLongPress } from '@/composables/useLongPress.js'
import { usePermission } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { appById, windowTitle } from '@/desktop/apps.js'
import Logo from '@/components/common/Logo.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'

const props = defineProps({
  /* Раскладка панели у телефона и планшета своя (desktopPrefs держит обе
     отдельно от стола) — какая именно, знает каркас. В шаблоне пропс доступен
     по имени, в скрипте — через props. */
  platform: { type: String, default: 'mobile' },
})

const desktop = useDesktopStore()
const prefs = useDesktopPrefsStore()
const { isSuperAdmin, hasActiveCompany } = usePermission()
const { settings } = useCompanySettings()

// Стартовый экран — на переднем плане, если его открыли или все разделы закрыты.
const startActive = computed(() => desktop.startOpen || !desktop.focused)

function isAvailable(app) {
  return !!app && app.available({
    hasCompany: hasActiveCompany(),
    isSuperAdmin: isSuperAdmin(),
    settings: settings.value,
  })
}

/* ── Кнопки панели: закреплённые разделы и открытые экраны ─────
   Закреплённый раздел идёт первым и превращается в кнопку экрана, как только
   его открыли; закрепления — свои для мобилы, независимо от рабочего стола. */
const buttons = computed(() => {
  const out = []
  const shown = new Set()

  for (const id of prefs.pinnedList(props.platform)) {
    const app = appById(id)
    if (!isAvailable(app)) continue
    shown.add(id)
    const win = desktop.windows.find((w) => w.appId === id)
    out.push(win
      ? { key: win.id, appId: id, icon: app.icon, title: titleOf(win), win }
      : { key: `pin-${id}`, appId: id, icon: app.icon, title: app.title, win: null })
  }

  for (const w of desktop.windows) {
    if (shown.has(w.appId)) continue
    out.push({ key: w.id, appId: w.appId, icon: appById(w.appId)?.icon || 'web_asset', title: titleOf(w), win: w })
  }
  return out
})

function titleOf(w) {
  return windowTitle(appById(w.appId), router.resolve(w.path))
}

function isActive(item) {
  return !!item.win && desktop.focusedId === item.win.id && !startActive.value
}

function toggleStart() {
  // Кнопка «Пуск» работает как качели: со стартового экрана возвращает к
  // разделу, на котором остановились.
  if (startActive.value && desktop.focused) desktop.focus(desktop.focusedId)
  else desktop.startOpen = true
}

function onClick(item) {
  if (longPress.consumed()) return
  if (!item.win) return void desktop.open(appById(item.appId).path)
  // Повторный тап по активному разделу — на стартовый экран (как «Пуск»).
  if (isActive(item)) desktop.startOpen = true
  else desktop.focus(item.win.id)
}

/* ── Контекстное меню ──────────────────────────────────────── */
const menu = reactive({ open: false, x: 0, y: 0, key: null })
const longPress = useLongPress((item, e) => openMenu(item, e))

const current = computed(() => buttons.value.find((b) => b.key === menu.key) || null)

const menuItems = computed(() => {
  const item = current.value
  if (!item) return []
  const pinItem = prefs.isPinned(props.platform, item.appId)
    ? { label: 'Открепить от панели', icon: 'keep_off', action: 'unpin' }
    : { label: 'Закрепить на панели', icon: 'keep', action: 'pin' }
  if (!item.win) return [{ label: 'Открыть', icon: 'open_in_new', action: 'open' }, { divider: true }, pinItem]
  return [
    { label: 'Перейти', icon: 'open_in_new', action: 'focus' },
    pinItem,
    { divider: true },
    { label: 'Закрыть раздел', icon: 'close', action: 'close', danger: true },
  ]
})

function openMenu(item, e) {
  menu.key = item.key
  menu.x = e.clientX
  menu.y = e.clientY
  menu.open = true
}

function onMenuSelect(action) {
  const item = current.value
  if (!item) return
  if (action === 'pin') return prefs.pin(props.platform, item.appId)
  if (action === 'unpin') return prefs.unpin(props.platform, item.appId)
  if (action === 'open') return void desktop.open(appById(item.appId).path)
  if (!item.win) return
  if (action === 'focus') desktop.focus(item.win.id)
  else if (action === 'close') desktop.close(item.win.id)
}

/* ── Геометрия панели ─────────────────────────────────────────
   Панель прижата к нижней кромке во всю ширину экрана; её размер публикуем в
   стор — по нему выравниваются всплывающие панели. */
const barEl = ref(null)
let observer = null

function syncRect() {
  const r = barEl.value?.getBoundingClientRect()
  if (!r) return
  Object.assign(desktop.taskbarRect, {
    x: Math.round(r.left),
    y: Math.round(r.top),
    w: Math.round(r.width),
    h: Math.round(r.height),
  })
}

onMounted(() => {
  syncRect()
  observer = new ResizeObserver(syncRect)
  if (barEl.value) observer.observe(barEl.value)
  window.addEventListener('resize', syncRect, { passive: true })
})

onBeforeUnmount(() => {
  observer?.disconnect()
  window.removeEventListener('resize', syncRect)
})
</script>

<style scoped>
/* Панель прижата к нижней кромке во всю ширину: без полей, скруглений и тени —
   раздел примыкает к ней вплотную. Системный жест «домой» отбирает свою полосу
   отступом, а не высотой содержимого. */
.mbar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  height: calc(var(--taskbar-height) + env(safe-area-inset-bottom, 0px));
  z-index: 900;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 8px env(safe-area-inset-bottom, 0px);
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border-top: 1px solid var(--acrylic-border);
}

.mbar button {
  -webkit-tap-highlight-color: transparent;
  border: none;
  background: none;
  cursor: pointer;
}

.mb-start {
  flex-shrink: 0;
  display: grid;
  place-items: center;
  width: 40px;
  min-width: 40px;
  height: 40px;
  min-height: 40px;
  padding: 0;
  border-radius: 50%;
  transition: background 0.15s ease, scale 0.12s ease;
}

.mb-start:active { scale: 0.92; }
.mb-start.active { background: color-mix(in oklch, var(--color-primary) 16%, transparent); }

/* Открытые разделы: лента значков с горизонтальной прокруткой — их может быть
   больше, чем влезает в ширину телефона. */
.mb-apps {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  overflow-x: auto;
  scrollbar-width: none;
  scroll-snap-type: x proximity;
}

.mb-apps::-webkit-scrollbar { display: none; }

.mb-app {
  position: relative;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  width: 36px;
  min-width: 36px;
  height: 36px;
  min-height: 36px;
  padding: 0;
  border-radius: var(--radius-md);
  color: var(--color-text);
  scroll-snap-align: center;
  transition: background 0.15s ease, color 0.15s ease, scale 0.12s ease;
}

.mb-app .material-symbols-outlined { font-size: 22px; }
.mb-app:active { scale: 0.9; }
.mb-app.shortcut { color: var(--color-text-dim); }

.mb-app.active {
  background: color-mix(in oklch, var(--color-primary) 18%, transparent);
  color: var(--color-primary);
}

/* Открытый, но не активный раздел помечен точкой — как в доке настольной ОС. */
.mb-app:not(.shortcut)::after {
  content: '';
  position: absolute;
  bottom: 2px;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: color-mix(in oklch, var(--color-text) 45%, transparent);
}

.mb-app.active::after { background: var(--color-primary); }

@media (prefers-reduced-motion: reduce) {
  .mbar button { transition: none; }
}
</style>
