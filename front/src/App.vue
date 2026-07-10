<template>
  <!-- has-unit-banner: мобильные fixed-экраны (мессенджер) сдвигаются под
       плашку активного юнита на --unit-banner-height. -->
  <div
    class="app-layout"
    :class="{ 'has-unit-banner': authStore.token && unitsStore.activeUnit && unitsStore.minimized }"
    :data-dark="themeStore.dark"
  >
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
      <div class="content-col">
        <ActiveUnitBanner v-if="unitsStore.activeUnit && unitsStore.minimized" />
        <main class="main-content">
          <router-view />
        </main>
      </div>
      <AppBottomNav />
      <ActiveUnitModal v-if="unitsStore.activeUnit && !unitsStore.minimized" />
      <AppTutorial v-if="isTutorialOpen" />
      <ChangelogModal v-if="isChangelogOpen" @close="closeChangelog" />
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
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import { useUnitsStore } from '@/stores/units.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { usePetsStore } from '@/stores/pets.js'
import { usePortalStore } from '@/stores/portal.js'
import { useAssistantStore } from '@/stores/assistant.js'
import { useCallStore } from '@/stores/call.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { useTutorial } from '@/composables/useTutorial.js'
import { useChangelog } from '@/composables/useChangelog.js'
import { connectSocket } from '@/socket/index.js'
import { navProgress } from '@/composables/useNavProgress.js'
import {
  registerNotifyServiceWorker, installNotifyUnlock, requestNotificationPermission,
  playNotifySound,
} from '@/utils/systemNotify.js'
import { installAppUpdateWatcher } from '@/utils/appUpdate.js'
import { initNativePush, syncNativeSystemBars } from '@/utils/nativeApp.js'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import AppBottomNav from '@/components/layout/AppBottomNav.vue'
import CompanyDisabledScreen from '@/components/layout/CompanyDisabledScreen.vue'
import ActiveUnitModal from '@/components/layout/ActiveUnitModal.vue'
import ActiveUnitBanner from '@/components/layout/ActiveUnitBanner.vue'
import AppTutorial from '@/components/layout/AppTutorial.vue'
import ChangelogModal from '@/components/layout/ChangelogModal.vue'
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
const petsStore = usePetsStore()
const portalStore = usePortalStore()
const assistantStore = useAssistantStore()
const callStore = useCallStore()
const notif = useNotificationsStore()
const route = useRoute()
const router = useRouter()
const { isMobile } = useBreakpoint()
const { usesGroove } = useCompanySettings()

const isFullscreenRoute = computed(() => !!route.meta?.fullscreen && !!authStore.user)
// isOpen деструктурирован как топ-левел ref — Vue auto-unwraps в шаблоне
const { isOpen: isTutorialOpen, open: openTutorial, shouldAutoShow } = useTutorial()
const { isOpen: isChangelogOpen, close: closeChangelog, checkForNewVersion } = useChangelog()
let tutorialTimer = null

watch(() => authStore.user, (user, prev) => {
  if (user && !prev && shouldAutoShow()) {
    clearTimeout(tutorialTimer)
    tutorialTimer = setTimeout(() => openTutorial(), 600)
  }
})

// Мобильная обёртка (Capacitor): после появления сессии регистрируем
// FCM-токен устройства; тап по системному пушу ведёт на адресный экран.
// В браузере/Electron initNativePush — no-op.
function openFromPush(data) {
  if (data.type === 'message' && data.conversation_id) {
    router.push(`/messenger/${data.conversation_id}`)
  } else if (data.type === 'task' && data.task_id) {
    router.push(`/tasks/${data.task_id}`)
  }
  // type=call: приложение открылось — входящий звонок подхватит WS.
}
watch(() => authStore.user, (user, prev) => {
  if (user && !prev) initNativePush(openFromPush)
})

// Мобильная обёртка: системные панели следуют теме — тёмная/светлая, смена
// пресета/палитры и тумблер градиента меняют фактический фон приложения.
// В браузере — no-op. flush:'post' обязателен: цвет резолвится из DOM, а
// [data-dark] на .app-layout обновляется только при перерисовке — иначе бар
// красится в прошлую тему (а при старте .app-layout ещё не существует вовсе).
watch(
  () => [themeStore.dark, themeStore.currentPreset, themeStore.bgGradient],
  () => syncNativeSystemBars(themeStore.dark),
  { immediate: true, flush: 'post', deep: true },
)

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
  // Слежение за новой версией приложения (SW + /api/changelog): тост о
  // доступном обновлении, мягкая перезагрузка при уходе вкладки в фон —
  // но не во время звонка.
  installAppUpdateWatcher({
    canReload: () => callStore.phase === 'idle',
    onUpdateAvailable: () => notif.notify({
      severity: 'info',
      summary: 'Доступно обновление',
      detail: 'Вышла новая версия приложения — страница обновится автоматически, когда вкладка будет неактивна.',
      life: 8000,
    }),
  })
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
    // Бейдж непрочитанных постов портала — только при активной компании.
    if (authStore.companyId != null) portalStore.fetchUnread()
    // Если страницу перезагрузили во время звонка — звонок ещё «жив» на
    // сервере (grace-окно). Предложим вернуться к нему.
    callStore.checkRejoin()
    // Лог версий показываем существующим пользователям; новичкам сначала тур,
    // а лог всплывёт при следующем входе.
    if (!shouldAutoShow()) {
      checkForNewVersion()
    }
  }
})

// Сброс данных при логауте, чтобы не утекли между сессиями.
watch(() => authStore.user, (user) => {
  if (!user) {
    messengerStore.reset()
    petsStore.reset()
    portalStore.reset()
    assistantStore.reset()
  }
})

// Смена активной компании — company-scoped сторы обнуляем; питомец нужен
// плавающему виджету сразу (он смонтирован постоянно и сам не перечитает).
watch(() => authStore.companyId, (id, prev) => {
  if (prev == null || id === prev) return
  petsStore.reset()
  portalStore.reset()
  assistantStore.reset()
  if (id != null && usesGroove.value) petsStore.fetchPet().catch(() => {})
  if (id != null) portalStore.fetchUnread()
})

/* Клик по системному уведомлению о звонке (через service worker) приходит
   сюда — фокусируем окно и разворачиваем CallView, если звонок уже принят
   и был свёрнут в мини-режим. Сам входящий overlay уже виден сам по себе
   (он на phase === 'incoming'). */
function onCallFocusOverlay() {
  if (callStore.isMinimized) callStore.expand()
}

/* Юнит в работе → закрытие/перезагрузка вкладки не проходит молча: браузер
   показывает нативный диалог «Покинуть сайт?» (свою модалку на beforeunload
   показать нельзя — только preventDefault), плюс звуковой «бип» как у
   уведомлений (сыграет, если AudioContext уже разогрет первым жестом). */
function onBeforeUnloadGuard(e) {
  if (!unitsStore.activeUnit) return
  playNotifySound()
  e.preventDefault()
  e.returnValue = '' // без returnValue старые Chrome диалог не показывают
}

onMounted(() => {
  window.addEventListener('call:focus-overlay', onCallFocusOverlay)
  window.addEventListener('beforeunload', onBeforeUnloadGuard)
})
onBeforeUnmount(() => {
  window.removeEventListener('call:focus-overlay', onCallFocusOverlay)
  window.removeEventListener('beforeunload', onBeforeUnloadGuard)
  clearTimeout(tutorialTimer)
})
</script>

<style>
/* Полный резерв под остров активного юнита на мобильном (верхний отступ 8px
   + плашка 54px) — синхронизирован с ActiveUnitBanner; мобильные fixed-экраны
   отступают на него сверху. */
.app-layout {
  --unit-banner-height: 62px;
}

/* Колонка «баннер активного юнита + контент»: баннер занимает свою высоту,
   .main-content сжимается под остаток и скроллится сам — без прокрутки шелла. */
.content-col {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

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
