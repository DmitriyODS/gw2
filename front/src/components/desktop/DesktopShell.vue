<template>
  <div
    class="desktop"
    :data-taskbar="prefs.taskbarSide"
    :style="{ '--taskbar-height': `${TASKBAR_HEIGHT}px` }"
    @contextmenu.self.prevent="openDeskMenu"
  >
    <!-- Обои: личная картинка/градиент пользователя либо мягкие волны из
         цветов темы. Тот же слой, что фон чатов и ленты портала. -->
    <ChatBackgroundLayer v-if="wallpaper" :recipe="wallpaper" class="desk-paper" />
    <div v-else class="desk-wallpaper" aria-hidden="true" />

    <!-- Слой окон. Порядок в массиве не меняется — перекрытие задаёт z-index,
         иначе перефокусировка перемонтировала бы содержимое разделов. -->
    <div class="desk-windows">
      <AppWindow v-for="win in desktop.windows" :key="win.id" :win="win" />
    </div>

    <!-- Подсветка зоны прилипания во время перетаскивания окна. -->
    <div
      v-if="desktop.snapPreview"
      class="desk-snap"
      :style="{
        transform: `translate3d(${desktop.snapPreview.rect.x}px, ${desktop.snapPreview.rect.y}px, 0)`,
        width: `${desktop.snapPreview.rect.w}px`,
        height: `${desktop.snapPreview.rect.h}px`,
      }"
      aria-hidden="true"
    />

    <!-- Указатель у верхней кромки экрана — выезжает кнопка полноэкранного
         режима браузера (как автоскрывающаяся панель настольной ОС). -->
    <Transition name="desk-full">
      <button
        v-if="fullBtnShown"
        class="desk-full"
        type="button"
        :title="pageFull ? 'Выйти из полноэкранного режима (F11)' : 'Развернуть браузер на весь экран (F11)'"
        @click="toggleFull"
      >
        <span class="material-symbols-outlined">{{ pageFull ? 'fullscreen_exit' : 'fullscreen' }}</span>
        {{ pageFull ? 'Свернуть окно' : 'Во весь экран' }}
      </button>
    </Transition>

    <Taskbar />
    <Transition name="sm">
      <StartMenu v-if="desktop.startOpen" />
    </Transition>
    <Transition name="np">
      <NotificationsPanel v-if="desktop.notifOpen" @close="desktop.notifOpen = false" />
    </Transition>
    <Transition name="hp">
      <HolaPopup v-if="desktop.holaOpen" @close="desktop.holaOpen = false" />
    </Transition>

    <ContextMenu
      :visible="deskMenu.open"
      :x="deskMenu.x"
      :y="deskMenu.y"
      :items="deskMenuItems"
      @select="onDeskMenuSelect"
      @close="deskMenu.open = false"
    />
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useDesktopStore } from '@/stores/desktop.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { useShellCore } from '@/desktop/shellCore.js'
import { TASKBAR_HEIGHT, areaInsets, taskbarReserve, taskbarSide } from '@/desktop/layout.js'
import {
  isPageFullscreen, onPageFullscreenChange, pageFullscreenSupported, togglePageFullscreen,
} from '@/utils/pageFullscreen.js'
import ContextMenu from '@/components/common/ContextMenu.vue'
import ChatBackgroundLayer from '@/components/common/ChatBackgroundLayer.vue'
import AppWindow from './AppWindow.vue'
import Taskbar from './Taskbar.vue'
import StartMenu from './StartMenu.vue'
import NotificationsPanel from './NotificationsPanel.vue'
import HolaPopup from './HolaPopup.vue'

const desktop = useDesktopStore()
const prefs = useDesktopPrefsStore()

/* Права на разделы, обои, живые плитки и синхронизация адреса — общие с
   мобильным каркасом (см. shellCore); здесь остаётся только раскладка. */
const { wallpaper, boot } = useShellCore({
  activePath: () => {
    const win = desktop.focused
    return win && !win.minimized ? win.path : '/home'
  },
  barHeight: TASKBAR_HEIGHT,
})

// Мобильная обёртка и старые браузеры без Fullscreen API кнопку не показывают.
const fullscreenSupported = pageFullscreenSupported()

/* ── Рабочая область: экран минус панель задач ───────────────
   К трём свободным краям окна прилегают вплотную; со стороны панели задач
   остаётся ровно место под неё (сторону задаёт пользователь в настройках). */
function syncArea() {
  desktop.setScreen({ x: 0, y: 0, w: window.innerWidth, h: window.innerHeight })
  const inset = areaInsets(prefs.taskbarSide)
  desktop.setArea({
    x: inset.left,
    y: inset.top,
    w: Math.max(320, window.innerWidth - inset.left - inset.right),
    h: Math.max(240, window.innerHeight - inset.top - inset.bottom),
  })
}

watch(() => desktop.fullscreen, (on) => { if (!on) desktop.taskbarPeek = false })

// Смена стороны панели меняет рабочую область: окна пересчитываются, а
// плавающий питомец узнаёт о ней через общий модуль геометрии.
watch(() => prefs.taskbarSide, (side) => {
  taskbarSide.value = side
  syncArea()
}, { immediate: true })

/* ── Контекстное меню пустого стола ────────────────────────── */
const deskMenu = reactive({ open: false, x: 0, y: 0 })

const deskMenuItems = computed(() => [
  { label: 'Свернуть все окна', icon: 'minimize', action: 'minimize-all' },
  { label: 'Восстановить окна', icon: 'open_in_full', action: 'restore-all' },
  { divider: true },
  { label: 'Персонализация', icon: 'wallpaper', action: 'personalize' },
  { divider: true },
  { label: 'Закрыть все окна', icon: 'close', action: 'close-all', danger: true },
])

function openDeskMenu(e) {
  deskMenu.x = e.clientX
  deskMenu.y = e.clientY
  deskMenu.open = true
}

function onDeskMenuSelect(action) {
  if (action === 'minimize-all') desktop.windows.forEach((w) => desktop.minimize(w.id))
  else if (action === 'restore-all') desktop.windows.forEach((w) => desktop.restore(w.id))
  else if (action === 'personalize') desktop.open('/settings?section=desktop')
  else if (action === 'close-all') desktop.closeAll()
}

/* Панель задач в полноэкранном режиме и кнопка «во весь экран» у верхней
   кромки — обе живут автоскрытием по краю экрана, как в настольных ОС. */
function onPointerMove(e) {
  trackTaskbarPeek(e)
  trackFullButton(e)
}

function trackTaskbarPeek(e) {
  if (!desktop.fullscreen) return
  // Расстояние до того края, к которому прижата панель.
  const dist = {
    bottom: window.innerHeight - e.clientY,
    top: e.clientY,
    left: e.clientX,
    right: window.innerWidth - e.clientX,
  }[prefs.taskbarSide] ?? (window.innerHeight - e.clientY)
  if (dist <= 4) desktop.taskbarPeek = true
  else if (desktop.taskbarPeek && dist > taskbarReserve() + 24) desktop.taskbarPeek = false
}

/* Кнопка полноэкранного режима: появляется, когда указатель касается верхней
   кромки по центру, и прячется, когда он ушёл из её зоны. Зона показа узкая
   (у самого края), зона удержания — шире самой кнопки, иначе до неё было бы
   не дотянуться мышью. */
const FULL_ZONE_X = 170
const FULL_HOLD_X = 260
const FULL_HOLD_Y = 68

const fullBtnShown = ref(false)
const pageFull = ref(isPageFullscreen())
let stopFullscreenWatch = null

function trackFullButton(e) {
  if (!fullscreenSupported) return
  const dx = Math.abs(e.clientX - window.innerWidth / 2)
  if (e.clientY <= 4 && dx <= FULL_ZONE_X) fullBtnShown.value = true
  else if (fullBtnShown.value && (e.clientY > FULL_HOLD_Y || dx > FULL_HOLD_X)) fullBtnShown.value = false
}

function toggleFull() {
  togglePageFullscreen()
  fullBtnShown.value = false
}

function onKeydown(e) {
  // Ctrl/Cmd+K — всплывающая панель Hola: поиск, команды и чат с ассистентом.
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    desktop.holaOpen = !desktop.holaOpen
    return
  }
  if (e.key !== 'Escape') return
  desktop.startOpen = false
  desktop.notifOpen = false
}

onMounted(() => {
  syncArea()
  window.addEventListener('resize', syncArea, { passive: true })
  window.addEventListener('pointermove', onPointerMove, { passive: true })
  window.addEventListener('keydown', onKeydown)
  // Рабочая область готова — можно поднимать разделы прошлой сессии.
  boot()
  // Режим меняют и мимо кнопки — клавишей F11 или Esc: следим за состоянием,
  // иначе кнопка предлагала бы уже сделанное.
  stopFullscreenWatch = onPageFullscreenChange(() => { pageFull.value = isPageFullscreen() })
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', syncArea)
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('keydown', onKeydown)
  stopFullscreenWatch?.()
})
</script>

<style scoped>
.desktop {
  position: relative;
  isolation: isolate;
  flex: 1;
  min-width: 0;
  height: 100dvh;
  overflow: hidden;
}

/* Обои: мягкие волны из цветов темы поверх фонового градиента приложения.
   Только токены — обои живут вместе с темой и тёмным режимом. */
/* Слой обоев рисует себя под контентом (z-index:-1 внутри компонента), поэтому
   рабочий стол обязан быть стекинг-контекстом — иначе обои уедут за него. */
.desk-paper {
  position: absolute;
  inset: 0;
}

.desk-wallpaper {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    radial-gradient(120vmax 60vmax at 18% 120%, color-mix(in oklch, var(--color-primary) 16%, transparent), transparent 58%),
    radial-gradient(90vmax 50vmax at 92% 8%, color-mix(in oklch, var(--color-tertiary) 12%, transparent), transparent 60%),
    linear-gradient(122deg, transparent 38%, color-mix(in oklch, var(--color-primary) 9%, transparent) 52%, transparent 66%);
}

/* Слой окон растянут на весь стол, но сам указатель не ловит — иначе правый
   клик по «пустому месту» доставался бы ему, а не рабочему столу. Окна
   возвращают себе pointer-events своим стилем. */
.desk-windows {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

/* Полупрозрачная «примерка» будущего положения окна при прилипании к краю. */
.desk-snap {
  position: absolute;
  top: 0;
  left: 0;
  border: 2px solid color-mix(in oklch, var(--color-primary) 55%, transparent);
  border-radius: var(--radius-xl);
  background: color-mix(in oklch, var(--color-primary) 14%, transparent);
  pointer-events: none;
  z-index: 890;
  transition: transform 0.12s ease, width 0.12s ease, height 0.12s ease;
}

.desk-hint {
  position: absolute;
  left: 0;
  right: 0;
  bottom: calc(var(--taskbar-height) + 36px);
  margin: 0;
  text-align: center;
  font-size: 13.5px;
  color: color-mix(in oklch, var(--color-text-dim) 80%, transparent);
  pointer-events: none;
}

/* Кнопка полноэкранного режима — выезжает из-за верхней кромки экрана.
   Над окнами, но ниже меню «Пуск» и центра уведомлений. */
.desk-full {
  position: fixed;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  z-index: 940;
  display: flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 18px;
  border: 1px solid var(--acrylic-border);
  border-top: none;
  border-radius: 0 0 var(--radius-lg) var(--radius-lg);
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  box-shadow: var(--shadow-lg);
  color: var(--color-text);
  font-size: 13.5px;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}

.desk-full:hover { color: var(--color-primary); border-color: color-mix(in oklch, var(--color-primary) 32%, var(--acrylic-border)); }
.desk-full .material-symbols-outlined { font-size: 20px; }

.desk-full-enter-active,
.desk-full-leave-active { transition: opacity 0.18s ease, translate 0.2s cubic-bezier(0.2, 0, 0, 1); }
.desk-full-enter-from,
.desk-full-leave-to { opacity: 0; translate: 0 -100%; }
</style>
