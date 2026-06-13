<template>
  <div class="app-layout" :data-dark="themeStore.dark">
    <div v-if="navProgress" class="nav-progress" aria-hidden="true">
      <div class="nav-progress-bar" />
    </div>
    <div v-if="initializing" class="app-loading">
      <ProgressSpinner />
    </div>
    <CompanyDisabledScreen v-else-if="authStore.companyDisabled" />
    <template v-else-if="isFullscreenRoute">
      <main class="main-content fullscreen-content">
        <router-view />
      </main>
    </template>
    <template v-else-if="authStore.token">
      <AppSidebar />
      <main class="main-content">
        <router-view />
      </main>
      <AppBottomNav />
      <ActiveUnitModal v-if="unitsStore.activeUnit" />
      <AppTutorial v-if="isTutorialOpen" />
      <ChangelogModal v-if="isChangelogOpen" @close="closeChangelog" />
      <MorningBriefingModal v-if="usesGroove && isBriefingOpen" :briefing="briefing" @close="closeBriefing" />
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
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { useTutorial } from '@/composables/useTutorial.js'
import { useChangelog } from '@/composables/useChangelog.js'
import { useMorningBriefing } from '@/composables/useMorningBriefing.js'
import { connectSocket } from '@/socket/index.js'
import { navProgress } from '@/composables/useNavProgress.js'
import {
  registerNotifyServiceWorker, installNotifyUnlock, requestNotificationPermission,
} from '@/utils/systemNotify.js'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import AppBottomNav from '@/components/layout/AppBottomNav.vue'
import CompanyDisabledScreen from '@/components/layout/CompanyDisabledScreen.vue'
import ActiveUnitModal from '@/components/layout/ActiveUnitModal.vue'
import AppTutorial from '@/components/layout/AppTutorial.vue'
import ChangelogModal from '@/components/layout/ChangelogModal.vue'
import MorningBriefingModal from '@/components/groove/MorningBriefingModal.vue'
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
const { usesGroove } = useCompanySettings()

const isFullscreenRoute = computed(() => !!route.meta?.fullscreen && !!authStore.user)
// isOpen деструктурирован как топ-левел ref — Vue auto-unwraps в шаблоне
const { isOpen: isTutorialOpen, open: openTutorial, shouldAutoShow } = useTutorial()
const { isOpen: isChangelogOpen, close: closeChangelog, checkForNewVersion } = useChangelog()
const { isOpen: isBriefingOpen, briefing, close: closeBriefing, check: checkMorningBriefing } = useMorningBriefing()
let tutorialTimer = null
let briefingTimer = null

watch(() => authStore.user, (user, prev) => {
  if (user && !prev && shouldAutoShow()) {
    clearTimeout(tutorialTimer)
    tutorialTimer = setTimeout(() => openTutorial(), 600)
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
  // Снимаем загрузочный экран сразу, как только известен статус авторизации.
  // Остальная инициализация (сокет, активный юнит, диалоги, уведомления) идёт
  // фоном и НЕ должна держать первый рендер: иначе по deep-link (/tasks/:id)
  // экран остаётся белым до конца всей цепочки запросов.
  initializing.value = false
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
    await messengerStore.fetchConversations().catch(() => {})
    // Если страницу перезагрузили во время звонка — звонок ещё «жив» на
    // сервере (grace-окно). Предложим вернуться к нему.
    callStore.checkRejoin()
    // Лог версий показываем существующим пользователям; новичкам сначала тур,
    // а лог всплывёт при следующем входе.
    if (!shouldAutoShow()) {
      checkForNewVersion()
    }
    // Утренний брифинг от Грувика — раз в день, и только если не показываем
    // тур/лог версий (чтобы не громоздить модалки друг на друга).
    clearTimeout(briefingTimer)
    briefingTimer = setTimeout(() => {
      if (usesGroove.value && !isTutorialOpen.value && !isChangelogOpen.value) {
        checkMorningBriefing()
      }
    }, 1200)
  }
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
  clearTimeout(tutorialTimer)
  clearTimeout(briefingTimer)
})
</script>

<style>
.app-loading {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 100dvh;
  background: var(--gw-bg);
}

.fullscreen-content {
  width: 100vw;
  min-height: 100dvh;
}

/* Тонкий индикатор перехода между разделами (поверх всего, под тостами). */
.nav-progress {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  overflow: hidden;
  z-index: 12000;
  background: color-mix(in oklab, var(--color-primary) 16%, transparent);
  pointer-events: none;
}
.nav-progress-bar {
  position: absolute;
  top: 0;
  height: 100%;
  width: 40%;
  border-radius: 0 2px 2px 0;
  background: var(--color-primary);
  animation: navProgressSlide 1.1s ease-in-out infinite;
}
@keyframes navProgressSlide {
  0% { left: -40%; }
  100% { left: 100%; }
}
</style>
