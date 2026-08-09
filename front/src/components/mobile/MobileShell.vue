<template>
  <div class="mshell">
    <!-- Обои — те же, что на рабочем столе (личная настройка переезжает между
         устройствами). Сняты все слои — фон остаётся чистым. -->
    <ChatBackgroundLayer v-if="wallpaper" :recipe="wallpaper" class="ms-paper" />

    <!-- Панель статусов — только на телефоне: там она единственное место для
         компании, юнита и уведомлений. На планшете всё это уместилось в
         панель задач и в колонку аккаунта «Пуска», а верхняя полоса лишь
         отнимала высоту. -->
    <MobileStatusBar v-if="!tablet" />
    <MobileStart v-show="startVisible" :platform="platform" :split="tablet" />

    <!-- Слой разделов: открытые остаются смонтированными, виден активный (на
         планшете — два: главная зона и вторая). Экраны лежат в ОДНОМ слое и
         разводятся по зонам инлайновой геометрией: перенос в другой контейнер
         перемонтировал бы раздел и стёр его состояние. -->
    <div class="ms-screens" :class="{ hidden: startVisible }">
      <AppScreen
        v-for="win in desktop.windows"
        :key="win.id"
        :win="win"
        :active="isVisible(win)"
        :style="zoneStyle(win)"
      />
    </div>

    <!-- Разделитель зон: тянется пальцем, при отпускании прилипает к ступени. -->
    <div
      v-if="splitOn"
      class="ms-split"
      :class="[{ dragging }, collapsing ? `collapse-${collapsing}` : '']"
      :style="{ left: `${desktop.splitRatio}%` }"
      role="separator"
      aria-label="Граница зон — потяните, чтобы изменить"
      @pointerdown="onSplitDown"
      @dblclick="desktop.closeSide()"
    >
      <span class="ms-split-grip" />
    </div>

    <!-- Панель задач: у планшета — настольная в компактном сенсорном виде
         (юнит, часы и уведомления там уже есть), у телефона — своя. -->
    <Taskbar v-if="tablet" platform="tablet" touch />
    <MobileTaskbar v-else :platform="platform" />

    <Transition name="np">
      <NotificationsPanel v-if="desktop.notifOpen" full @close="desktop.notifOpen = false" />
    </Transition>
    <Transition name="hp">
      <HolaPopup v-if="desktop.holaOpen" full @close="desktop.holaOpen = false" />
    </Transition>
  </div>
</template>

<script setup>
/**
 * Сенсорный каркас в духе Windows 10 Mobile: стартовый экран с плитками,
 * всегда видимая панель задач и разделы-«приложения» поверх неё. Обслуживает
 * ДВА устройства — телефон и планшет (`platform`): у них разные раскладки
 * «Пуска» и разный предел открытых разделов, а планшет вдобавок умеет вторую
 * зону.
 *
 * Разделы открываются тем же стором, что и окна рабочего стола (см.
 * stores/desktop.js), поэтому переключение между ними сохраняет состояние, а
 * личные настройки плиток и закреплений общие с настольным каркасом.
 * Отличия — в раскладке (окон нет, есть зоны) и в навигации: адрес пишется
 * через push, поэтому системная кнопка «назад» ходит по разделам сама, без
 * перехвата жестов.
 *
 * Вторая зона (планшет) — ответ на то, что окна плохо слушаются пальца: вместо
 * кромок и заголовков раздел ставится рядом одним пунктом меню, а соотношение
 * задаётся одним разделителем со ступенями 1/3, 1/2, 2/3.
 */
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useDesktopStore } from '@/stores/desktop.js'
import { useShellCore } from '@/desktop/shellCore.js'
import {
  MOBILE_STATUSBAR_HEIGHT, MOBILE_TASKBAR_HEIGHT, TABLET_TASKBAR_HEIGHT,
  areaInsets, taskbarSide,
} from '@/desktop/layout.js'
import ChatBackgroundLayer from '@/components/common/ChatBackgroundLayer.vue'

/* Панели поверх каркаса открываются по кнопке — ленивыми чанками (Hola тянет
   за собой поиск по всем разделам и чат ассистента). */
const NotificationsPanel = defineAsyncComponent(() => import('@/components/desktop/NotificationsPanel.vue'))
const HolaPopup = defineAsyncComponent(() => import('@/components/desktop/HolaPopup.vue'))
import MobileStatusBar from './MobileStatusBar.vue'
import MobileStart from './MobileStart.vue'
import MobileTaskbar from './MobileTaskbar.vue'
import Taskbar from '@/components/desktop/Taskbar.vue'
import AppScreen from './AppScreen.vue'

/* Сколько разделов держим открытыми: они остаются смонтированными, а память
   устройства не бесконечна — самый давний закрывается сам (см. desktop.limit).
   У планшета памяти больше, а зон две — предел выше. */
const SCREEN_LIMIT = { mobile: 6, tablet: 8 }

const props = defineProps({
  /** 'mobile' — телефон, 'tablet' — большой сенсорный экран с двумя зонами. */
  platform: { type: String, default: 'mobile' },
})

const desktop = useDesktopStore()

const tablet = computed(() => props.platform === 'tablet')

// Стартовый экран на переднем плане: его открыли кнопкой «Пуск» либо разделов
// не осталось вовсе.
const startVisible = computed(() => desktop.startOpen || !desktop.focused)

const { wallpaper, boot } = useShellCore({
  activePath: () => (startVisible.value ? '/home' : desktop.focused.path),
  barHeight: props.platform === 'tablet' ? TABLET_TASKBAR_HEIGHT : MOBILE_TASKBAR_HEIGHT,
  // Панель прижата к кромке — зазора между ней и разделом нет.
  barMargin: 0,
  limit: SCREEN_LIMIT[props.platform] || SCREEN_LIMIT.mobile,
  // Каждый раздел — своя запись в истории браузера, поэтому системное «назад»
  // возвращает к предыдущему разделу, а из первого — на стартовый экран.
  navigate: 'push',
  platform: props.platform,
  onHome: () => { desktop.startOpen = true },
})

/* Рабочая область: весь экран минус панель задач. Раскладке разделов она не
   нужна (каждый занимает экран целиком), но её читают общие модули — панель
   Hola и плавающий питомец. */
function syncArea() {
  desktop.setScreen({ x: 0, y: 0, w: window.innerWidth, h: window.innerHeight })
  const inset = areaInsets('bottom')
  desktop.setArea({
    x: 0,
    y: 0,
    w: window.innerWidth,
    h: Math.max(240, window.innerHeight - inset.bottom),
  })
}

/* ── Две зоны (планшет) ────────────────────────────────────────
   Экраны живут в одном слое и разводятся по зонам инлайновой геометрией:
   инлайн перебивает `inset` из стилей AppScreen, а сам экран не переезжает
   между контейнерами и потому не перемонтируется. */
const splitOn = computed(() => tablet.value && desktop.split && !startVisible.value)

function isVisible(win) {
  if (startVisible.value) return false
  if (desktop.focusedId === win.id) return true
  return splitOn.value && desktop.sideId === win.id
}

function zoneStyle(win) {
  if (!splitOn.value) return null
  if (desktop.sideId === win.id) return { left: `${desktop.splitRatio}%`, right: '0' }
  if (desktop.focusedId === win.id) return { right: `${100 - desktop.splitRatio}%` }
  return null
}

// Портрет или просто узкий экран — второй зоне негде поместиться: держим её
// свёрнутой, пока места не станет достаточно.
const MIN_ZONE = 340

function fitSplit() {
  if (desktop.sideId && window.innerWidth < MIN_ZONE * 2) desktop.closeSide()
}

function onResize() {
  syncArea()
  fitSplit()
}

/* Уведённый к самому краю разделитель схлопывает зону: так возвращаются к
   одному разделу во весь экран — жест обратный тому, которым зону открывали.
   Влево — уходит главная (её место занимает вторая), вправо — уходит вторая. */
const COLLAPSE_AT = 12

const dragging = ref(false)
const collapsing = ref(null)

function ratioAt(clientX) {
  return (clientX / window.innerWidth) * 100
}

function onSplitDown(e) {
  if (e.button !== undefined && e.button !== 0) return
  e.currentTarget.setPointerCapture?.(e.pointerId)
  dragging.value = true

  const onMove = (ev) => {
    const pct = ratioAt(ev.clientX)
    // Подсвечиваем намерение заранее: у края видно, какая зона сейчас уйдёт.
    collapsing.value = pct <= COLLAPSE_AT ? 'main' : (pct >= 100 - COLLAPSE_AT ? 'side' : null)
    desktop.setSplitRatio(pct)
  }

  const onUp = (ev) => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
    window.removeEventListener('pointercancel', onUp)
    dragging.value = false
    const zone = collapsing.value
    collapsing.value = null
    if (zone) {
      // Схлопываем и возвращаем ровное соотношение — следующее деление
      // начнётся пополам, а не с края.
      if (zone === 'main') desktop.swapSides()
      desktop.closeSide()
      desktop.setSplitRatio(50, { snap: true })
      return
    }
    desktop.setSplitRatio(ratioAt(ev.clientX), { snap: true })
  }

  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
  window.addEventListener('pointercancel', onUp)
}

// Открытый центр уведомлений накрывает раздел целиком — переключение экранов
// его закрывает.
watch(startVisible, () => { desktop.notifOpen = false })

onMounted(() => {
  // На телефоне панель задач всегда снизу: вертикальная сторона из настроек
  // рабочего стола сюда не переносится.
  taskbarSide.value = 'bottom'
  /* Толщины панелей — на КОРНЕ документа, а не на каркасе: их читают и всплывашки,
     телепортированные в body (тосты), которым каркас ничего не наследует. */
  const root = document.documentElement.style
  root.setProperty('--taskbar-height', `${tablet.value ? TABLET_TASKBAR_HEIGHT : MOBILE_TASKBAR_HEIGHT}px`)
  // Верхней панели на планшете нет — резерв под неё нулевой.
  root.setProperty('--statusbar-height', `${tablet.value ? 0 : MOBILE_STATUSBAR_HEIGHT}px`)
  syncArea()
  fitSplit()
  window.addEventListener('resize', onResize, { passive: true })
  boot()
  /* Сессия общая с настольным каркасом (одно и то же устройство могло
     работать в обоих): свёрнутых экранов на телефоне не бывает. */
  desktop.windows.forEach((w) => desktop.restore(w.id))
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  const root = document.documentElement.style
  root.removeProperty('--taskbar-height')
  root.removeProperty('--statusbar-height')
})
</script>

<style scoped>
.mshell {
  position: relative;
  isolation: isolate;
  flex: 1;
  min-width: 0;
  height: 100dvh;
  overflow: hidden;
}

/* Слой обоев рисует себя под контентом (z-index:-1 внутри компонента), поэтому
   каркас обязан быть стекинг-контекстом — иначе обои уедут за него. */
.ms-paper {
  position: absolute;
  inset: 0;
}

.ms-screens {
  position: absolute;
  inset: 0;
}

/* Стартовый экран поверх — разделы не должны перехватывать касания. */
.ms-screens.hidden { visibility: hidden; }

/* ── Разделитель зон ──
   Зона хватания шире, чем видимая линия: попасть пальцем в пару пикселей
   нельзя. Живёт между панелями каркаса — те остаются доступны при любом
   соотношении. */
.ms-split {
  position: absolute;
  top: calc(var(--statusbar-height) + env(safe-area-inset-top, 0px));
  bottom: calc(var(--taskbar-height) + env(safe-area-inset-bottom, 0px));
  width: 30px;
  margin-left: -15px;
  z-index: 5;
  display: grid;
  place-items: center;
  cursor: ew-resize;
  touch-action: none;
}

/* Сама граница — видимая линия во всю высоту: без неё два раздела встык
   сливались в один экран, и было непонятно, где кончается первый. */
.ms-split::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 50%;
  width: 2px;
  margin-left: -1px;
  background: color-mix(in oklch, var(--color-text) 22%, transparent);
  transition: background 0.15s;
}

.ms-split:hover::before,
.ms-split.dragging::before { background: var(--color-primary); }

/* У края линия наливается и толстеет — видно, что отпускание схлопнет зону. */
.ms-split.collapse-main::before,
.ms-split.collapse-side::before {
  width: 6px;
  margin-left: -3px;
  background: var(--color-primary);
}

/* Ручка поверх линии — подсказка «это можно тянуть». */
.ms-split-grip {
  position: relative;
  width: 6px;
  height: 56px;
  border-radius: var(--radius-full);
  background: color-mix(in oklch, var(--color-text) 42%, var(--color-surface));
  box-shadow: 0 1px 4px color-mix(in oklch, var(--color-text) 22%, transparent);
  transition: background 0.15s, height 0.15s;
}

.ms-split:hover .ms-split-grip,
.ms-split.dragging .ms-split-grip {
  background: var(--color-primary);
  height: 84px;
}
</style>
