<template>
  <div class="app-layout" :data-dark="themeStore.dark">
    <div v-if="initializing" class="app-loading">
      <ProgressSpinner />
    </div>
    <template v-else-if="isFullscreenRoute">
      <main class="main-content fullscreen-content">
        <router-view />
      </main>
    </template>
    <template v-else-if="authStore.user">
      <AppSidebar />
      <main class="main-content">
        <router-view />
      </main>
      <AppBottomNav />
      <ActiveUnitModal v-if="unitsStore.activeUnit" />
      <AppTutorial v-if="isTutorialOpen" />
      <ChangelogModal v-if="isChangelogOpen" @close="closeChangelog" />
      <StaleTasksModal v-if="isStaleOpen" :tasks="staleTasks" @close="closeStale" />
      <MiniMessenger />
      <IncomingCallOverlay @accept="callStore.accept()" @decline="callStore.decline()" />
      <CallView />
      <ReturnCallBanner />
    </template>
    <template v-else>
      <main class="main-content">
        <router-view />
      </main>
    </template>
    <Toast :position="isMobile ? 'top-center' : 'top-right'" />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import { useUnitsStore } from '@/stores/units.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useCallStore } from '@/stores/call.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useTutorial } from '@/composables/useTutorial.js'
import { useChangelog } from '@/composables/useChangelog.js'
import { useStaleReminder } from '@/composables/useStaleReminder.js'
import { connectSocket } from '@/socket/index.js'
import {
  registerNotifyServiceWorker, installNotifyUnlock, requestNotificationPermission,
} from '@/utils/systemNotify.js'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import AppBottomNav from '@/components/layout/AppBottomNav.vue'
import ActiveUnitModal from '@/components/layout/ActiveUnitModal.vue'
import AppTutorial from '@/components/layout/AppTutorial.vue'
import ChangelogModal from '@/components/layout/ChangelogModal.vue'
import StaleTasksModal from '@/components/tasks/StaleTasksModal.vue'
import MiniMessenger from '@/components/messenger/MiniMessenger.vue'
import IncomingCallOverlay from '@/components/call/IncomingCallOverlay.vue'
import CallView from '@/components/call/CallView.vue'
import ReturnCallBanner from '@/components/call/ReturnCallBanner.vue'
import Toast from 'primevue/toast'
import ProgressSpinner from 'primevue/progressspinner'

const authStore = useAuthStore()
const themeStore = useThemeStore()
const unitsStore = useUnitsStore()
const messengerStore = useMessengerStore()
const callStore = useCallStore()
const notif = useNotificationsStore()
const route = useRoute()
const { isMobile } = useBreakpoint()

const isFullscreenRoute = computed(() => !!route.meta?.fullscreen && !!authStore.user)
// isOpen деструктурирован как топ-левел ref — Vue auto-unwraps в шаблоне
const { isOpen: isTutorialOpen, open: openTutorial, shouldAutoShow } = useTutorial()
const { isOpen: isChangelogOpen, close: closeChangelog, checkForNewVersion } = useChangelog()
const { isOpen: isStaleOpen, tasks: staleTasks, close: closeStale, check: checkStaleTasks } = useStaleReminder()

watch(() => authStore.user, (user, prev) => {
  if (user && !prev && shouldAutoShow()) {
    setTimeout(() => openTutorial(), 600)
  }
})

// useToast() требует setup-контекст — вызываем здесь, не в onMounted
const toast = useToast()
notif.setToast(toast)

// Применяем тему сразу (без FOUC)
themeStore.init()

const initializing = ref(true)

onMounted(async () => {
  // Восстановление сессии централизовано в auth-store и уже инициируется
  // router guard'ом — здесь лишь дожидаемся его и поднимаем сокет/юнит.
  await authStore.ensureReady()
  if (authStore.token) {
    connectSocket()
    await unitsStore.fetchActiveUnit()
    // Уведомления: регистрируем SW (нужен для OS-уведомлений на мобильных),
    // сразу просим разрешение (Chrome/Firefox показывают prompt без жеста) и
    // вешаем «разогрев» аудио + повторный запрос по первому жесту (для Safari).
    registerNotifyServiceWorker()
    requestNotificationPermission()
    installNotifyUnlock()
    // Список диалогов нужен сразу после входа: бейдж непрочитанных, мини-чат
    // и корректный заголовок в push-уведомлении (иначе fio неизвестно).
    messengerStore.fetchConversations()
    // Если страницу перезагрузили во время звонка — звонок ещё «жив» на
    // сервере (grace-окно). Предложим вернуться к нему.
    callStore.checkRejoin()
    // Лог версий показываем существующим пользователям; новичкам сначала тур,
    // а лог всплывёт при следующем входе.
    if (!shouldAutoShow()) {
      checkForNewVersion()
    }
    // Напоминание о давних задачах — раз в день, и только если не показываем
    // тур/лог версий (чтобы не громоздить модалки друг на друга).
    setTimeout(() => {
      if (!isTutorialOpen.value && !isChangelogOpen.value) {
        checkStaleTasks()
      }
    }, 1200)
  }
  initializing.value = false
})

// Сброс данных мессенджера при логауте, чтобы не утекли между сессиями.
watch(() => authStore.user, (user) => {
  if (!user) {
    messengerStore.reset()
  }
})

/* Клик по системному уведомлению о звонке (через service worker) приходит
   сюда — фокусируем окно и разворачиваем CallView, если звонок уже принят
   и был свёрнут в мини-режим. Сам входящий overlay уже виден сам по себе
   (он на phase === 'incoming'). */
function onCallFocusOverlay() {
  if (callStore.isMinimized) callStore.expand()
}
onMounted(() => {
  window.addEventListener('call:focus-overlay', onCallFocusOverlay)
})
onBeforeUnmount(() => {
  window.removeEventListener('call:focus-overlay', onCallFocusOverlay)
})
</script>

<style>
.app-loading {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 100vh;
  background: var(--gw-bg);
}

.fullscreen-content {
  width: 100vw;
  min-height: 100vh;
}
</style>
