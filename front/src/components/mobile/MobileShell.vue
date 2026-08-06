<template>
  <div class="mshell">
    <!-- Обои — те же, что на рабочем столе (личная настройка переезжает между
         устройствами). Сняты все слои — фон остаётся чистым. -->
    <ChatBackgroundLayer v-if="wallpaper" :recipe="wallpaper" class="ms-paper" />

    <!-- Панель статусов видна всегда: компания, идущий юнит и уведомления
         нужны и внутри раздела. -->
    <MobileStatusBar />
    <MobileStart v-show="startVisible" />

    <!-- Слой разделов: открытые остаются смонтированными, виден активный. -->
    <div class="ms-screens" :class="{ hidden: startVisible }">
      <AppScreen
        v-for="win in desktop.windows"
        :key="win.id"
        :win="win"
        :active="!startVisible && desktop.focusedId === win.id"
      />
    </div>

    <MobileTaskbar />

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
 * Мобильный каркас в духе Windows 10 Mobile: стартовый экран с плитками,
 * всегда видимая панель задач и разделы-«приложения» поверх неё.
 *
 * Разделы открываются тем же стором, что и окна рабочего стола (см.
 * stores/desktop.js), поэтому переключение между ними сохраняет состояние, а
 * личные настройки плиток и закреплений общие с настольным каркасом.
 * Отличия — в раскладке (одно окно на экран, без геометрии) и в навигации:
 * адрес пишется через push, поэтому системная кнопка «назад» ходит по
 * разделам сама, без перехвата жестов.
 */
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, watch } from 'vue'
import { useDesktopStore } from '@/stores/desktop.js'
import { useShellCore } from '@/desktop/shellCore.js'
import { MOBILE_STATUSBAR_HEIGHT, MOBILE_TASKBAR_HEIGHT, areaInsets, taskbarSide } from '@/desktop/layout.js'
import ChatBackgroundLayer from '@/components/common/ChatBackgroundLayer.vue'

/* Панели поверх каркаса открываются по кнопке — ленивыми чанками (Hola тянет
   за собой поиск по всем разделам и чат ассистента). */
const NotificationsPanel = defineAsyncComponent(() => import('@/components/desktop/NotificationsPanel.vue'))
const HolaPopup = defineAsyncComponent(() => import('@/components/desktop/HolaPopup.vue'))
import MobileStatusBar from './MobileStatusBar.vue'
import MobileStart from './MobileStart.vue'
import MobileTaskbar from './MobileTaskbar.vue'
import AppScreen from './AppScreen.vue'

/* Сколько разделов держим открытыми: они остаются смонтированными, а память
   телефона не бесконечна — самый давний закрывается сам (см. desktop.limit). */
const SCREEN_LIMIT = 6

const desktop = useDesktopStore()

// Стартовый экран на переднем плане: его открыли кнопкой «Пуск» либо разделов
// не осталось вовсе.
const startVisible = computed(() => desktop.startOpen || !desktop.focused)

const { wallpaper, boot } = useShellCore({
  activePath: () => (startVisible.value ? '/home' : desktop.focused.path),
  barHeight: MOBILE_TASKBAR_HEIGHT,
  // Панель прижата к кромке — зазора между ней и разделом нет.
  barMargin: 0,
  limit: SCREEN_LIMIT,
  // Каждый раздел — своя запись в истории браузера, поэтому системное «назад»
  // возвращает к предыдущему разделу, а из первого — на стартовый экран.
  navigate: 'push',
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
  root.setProperty('--taskbar-height', `${MOBILE_TASKBAR_HEIGHT}px`)
  root.setProperty('--statusbar-height', `${MOBILE_STATUSBAR_HEIGHT}px`)
  syncArea()
  window.addEventListener('resize', syncArea, { passive: true })
  boot()
  /* Сессия общая с настольным каркасом (одно и то же устройство могло
     работать в обоих): свёрнутых экранов на телефоне не бывает. */
  desktop.windows.forEach((w) => desktop.restore(w.id))
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', syncArea)
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
</style>
