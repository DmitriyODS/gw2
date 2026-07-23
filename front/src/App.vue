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
      <BrandLoader />
      <p v-if="authStore.connecting" class="app-loading-hint">
        Нет соединения с сервером — подключимся, как только появится сеть
      </p>
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
    <!-- Pull-to-refresh на мобиле (обновление страницы оттяжкой вниз у верха
         экрана). Отключён на fullscreen-роутах — там вертикальные жесты заняты. -->
    <PullToRefresh :active="!!authStore.token && !isFullscreenRoute && callStore.phase === 'idle'" />
    <!-- Выбор получателя для текста из системного «Поделиться» (Android). -->
    <NewChatDialog v-if="authStore.token" v-model="sharePickOpen" @pick="onSharePickRecipient" />
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
  playNotifySound, focusAppWindow,
} from '@/utils/systemNotify.js'
import { installAppUpdateWatcher } from '@/utils/appUpdate.js'
import { initNativePush, syncNativeSystemBars, getSharedPayload } from '@/utils/nativeApp.js'
import {
  startCallService, stopCallService, setCallProximity, setCallShowOverLock,
  audioStart, audioStop,
} from '@/utils/nativeApp.js'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import AppBottomNav from '@/components/layout/AppBottomNav.vue'
import CompanyDisabledScreen from '@/components/layout/CompanyDisabledScreen.vue'
import ActiveUnitModal from '@/components/layout/ActiveUnitModal.vue'
import ActiveUnitBanner from '@/components/layout/ActiveUnitBanner.vue'
import AppTutorial from '@/components/layout/AppTutorial.vue'
import ChangelogModal from '@/components/layout/ChangelogModal.vue'
import MiniMessenger from '@/components/messenger/MiniMessenger.vue'
import PullToRefresh from '@/components/common/PullToRefresh.vue'
import NewChatDialog from '@/components/messenger/NewChatDialog.vue'
import IncomingCallOverlay from '@/components/call/IncomingCallOverlay.vue'
import CallView from '@/components/call/CallView.vue'
import ReturnCallBanner from '@/components/call/ReturnCallBanner.vue'
import Toast from 'primevue/toast'
import BrandLoader from '@/components/common/BrandLoader.vue'

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

// Десктоп-обёртка: счётчик непрочитанных на иконке приложения
// (док/панель задач/трей). В браузере GrooveDesktop нет — no-op.
watch(() => (authStore.user ? messengerStore.totalUnread : 0), (n) => {
  window.GrooveDesktop?.setBadge?.(n)
}, { immediate: true })

// Мобильная обёртка (Capacitor): после появления сессии регистрируем
// FCM-токен устройства; тап по системному пушу ведёт на адресный экран.
// В браузере/Electron initNativePush — no-op.
function openFromPush(data) {
  if (data.type === 'message' && data.conversation_id) {
    router.push(`/messenger/${data.conversation_id}`)
  } else if (data.type === 'task' && data.task_id) {
    router.push(`/tasks/${data.task_id}`)
  } else if (data.type === 'post' && data.post_id) {
    router.push(`/portal/${data.post_id}`)
  } else if (data.type === 'kudos') {
    router.push('/pets/bank')
  }
  // type=call: приложение открылось — входящий звонок подхватит WS.
}

/* Клик по системному уведомлению (десктоп-обёртка Electron и веб через SW)
   переносит В РАЗДЕЛ, откуда оно пришло — для сообщения открывает нужный чат
   из ЛЮБОГО экрана. Оба пути (SW-postMessage и Electron new Notification)
   сходятся на событии messenger:open-conversation — здесь единый глобальный
   роутинг, не зависящий от того, смонтирован ли MessengerView. */
function onOpenConversation(e) {
  const id = e.detail?.conversation_id
  if (!id) return
  focusAppWindow()
  router.push(`/messenger/${id}`)
}

/* Системное «Поделиться» (Android-обёртка): текст И/ИЛИ любые файлы (в т.ч.
   несколько). Pull-модель — забираем контент у нативки, когда сессия готова
   (надёжно к холодному старту), открываем выбор получателя, затем чат с
   вложениями и готовым текстом (как в Telegram). */
const sharePickOpen = ref(false)
const sharedText = ref('')
const sharedFiles = ref([])

function b64ToFile(f) {
  const bin = atob(f.data)
  const arr = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) arr[i] = bin.charCodeAt(i)
  return new File([arr], f.name || 'файл', { type: f.mimeType || 'application/octet-stream' })
}

let pullingShare = false
async function pullShare() {
  if (pullingShare || !authStore.token) return
  pullingShare = true
  try {
    const payload = await getSharedPayload()
    if (!payload) return
    const text = (payload.text || '').trim()
    const raw = Array.isArray(payload.files) ? payload.files : []
    const tooBig = raw.filter(f => f?.tooLarge)
    const files = raw.filter(f => f?.data).map(b64ToFile)
    if (tooBig.length) {
      notif.error(`Не отправить (больше 500 МБ): ${tooBig.map(f => f.name).join(', ')}`)
    }
    if (!text && !files.length) return
    sharedText.value = text
    sharedFiles.value = files
    sharePickOpen.value = true
  } finally {
    pullingShare = false
  }
}

async function onSharePickRecipient(user) {
  sharePickOpen.value = false
  try {
    const id = await messengerStore.openWith(user.id)
    // Текст + файлы подхватит MessengerView при активации чата (загрузит
    // вложения существующим механизмом, останется написать и отправить).
    messengerStore.pendingDraft = { convId: id, text: sharedText.value, files: sharedFiles.value }
    router.push(`/messenger/${id}`)
  } catch { /* ошибка открытия чата — молча */ }
  sharedText.value = ''
  sharedFiles.value = []
}
watch(() => authStore.user, (user, prev) => {
  if (user && !prev) {
    initNativePush(openFromPush)
    pullShare() // сессия готова — забираем контент из «Поделиться», если он ждёт
  }
})

// Мобильная обёртка: системные панели следуют теме — тёмная/светлая, смена
// пресета/палитры и тумблер градиента меняют фактический фон приложения.
// В браузере — no-op. flush:'post' обязателен: цвет резолвится из DOM, а
// [data-dark] на .app-layout обновляется только при перерисовке — иначе бар
// красится в прошлую тему (а при старте .app-layout ещё не существует вовсе).
watch(
  () => [themeStore.dark, themeStore.currentPreset, themeStore.bgGradient],
  () => syncNativeSystemBars(),
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
    // Личное оформление чатов (градиенты/узоры) — синхронно с бэкендом.
    messengerStore.fetchChatBackgrounds()
    // Бейдж непрочитанных постов портала — только при активной компании.
    if (authStore.companyId != null) portalStore.fetchUnread()
    // Личное оформление ленты портала (градиент/узор/картинка) — синк с бэкендом.
    portalStore.fetchBackground()
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
  // Активный юнит покидаемой компании завершается в switchCompany; здесь
  // пересинхронизируем баннер с бэкендом (страховка на случай, если стоп не
  // прошёл, и на будущее — если появятся юниты, специфичные для компании).
  unitsStore.fetchActiveUnit().catch(() => {})
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

/* Мобильная обёртка: жизнь звонка при блокировке экрана. Foreground-сервис
   держит соединение живым, когда экран гаснет/в кармане (приложение НЕ
   закрывается). Экран НЕ держим принудительно включённым: у уха он гаснет по
   датчику приближения (аудио-звонок), а блокировка гасит его штатно. Показ
   поверх локскрина — только для входящего. В браузере/Electron — no-op. */
let callAudioOn = false
watch(() => [callStore.phase, callStore.media], ([phase, media]) => {
  if (phase === 'incoming') {
    setCallShowOverLock(true)
    return
  }
  if (phase === 'active' || phase === 'outgoing') {
    setCallShowOverLock(false)             // в разговоре блокировка гасит экран штатно
    setCallProximity(media === 'audio')    // аудио-звонок: экран гаснет у уха
    startCallService()
    if (!callAudioOn) { callAudioOn = true; audioStart() }
  } else { // idle
    setCallShowOverLock(false)
    setCallProximity(false)
    if (callAudioOn) { callAudioOn = false; audioStop() }
    stopCallService()
  }
})

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
  window.addEventListener('messenger:open-conversation', onOpenConversation)
  window.addEventListener('gw:share-available', pullShare)
  // Холодный старт: нативка могла выставить флаг до навешивания слушателя —
  // и в любом случае буфер шаринга ждёт в плагине, заберём его при готовности.
  if (window.__gwShareAvailable) { window.__gwShareAvailable = false; pullShare() }
})
onBeforeUnmount(() => {
  window.removeEventListener('call:focus-overlay', onCallFocusOverlay)
  window.removeEventListener('beforeunload', onBeforeUnloadGuard)
  window.removeEventListener('messenger:open-conversation', onOpenConversation)
  window.removeEventListener('gw:share-available', pullShare)
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
  flex-direction: column;
  gap: 16px;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 100dvh;
  background: var(--gw-bg);
}

.app-loading-hint {
  margin: 0;
  padding: 0 24px;
  font-size: 14px;
  color: var(--color-text-dim);
  text-align: center;
}

.fullscreen-content {
  /* 100% (не 100vw): vw включает ширину скроллбара и даёт горизонтальную
     полосу прокрутки/полоску фона у правого края. */
  width: 100%;
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
