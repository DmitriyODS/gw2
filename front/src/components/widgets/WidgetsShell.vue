<template>
  <div class="wshell">
    <!-- Обои: те же, что у стола и телефона (личная настройка едет за
         человеком). Видны на пустом холсте и в зазорах панели. -->
    <ChatBackgroundLayer v-if="wallpaper" :recipe="wallpaper" class="ws-paper" />

    <WidgetsPanel :width="panelWidth" :collapsed="collapsed" @toggle="toggleCollapsed" />

    <!-- Сцена: раздел занимает всё место справа от панели. Экраны остаются
         смонтированными (v-show внутри AppScreen) — переключение возвращает
         раздел с его прокруткой, фильтрами и черновиками. -->
    <div class="ws-stage">
      <AppScreen
        v-for="win in desktop.windows"
        :key="win.id"
        :win="win"
        :active="!startVisible && desktop.focusedId === win.id"
      />

      <!-- Пока раздел не выбран, место работает: лента «Моя активность»
           открывает то, чем занимались. -->
      <div v-if="startVisible" class="ws-empty">
        <ActivityPanel class="ws-activity" @open="desktop.open" />
      </div>
    </div>

    <!-- Граница панели: тянется мышью, двойной клик возвращает ширину по
         умолчанию (пятая часть экрана). У свёрнутой полоски тянуть нечего. -->
    <div
      v-if="!collapsed"
      class="ws-grip"
      :class="{ dragging }"
      :style="{ left: `${panelWidth}px` }"
      role="separator"
      aria-label="Ширина панели — потяните, чтобы изменить"
      @pointerdown="onGripDown"
      @dblclick="prefs.setWidgetsWidth(0)"
    />

    <Transition name="np">
      <NotificationsPanel v-if="desktop.notifOpen" side="left" @close="desktop.notifOpen = false" />
    </Transition>
    <Transition name="hp">
      <HolaPopup v-if="desktop.holaOpen" @close="desktop.holaOpen = false" />
    </Transition>
  </div>
</template>

<script setup>
/**
 * Каркас «Виджеты» — раскладка обычного приложения: слева панель разделов
 * живыми плитками, справа один раздел во весь остаток экрана. Окон нет,
 * панели задач нет; всё, что жило в них, собрала боковая панель.
 *
 * Разделы открываются ТЕМ ЖЕ стором, что окна стола и экраны телефона
 * (stores/desktop.js), поэтому переключение сохраняет состояние, а права,
 * обои, живые плитки и синхронизация адреса приходят из общего ядра
 * (desktop/shellCore.js). Здесь остаётся только раскладка.
 *
 * Открытыми держим не больше SCREEN_LIMIT разделов: они остаются
 * смонтированными, и память не бесконечна — самый давний закрывается сам.
 */
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useDesktopStore } from '@/stores/desktop.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { useShellCore } from '@/desktop/shellCore.js'
import {
  WIDGETS_PANEL_MAX, WIDGETS_PANEL_MIN, WIDGETS_RAIL_WIDTH,
  taskbarHeight, taskbarSide, widgetsPanelWidth,
} from '@/desktop/layout.js'
import ChatBackgroundLayer from '@/components/common/ChatBackgroundLayer.vue'
import AppScreen from '@/components/mobile/AppScreen.vue'
import ActivityPanel from '@/components/desktop/ActivityPanel.vue'
import WidgetsPanel from './WidgetsPanel.vue'

/* Панели поверх каркаса открываются по кнопке — ленивыми чанками (Hola тянет
   за собой поиск по всем разделам и чат ассистента). */
const NotificationsPanel = defineAsyncComponent(() => import('@/components/desktop/NotificationsPanel.vue'))
const HolaPopup = defineAsyncComponent(() => import('@/components/desktop/HolaPopup.vue'))

const SCREEN_LIMIT = 5

const desktop = useDesktopStore()
const prefs = useDesktopPrefsStore()

const collapsed = computed(() => prefs.widgetsCollapsed)

// Ширина панели: личная настройка, зажатая границами экрана; свёрнутая
// панель — полоска значков постоянной ширины.
const screenW = ref(typeof window === 'undefined' ? 1280 : window.innerWidth)
const panelWidth = computed(() => (collapsed.value
  ? WIDGETS_RAIL_WIDTH
  : widgetsPanelWidth(prefs.widgetsWidth, screenW.value)))

/* Холст без раздела: его показывают и «домой» из адресной строки, и отсутствие
   открытых разделов. Флаг общий со стартовым экраном телефона — им же
   пользуется ядро каркасов. */
const startVisible = computed(() => desktop.startOpen || !desktop.focused)

const { wallpaper, boot } = useShellCore({
  activePath: () => (startVisible.value ? '/home' : desktop.focused.path),
  barHeight: panelWidth.value,
  // Панель прижата к кромке — зазора между ней и разделом нет.
  barMargin: 0,
  limit: SCREEN_LIMIT,
  /* Каждый раздел — своя запись в истории браузера: «назад» возвращает к
     предыдущему разделу, как в обычном приложении с боковой навигацией. */
  navigate: 'push',
  platform: 'widgets',
  onHome: () => { desktop.startOpen = true },
})

function toggleCollapsed() {
  prefs.setWidgetsCollapsed(!collapsed.value)
}

/* ── Геометрия ────────────────────────────────────────────────
   Рабочая область — экран минус панель: её читают плавающий питомец, панель
   Hola и центр уведомлений. Панель вертикальная, поэтому в общей геометрии
   она «панель задач слева» толщиной в свою ширину. */
function syncArea() {
  screenW.value = window.innerWidth
  desktop.setScreen({ x: 0, y: 0, w: window.innerWidth, h: window.innerHeight })
  desktop.setArea({
    x: panelWidth.value,
    y: 0,
    w: Math.max(320, window.innerWidth - panelWidth.value),
    h: window.innerHeight,
  })
  // Центр уведомлений выезжает из панели — ему нужен её прямоугольник.
  Object.assign(desktop.taskbarRect, { x: 0, y: 0, w: panelWidth.value, h: window.innerHeight })
}

watch(panelWidth, (w) => {
  taskbarHeight.value = w
  syncArea()
})

/* ── Перетаскивание границы ──────────────────────────────────── */
const dragging = ref(false)

function onGripDown(e) {
  if (e.button !== undefined && e.button !== 0) return
  e.currentTarget.setPointerCapture?.(e.pointerId)
  dragging.value = true

  const apply = (ev) => {
    const px = Math.round(ev.clientX)
    // Уведённая за минимум граница сворачивает панель — жест обратный тому,
    // которым её разворачивают кнопкой.
    if (px < WIDGETS_PANEL_MIN - 40) return
    prefs.setWidgetsWidth(Math.min(WIDGETS_PANEL_MAX, Math.max(WIDGETS_PANEL_MIN, px)))
  }

  const onUp = (ev) => {
    window.removeEventListener('pointermove', apply)
    window.removeEventListener('pointerup', onUp)
    window.removeEventListener('pointercancel', onUp)
    dragging.value = false
    if (Math.round(ev.clientX) < WIDGETS_PANEL_MIN - 40) prefs.setWidgetsCollapsed(true)
  }

  window.addEventListener('pointermove', apply)
  window.addEventListener('pointerup', onUp)
  window.addEventListener('pointercancel', onUp)
}

function onKeydown(e) {
  // Ctrl/Cmd+K — всплывающая панель Hola: поиск, команды и чат с ассистентом.
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    desktop.holaOpen = !desktop.holaOpen
    return
  }
  if (e.key !== 'Escape') return
  desktop.notifOpen = false
  desktop.holaOpen = false
}

onMounted(() => {
  // Панель вертикальная: высоты экрана она не отнимает, поэтому переменные
  // толщины — нулевые. Ставим их на КОРНЕ документа: их читают и всплывашки,
  // телепортированные в body (тосты), которым каркас ничего не наследует.
  const root = document.documentElement.style
  root.setProperty('--taskbar-height', '0px')
  root.setProperty('--statusbar-height', '0px')
  taskbarSide.value = 'left'
  syncArea()
  window.addEventListener('resize', syncArea, { passive: true })
  window.addEventListener('keydown', onKeydown)
  // Геометрия готова — можно поднимать разделы прошлой сессии.
  boot()
  /* Сессия общая с настольным каркасом (одно и то же устройство могло работать
     в обоих): свёрнутых разделов здесь не бывает. */
  desktop.windows.forEach((w) => desktop.restore(w.id))
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', syncArea)
  window.removeEventListener('keydown', onKeydown)
  const root = document.documentElement.style
  root.removeProperty('--taskbar-height')
  root.removeProperty('--statusbar-height')
})
</script>

<style scoped>
.wshell {
  position: relative;
  isolation: isolate;
  flex: 1;
  min-width: 0;
  height: 100dvh;
  display: flex;
  overflow: hidden;
}

/* Слой обоев рисует себя под контентом (z-index:-1 внутри компонента), поэтому
   каркас обязан быть стекинг-контекстом — иначе обои уедут за него. */
.ws-paper {
  position: absolute;
  inset: 0;
}

/* Сцена — контейнер раздела: экраны лежат в нём абсолютом (inset считает сам
   AppScreen по нулевым толщинам панелей). */
.ws-stage {
  position: relative;
  flex: 1;
  min-width: 0;
}

/* Флекс, а не грид: в гриде строка auto-размера зависит от содержимого, и
   процентный max-height ленты не разрешался — она вырастала во всю длину
   журнала и обрезалась холстом вместо того, чтобы прокручиваться. */
.ws-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.ws-activity {
  width: min(460px, 100%);
  max-height: min(560px, 100%);
}

/* Зона хватания шире видимой линии: попасть в пару пикселей мышью трудно. */
.ws-grip {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 12px;
  margin-left: -6px;
  z-index: 901;
  cursor: col-resize;
  touch-action: none;
}

.ws-grip::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 50%;
  width: 2px;
  margin-left: -1px;
  background: transparent;
  transition: background 0.15s;
}

.ws-grip:hover::before,
.ws-grip.dragging::before { background: var(--color-primary); }
</style>
