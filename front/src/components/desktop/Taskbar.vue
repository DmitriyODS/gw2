<template>
  <footer
    ref="barEl"
    class="taskbar"
    :class="[
      `side-${side}`,
      { compact: touch, hidden: !touch && ((desktop.fullscreen && !desktop.taskbarPeek) || desktop.startFull) },
    ]"
  >
    <button
      class="tb-start"
      type="button"
      :class="{ active: desktop.startOpen }"
      title="Пуск"
      aria-label="Пуск"
      @click="desktop.startOpen = !desktop.startOpen"
    >
      <Logo :size="42" />
    </button>

    <button
      class="tb-hola"
      type="button"
      :class="{ active: desktop.holaOpen }"
      title="Hola ассистент — поиск, команды и чат (Ctrl+K)"
      aria-label="Hola ассистент"
      @click="openHola"
    >
      <HolaIcon :size="21" />
    </button>

    <span class="tb-sep" aria-hidden="true" />

    <div v-if="buttons.length" class="tb-windows">
      <button
        v-for="item in buttons"
        :key="item.key"
        class="tb-win"
        type="button"
        :class="{
          active: isOnScreen(item),
          minimized: !touch && item.win?.minimized,
          shortcut: !item.win,
          dragging: drag.appId === item.appId,
        }"
        :title="item.title"
        :draggable="!touch && prefs.isPinned(platform, item.appId)"
        @click="onButtonClick(item)"
        @contextmenu.prevent="openMenu(item, $event)"
        @pointerdown="onDown(item, $event)"
        @pointermove="longPress.move($event)"
        @pointerup="onUp(item, $event)"
        @pointercancel="longPress.cancel()"
        @dragstart="onDragStart(item, $event)"
        @dragover.prevent="onDragOver(item)"
        @drop.prevent="onDragEnd"
        @dragend="onDragEnd"
      >
        <span class="material-symbols-outlined tb-win-icon">{{ item.icon }}</span>
        <span class="tb-win-label">{{ item.title }}</span>
      </button>
    </div>

    <span v-if="buttons.length" class="tb-sep tb-sep-right" aria-hidden="true" />

    <div class="tb-right">
      <button v-if="unit" class="tb-unit" type="button" title="Идёт работа — открыть юнит" @click="expand">
        <span class="material-symbols-outlined">timer</span>
        <span class="tb-unit-clock">{{ clock }}</span>
      </button>

      <button
        class="tb-clock"
        type="button"
        :title="`${fullDate} — открыть календари`"
        @click="openCalendars"
      >
        <span class="tb-time">{{ time }}</span>
        <span class="tb-date">{{ date }}</span>
      </button>

      <button
        ref="bellEl"
        class="tb-bell"
        type="button"
        :class="{ active: desktop.notifOpen, muted: notifyMuted }"
        :title="notifyMuted ? `Уведомления отключены ${muteUntilLabel}` : 'Уведомления'"
        aria-label="Уведомления"
        @click="toggleNotifications"
        @contextmenu.prevent="openBellMenu"
      >
        <span class="material-symbols-outlined">{{ notifyMuted ? 'notifications_off' : 'notifications' }}</span>
        <span v-if="alerts" class="tb-bell-dot">{{ alerts > 99 ? '99+' : alerts }}</span>
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
import { useUnitsStore } from '@/stores/units.js'
import { useActiveUnit } from '@/composables/useActiveUnit.js'
import { useDesktopNotifications } from '@/composables/useDesktopNotifications.js'
import { useNotifyMute } from '@/composables/useNotifyMute.js'
import { useElapsed } from '@/composables/useElapsed.js'
import { useLongPress } from '@/composables/useLongPress.js'
import { usePermission } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { appById, windowTitle } from '@/desktop/apps.js'
import { TASKBAR_MARGIN } from '@/desktop/layout.js'
import Logo from '@/components/common/Logo.vue'
import HolaIcon from '@/components/common/HolaIcon.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'

const props = defineProps({
  /* Чья раскладка панели (desktopPrefs держит стол, планшет и мобилу
     раздельно). В шаблоне пропс доступен по имени, в скрипте — через props. */
  platform: { type: String, default: 'desktop' },
  /* Сенсорный режим (планшет): меню открывает долгое нажатие, перетаскивания
     кнопок нет (тач его не понимает), панель компактнее и всегда снизу,
     а тап переключает раздел, а не сворачивает окно. */
  touch: { type: Boolean, default: false },
})

const desktop = useDesktopStore()
const prefs = useDesktopPrefsStore()

/* Сторона панели — личная настройка («Настройки → Рабочий стол»). От неё
   зависят и раскладка кнопок, и якоря всплывающих панелей. На планшете панель
   всегда снизу: вертикальная колонка значков пальцу неудобна, а места по бокам
   и так мало. */
const side = computed(() => (props.touch ? 'bottom' : prefs.taskbarSide))
const units = useUnitsStore()
const { expand } = useActiveUnit()
// Бейдж на кнопке = сколько карточек лежит в центре уведомлений (и убранные
// оттуда гаснут вместе с ним) — счёт один на оба места.
const { count: alerts } = useDesktopNotifications()
const { muted: notifyMuted, untilLabel: muteUntilLabel, mute, unmute } = useNotifyMute()
const { isSuperAdmin, hasActiveCompany } = usePermission()
const { settings } = useCompanySettings()

const unit = computed(() => units.activeUnit)
const { clock } = useElapsed(() => unit.value?.datetime_start)

function isAvailable(app) {
  return !!app && app.available({
    hasCompany: hasActiveCompany(),
    isSuperAdmin: isSuperAdmin(),
    settings: settings.value,
  })
}

/* ── Кнопки панели: закреплённые разделы и открытые окна ───────
   Закреплённый раздел идёт первым и превращается в кнопку окна, как только
   его открыли (как в доке настольной ОС). */
const buttons = computed(() => {
  const out = []
  const shown = new Set()

  for (const id of prefs.pinnedList(props.platform)) {
    const app = appById(id)
    if (!isAvailable(app)) continue
    shown.add(id)
    const wins = desktop.windows.filter((w) => w.appId === id)
    if (!wins.length) {
      out.push({ key: `pin-${id}`, appId: id, icon: app.icon, title: app.title, win: null })
      continue
    }
    wins.forEach((w) => out.push({ key: w.id, appId: id, icon: app.icon, title: titleOf(w), win: w }))
  }

  for (const w of desktop.windows) {
    if (shown.has(w.appId)) continue
    out.push({ key: w.id, appId: w.appId, icon: appById(w.appId)?.icon || 'web_asset', title: titleOf(w), win: w })
  }
  return out
})

/* Hola — постоянная кнопка рядом с «Пуском»: открывает всплывающую панель
   поиска, команд и чата (наследницу строки Spotlight). */
function openHola() {
  desktop.holaOpen = !desktop.holaOpen
}

function titleOf(w) {
  return windowTitle(appById(w.appId), router.resolve(w.path))
}

/* Подсвечен ровно ОДИН раздел — тот, что в фокусе. Вторая зона планшета видна
   одновременно с главной, но активная среди них всё равно одна: подсветка
   отвечает на вопрос «куда я сейчас печатаю», а не «что видно». */
function isOnScreen(item) {
  if (!item.win || desktop.focusedId !== item.win.id) return false
  return props.touch ? !desktop.startOpen : !item.win.minimized
}

function onButtonClick(item) {
  if (dragged) { dragged = false; return }
  if (longPress.consumed()) return
  if (!item.win) return void desktop.open(appById(item.appId).path)
  /* Сенсорный каркас: окна не сворачиваются — повторный тап по разделу, который
     уже на экране, уводит на стартовый экран (так же ведёт себя «Пуск»). */
  if (!props.touch) return desktop.toggleFromTaskbar(item.win.id)
  const onScreen = !desktop.startOpen
    && (desktop.focusedId === item.win.id || desktop.sideId === item.win.id)
  if (onScreen) desktop.startOpen = true
  else desktop.focus(item.win.id)
}

/* ── Сенсорные жесты кнопки ───────────────────────────────────
   Пальцем правой кнопки нет: меню открывает долгое нажатие. Потянуть кнопку
   вверх и отпустить в правой половине — поставить раздел второй зоной (левая
   половина просто открывает его). Порог отсекает обычный тап. */
const DRAG_UP = 48
const longPress = useLongPress((item, e) => openMenu(item, e))
let dragFrom = null
let dragged = false

function onDown(item, e) {
  if (!props.touch) return
  longPress.start(item, e)
  dragFrom = { x: e.clientX, y: e.clientY }
}

function onUp(item, e) {
  if (!props.touch) return
  longPress.cancel()
  const from = dragFrom
  dragFrom = null
  if (!from || from.y - e.clientY < DRAG_UP) return
  dragged = true
  const path = appById(item.appId)?.path
  if (e.clientX > window.innerWidth / 2) desktop.openSide(item.win?.id || path)
  else if (item.win) desktop.focus(item.win.id)
  else if (path) desktop.open(path)
}

/* ── Перетаскивание закреплённых кнопок ───────────────────────
   Меняется порядок в личных настройках, поэтому раскладка панели едет между
   устройствами. Кнопки незакреплённых окон не переставляются: их порядок
   задаёт очерёдность открытия. */
const drag = reactive({ appId: null })

function onDragStart(item, e) {
  if (!prefs.isPinned(props.platform, item.appId)) return
  drag.appId = item.appId
  e.dataTransfer.effectAllowed = 'move'
  // Firefox не начинает перетаскивание без данных в буфере.
  e.dataTransfer.setData('text/plain', item.appId)
}

function onDragOver(item) {
  if (!drag.appId || item.appId === drag.appId || !prefs.isPinned(props.platform, item.appId)) return
  const ids = [...prefs.pinnedList(props.platform)]
  const from = ids.indexOf(drag.appId)
  const to = ids.indexOf(item.appId)
  if (from < 0 || to < 0) return
  ids.splice(to, 0, ...ids.splice(from, 1))
  prefs.setPinnedOrder(props.platform, ids)
}

function onDragEnd() {
  drag.appId = null
}

function openCalendars() {
  const app = appById('calendars')
  if (isAvailable(app)) desktop.open(app.path)
}

/* ── Часы ──────────────────────────────────────────────────── */
const now = ref(new Date())
let timer = null
onMounted(() => { timer = setInterval(() => { now.value = new Date() }, 10000) })
onBeforeUnmount(() => clearInterval(timer))

const time = computed(() => now.value.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' }))
const date = computed(() => now.value.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: '2-digit' }))
const fullDate = computed(() => now.value.toLocaleDateString('ru-RU', {
  weekday: 'long', day: 'numeric', month: 'long', year: 'numeric',
}))

/* Центр панели уведомлений — центр кнопки, которая её открывает. */
const bellEl = ref(null)

function toggleNotifications() {
  const r = bellEl.value?.getBoundingClientRect()
  if (r) desktop.bellCenter = r.left + r.width / 2
  desktop.notifOpen = !desktop.notifOpen
}

/* ── Геометрия панели ──────────────────────────────────────────
   Панель центрирована и растёт по числу кнопок до предела — меню «Пуск» и
   центр уведомлений выравниваются по её краям, поэтому размер публикуем в
   стор рабочего стола. */
const barEl = ref(null)
let observer = null

function syncRect() {
  const el = barEl.value
  if (!el) return
  // Размеры, а не getBoundingClientRect: спрятанная панель сдвинута transform'ом,
  // и якоря панелей уехали бы вместе с ней. Положение считаем по стороне, к
  // которой панель прижата.
  const w = el.offsetWidth
  const h = el.offsetHeight
  const gap = TASKBAR_MARGIN
  const centerX = Math.round((window.innerWidth - w) / 2)
  const centerY = Math.round((window.innerHeight - h) / 2)
  const pos = {
    bottom: { x: centerX, y: window.innerHeight - h - gap },
    top: { x: centerX, y: gap },
    left: { x: gap, y: centerY },
    right: { x: window.innerWidth - w - gap, y: centerY },
  }[side.value] || { x: centerX, y: window.innerHeight - h - gap }
  Object.assign(desktop.taskbarRect, { ...pos, w, h })
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

/* ── Контекстное меню кнопки ───────────────────────────────── */
const menu = reactive({ open: false, x: 0, y: 0, key: null, kind: 'button' })

const current = computed(() => buttons.value.find((b) => b.key === menu.key) || null)

// Сроки тишины: от «на десять минут отойти» до «сегодня меня нет».
const MUTE_OPTIONS = [
  { label: '10 минут', minutes: 10 },
  { label: '30 минут', minutes: 30 },
  { label: '1 час', minutes: 60 },
  { label: '4 часа', minutes: 240 },
  { label: '8 часов', minutes: 480 },
  { label: 'Навсегда', minutes: null },
]

const bellMenuItems = computed(() => {
  if (notifyMuted.value) {
    return [{ label: `Включить уведомления (${muteUntilLabel.value})`, icon: 'notifications_active', action: 'unmute' }]
  }
  return [{
    label: 'Отключить уведомления',
    icon: 'notifications_off',
    children: MUTE_OPTIONS.map((o) => ({
      label: o.label,
      icon: o.minutes ? 'schedule' : 'do_not_disturb_on',
      action: `mute:${o.minutes ?? 'forever'}`,
      danger: !o.minutes,
    })),
  }]
})

const menuItems = computed(() => {
  if (menu.kind === 'bell') return bellMenuItems.value
  const item = current.value
  if (!item) return []
  const pinItem = prefs.isPinned(props.platform, item.appId)
    ? { label: 'Открепить от панели задач', icon: 'keep_off', action: 'unpin' }
    : { label: 'Закрепить на панели задач', icon: 'keep', action: 'pin' }

  /* Сенсорный каркас: окон нет — есть две зоны, поэтому вместо «свернуть /
     развернуть / ещё одно окно» предлагаем поставить раздел рядом. */
  if (props.touch) {
    const sideRow = item.win && desktop.sideId === item.win.id
      ? { label: 'Убрать вторую зону', icon: 'close_fullscreen', action: 'unside' }
      : { label: 'Открыть рядом', icon: 'splitscreen_right', action: 'side' }
    if (!item.win) {
      return [
        { label: 'Открыть', icon: 'open_in_new', action: 'open' },
        sideRow,
        { divider: true },
        pinItem,
      ]
    }
    return [
      { label: 'Перейти', icon: 'open_in_new', action: 'focus' },
      sideRow,
      pinItem,
      { divider: true },
      { label: 'Закрыть раздел', icon: 'close', action: 'close', danger: true },
    ]
  }

  if (!item.win) {
    return [
      { label: 'Открыть', icon: 'open_in_new', action: 'open' },
      { divider: true },
      pinItem,
    ]
  }
  return [
    { label: 'Открыть ещё одно окно', icon: 'add', action: 'new' },
    pinItem,
    { divider: true },
    item.win.minimized
      ? { label: 'Восстановить', icon: 'open_in_full', action: 'restore' }
      : { label: 'Свернуть', icon: 'remove', action: 'minimize' },
    { label: item.win.mode === 'normal' ? 'Развернуть' : 'Вернуть размер', icon: 'crop_square', action: 'max' },
    { divider: true },
    { label: 'Закрыть', icon: 'close', action: 'close', danger: true },
  ]
})

function openMenu(item, e) {
  menu.kind = 'button'
  menu.key = item.key
  menu.x = e.clientX
  menu.y = e.clientY
  menu.open = true
}

/* ПКМ по колокольчику — «не беспокоить»: тишина на срок или насовсем.
   Уведомления при этом продолжают копиться в центре — гаснут только звук и
   всплывашки ОС. */
function openBellMenu(e) {
  menu.kind = 'bell'
  menu.key = null
  menu.x = e.clientX
  menu.y = e.clientY
  menu.open = true
}

function onMenuSelect(action) {
  if (menu.kind === 'bell') {
    if (action === 'unmute') return unmute()
    if (action?.startsWith('mute:')) {
      const arg = action.slice(5)
      return mute(arg === 'forever' ? null : Number(arg))
    }
    return
  }
  const item = current.value
  if (!item) return
  if (action === 'pin') return prefs.pin(props.platform, item.appId)
  if (action === 'unpin') return prefs.unpin(props.platform, item.appId)
  if (action === 'open') return void desktop.open(appById(item.appId).path)
  if (action === 'side') return void desktop.openSide(item.win?.id || appById(item.appId).path)
  if (action === 'unside') return desktop.closeSide()
  const win = item.win
  if (!win) return
  if (action === 'focus') desktop.focus(win.id)
  else if (action === 'new') desktop.open(win.path, { newWindow: true })
  else if (action === 'minimize') desktop.minimize(win.id)
  else if (action === 'restore') { desktop.restore(win.id); desktop.focus(win.id) }
  else if (action === 'max') desktop.toggleMaximize(win.id)
  else if (action === 'close') desktop.close(win.id)
}
</script>

<style scoped>
/* Плавающая акриловая панель задач по центру экрана: ширина по содержимому и
   растёт с числом кнопок до предела, дальше кнопки прокручиваются внутри. */
.taskbar {
  position: fixed;
  left: 50%;
  transform: translateX(-50%);
  bottom: 12px;
  height: var(--taskbar-height);
  width: max-content;
  max-width: min(1400px, calc(100vw - 48px));
  z-index: 900;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 10px;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-md);
  transition: transform 0.22s cubic-bezier(0.2, 0, 0, 1), opacity 0.18s ease;
}

/* Панель у верхней кромки — то же самое, только сверху. */
.taskbar.side-top {
  top: 12px;
  bottom: auto;
}

/* Вертикальные стороны: колонка кнопок вдоль края, ширина — та же толщина. */
.taskbar.side-left,
.taskbar.side-right {
  top: 50%;
  bottom: auto;
  left: auto;
  transform: translateY(-50%);
  flex-direction: column;
  width: var(--taskbar-height);
  height: max-content;
  max-width: none;
  max-height: min(1400px, calc(100dvh - 48px));
  padding: 10px 0;
  /* Прокручивается только список окон внутри — иначе уезжали бы и часы. */
  overflow: hidden;
}

.taskbar.side-left { left: 12px; }
.taskbar.side-right { right: 12px; }

/* Вертикальная раскладка: все зоны идут колонкой, разделители ложатся
   поперёк, а подписи окон уступают место значкам — на 68 пикселях ширины
   текст всё равно не читается. */
.taskbar.side-left .tb-sep,
.taskbar.side-right .tb-sep {
  width: 28px;
  height: 1px;
}

.taskbar.side-left .tb-windows,
.taskbar.side-right .tb-windows {
  flex-direction: column;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 0 2px;
}

.taskbar.side-left .tb-win,
.taskbar.side-right .tb-win {
  width: 48px;
  max-width: 48px;
  padding: 0;
  justify-content: center;
}

.taskbar.side-left .tb-win-label,
.taskbar.side-right .tb-win-label { display: none; }

.taskbar.side-left .tb-right,
.taskbar.side-right .tb-right {
  flex-direction: column;
  margin-left: 0;
  margin-top: auto;
}

.taskbar.side-left .tb-unit,
.taskbar.side-right .tb-unit,
.taskbar.side-left .tb-clock,
.taskbar.side-right .tb-clock {
  width: 48px;
  padding: 4px 0;
  height: auto;
}

/* Часы в колонке: время и дата в две строки, шрифт мельче — иначе дата не
   влезает в ширину панели. */
.taskbar.side-left .tb-time,
.taskbar.side-right .tb-time { font-size: 0.82rem; }

.taskbar.side-left .tb-date,
.taskbar.side-right .tb-date { font-size: 0.6rem; letter-spacing: -0.01em; }

.taskbar.side-left .tb-unit-clock,
.taskbar.side-right .tb-unit-clock { font-size: 0.7rem; }

/* Полноэкранное окно занимает весь экран — панель уезжает за свой край и
   возвращается по подведению указателя к нему. */
.taskbar.hidden {
  transform: translate(-50%, calc(100% + 24px));
  opacity: 0;
  pointer-events: none;
}

.taskbar.side-top.hidden { transform: translate(-50%, calc(-100% - 24px)); }
.taskbar.side-left.hidden { transform: translate(calc(-100% - 24px), -50%); }
.taskbar.side-right.hidden { transform: translate(calc(100% + 24px), -50%); }

/* Hola — рядом с «Пуском»: он тоже про «начать отсюда». Только значок. */
.tb-hola {
  display: grid;
  place-items: center;
  width: 48px;
  min-width: 48px;
  max-width: 48px;
  height: 48px;
  min-height: 48px;
  max-height: 48px;
  flex-shrink: 0;
  padding: 0;
  border: none;
  border-radius: var(--radius-lg);
  /* Матовое стекло без блика, тени и обводки: панель задач сама backdrop root,
     поэтому подложка кнопок — плотный акрил, а не второй backdrop-filter. */
  background: var(--acrylic-card-bg);
  /* Тот же тон, что у кнопок окон и колокольчика: приглушённый вариант в
     тёмной теме читался как «погасшая» иконка. */
  color: var(--color-text);
  font-size: 13.5px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.tb-hola:hover,
.tb-hola.active {
  background: color-mix(in oklch, var(--color-primary) 14%, var(--acrylic-card-bg));
  color: var(--color-primary);
}

/* Тонкие разделители зон: «Пуск» | разделы | часы с уведомлениями. */
.tb-sep {
  width: 1px;
  height: 28px;
  flex-shrink: 0;
  background: var(--acrylic-border);
}

/* ── Пуск ── */
.tb-start {
  width: 48px;
  min-width: 48px;
  max-width: 48px;
  height: 48px;
  min-height: 48px;
  max-height: 48px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  padding: 0;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  cursor: pointer;
  transition: background 0.15s;
}

.tb-start:hover { background: color-mix(in oklch, var(--color-primary) 12%, transparent); }
.tb-start.active { background: var(--grad-primary-soft); }

/* ── Кнопки закреплённых разделов и открытых окон ──
   flex: 0 1 auto + min-width: 0 — блок не растягивает панель сверх контента,
   но сжимается и прокручивается, когда кнопок больше, чем помещается. */
.tb-windows {
  flex: 0 1 auto;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  scrollbar-width: none;
  padding: 2px 0;
}
.tb-windows::-webkit-scrollbar { height: 0; }

/* Кнопка окна лежит на такой же акриловой подложке, что и сама панель, и без
   кромки сливалась с ней — границу задаём явно. */
.tb-win {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 48px;
  max-width: 190px;
  flex-shrink: 0;
  padding: 0 14px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font-size: 13.5px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s, opacity 0.15s;
}

.tb-win:hover {
  background: color-mix(in oklch, var(--color-primary) 12%, var(--acrylic-card-bg));
  border-color: color-mix(in oklch, var(--color-primary) 35%, var(--acrylic-border));
}

/* Активное окно — заметно ярче: это единственная кнопка, у которой раздел
   сейчас на экране. */
.tb-win.active {
  background: var(--grad-primary-soft);
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.tb-win.minimized { opacity: 0.6; }

/* Перетаскиваемая кнопка приглушена: её место уже показывают разъехавшиеся
   соседи. */
.tb-win.dragging { opacity: 0.45; }

/* Закреплённый, но не открытый раздел — только иконка (ярлык). */
.tb-win.shortcut {
  width: 48px;
  min-width: 48px;
  max-width: 48px;
  padding: 0;
  justify-content: center;
}
.tb-win.shortcut .tb-win-label { display: none; }

.tb-win-icon { font-size: 19px; flex-shrink: 0; }

.tb-win-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ── Правый блок: юнит, часы, уведомления ── */
.tb-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  margin-left: auto;
}

.tb-unit {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 48px;
  padding: 0 16px;
  border: none;
  border-radius: var(--radius-lg);
  background: color-mix(in oklch, var(--color-tertiary) 18%, transparent);
  color: var(--color-tertiary);
  font-size: 14px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition: background 0.15s;
}

.tb-unit:hover { background: color-mix(in oklch, var(--color-tertiary) 26%, transparent); }
.tb-unit .material-symbols-outlined { font-size: 20px; }

.tb-clock {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  line-height: 1.15;
  height: 48px;
  padding: 0 14px;
  border: none;
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition: background 0.15s;
}

.tb-clock:hover { background: color-mix(in oklch, var(--color-primary) 12%, var(--acrylic-card-bg)); }

.tb-time { font-size: 15px; font-weight: 800; color: var(--color-text); }
.tb-date { font-size: 11.5px; color: var(--color-text-dim); }

.tb-bell {
  position: relative;
  width: 48px;
  min-width: 48px;
  max-width: 48px;
  height: 48px;
  min-height: 48px;
  max-height: 48px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.tb-bell:hover,
.tb-bell.active {
  background: color-mix(in oklch, var(--color-primary) 14%, var(--acrylic-card-bg));
  color: var(--color-primary);
}

/* «Не беспокоить»: колокольчик приглушён, но бейдж остаётся — уведомления
   копятся, их просто не слышно. */
.tb-bell.muted { color: var(--color-text-dim); }

.tb-bell-dot {
  position: absolute;
  top: -5px;
  right: -5px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-full);
  background: var(--color-error);
  color: var(--color-on-error);
  font-size: 10.5px;
  font-weight: 800;
}

.tb-bell.muted .tb-bell-dot { background: var(--color-text-dim); }

/* ── Компактный сенсорный вид (планшет) ──
   Та же панель, что на столе, только ниже и без подписей у кнопок разделов:
   на планшете её видно всё время, и каждый лишний пиксель высоты — минус
   разделу. Цели остаются пальцевыми (не меньше 40px), панель прижата к нижней
   кромке во всю ширину — свободные поля по бокам ей ни к чему. */
.taskbar.compact {
  left: 0;
  right: 0;
  bottom: 0;
  width: auto;
  max-width: none;
  transform: none;
  height: calc(var(--taskbar-height) + env(safe-area-inset-bottom, 0px));
  padding: 0 10px env(safe-area-inset-bottom, 0px);
  gap: 8px;
  border: none;
  border-radius: 0;
  border-top: 1px solid var(--acrylic-border);
}

.taskbar.compact .tb-start,
.taskbar.compact .tb-hola,
.taskbar.compact .tb-bell {
  width: 40px;
  min-width: 40px;
  max-width: 40px;
  height: 40px;
  min-height: 40px;
  max-height: 40px;
}

.taskbar.compact .tb-start :deep(svg) { width: 34px; height: 34px; }

/* Кнопки разделов — только значки: подпись съедала бы место, а разделов на
   планшете открыто больше. */
.taskbar.compact .tb-win {
  width: 44px;
  min-width: 44px;
  max-width: 44px;
  height: 40px;
  padding: 0;
  justify-content: center;
}

.taskbar.compact .tb-win-label { display: none; }
.taskbar.compact .tb-sep { height: 22px; }

/* Настольная панель центрирована и шириной по содержимому, поэтому правая
   зона там сама оказывается у края. Компактная растянута во всю ширину —
   свободное место надо отдать явно, иначе часы с уведомлениями и разделитель
   перед ними жмутся к разделам где-то посередине.

   Забирает место РОВНО ОДИН элемент: два `margin-left: auto` поделили бы
   свободную ширину поровну и увели разделитель на середину. Разделителя может
   не быть вовсе (разделы не открыты) — тогда место забирает правая зона. */
.taskbar.compact .tb-sep-right { margin-left: auto; }
.taskbar.compact .tb-right { margin-left: auto; }
.taskbar.compact .tb-sep-right ~ .tb-right { margin-left: 0; }
.taskbar.compact .tb-clock { padding: 0 8px; }
.taskbar.compact .tb-date { display: none; }
.taskbar.compact .tb-right { gap: 6px; }
</style>
