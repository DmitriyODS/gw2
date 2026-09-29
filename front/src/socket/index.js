import { GatewaySocket } from '@/socket/gateway.js'
import { refreshSession } from '@/api/client.js'
import { useAuthStore } from '@/stores/auth.js'
import { isNativeApp } from '@/utils/nativeApp.js'
import { useUnitsStore } from '@/stores/units.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useCallStore } from '@/stores/call.js'
import { registerTaskSocketHandlers } from '@/socket/tasks.js'
import { registerMessengerSocketHandlers } from '@/socket/messenger.js'
import { registerCallSocketHandlers } from '@/socket/calls.js'
import { registerPetsSocketHandlers } from '@/socket/pets.js'
import { registerRegistrySocketHandlers } from '@/socket/registry.js'
import { registerFormsSocketHandlers } from '@/socket/forms.js'
import { registerCalendarSocketHandlers } from '@/socket/calendar.js'
import { registerDiarySocketHandlers } from '@/socket/diary.js'
import { registerScheduleSocketHandlers } from '@/socket/schedule.js'
import { registerNotesSocketHandlers } from '@/socket/notes.js'
import { registerBoardsSocketHandlers } from '@/socket/boards.js'
import { registerDriveSocketHandlers } from '@/socket/drive.js'
import { registerRemindersSocketHandlers } from '@/socket/reminders.js'
import { registerPortalSocketHandlers } from '@/socket/portal.js'
import { registerBillingSocketHandlers } from '@/socket/billing.js'

let socket = null
let visibilityHookInstalled = false
let heartbeatTimer = null
let resyncPromise = null
const HEARTBEAT_MS = 25_000

export function getSocket() {
  return socket
}

export function updateSocketAuth(token) {
  socket?.setToken(token)
}

/* Подтягивает состояние мессенджера с сервера. Используется при reconnect
   сокета и при возврате после долгой скрытости — закрывает дыру, если событие
   message:new/message:deleted/conversation:* потерялось при polling-обрыве
   или пока вкладка была в фоне. */
function resyncMessenger() {
  if (resyncPromise) return resyncPromise
  resyncPromise = (async () => {
    try {
      const messenger = useMessengerStore()
      await messenger.resyncConversations()
      if (messenger.activeConversationId) {
        await messenger.fetchMessages(messenger.activeConversationId)
      }
      await messenger.fetchUnreadCount()
      messenger.fetchChatBackgrounds()
    } catch {}
  })().finally(() => {
    resyncPromise = null
  })
  return resyncPromise
}

/* Когда вкладка снова в фокусе и открыт какой-то чат — сразу помечаем его
   прочитанным (пользователь его видит). Раньше прочтение происходило только
   при открытии чата, и сообщения, пришедшие пока вкладка была в фоне,
   оставались непрочитанными до клика. */
function markActiveReadOnFocus() {
  try {
    const messenger = useMessengerStore()
    if (messenger.activeConversationId
        && document.visibilityState === 'visible'
        && (typeof document.hasFocus !== 'function' || document.hasFocus())) {
      messenger.markRead(messenger.activeConversationId)
    }
  } catch {}
}

/* Сообщаем серверу о видимости вкладки — это драйвер онлайн-статуса.
   На мобильных дисконнект при сворачивании/блокировке приходит с задержкой
   или теряется, поэтому явный сигнал даёт точный last_seen и честный «в сети». */
function emitVisibility(visible) {
  // Фокус без смены видимости (alt-tab на столе) — не событие для сервера.
  if (visible && heartbeatTimer) return
  if (socket?.connected) {
    try { socket.emit('presence:visibility', { visible }) } catch {}
  }
  if (visible) startHeartbeat()
  else stopHeartbeat()
}

/* Heartbeat для presence: подтверждает серверу, что вкладка реально жива.
   Если на серверной стороне heartbeat не приходил дольше ~60с, sweep
   пометит соединение «не в сети». На мобильных это лечит долгие
   зависшие сокеты, которые иначе оставляют пользователя «онлайн». */
function sendHeartbeat() {
  if (!socket?.connected) return
  if (typeof document !== 'undefined' && document.visibilityState !== 'visible') return
  try { socket.emit('presence:heartbeat') } catch {}
}

function startHeartbeat() {
  stopHeartbeat()
  sendHeartbeat()
  heartbeatTimer = setInterval(sendHeartbeat, HEARTBEAT_MS)
}

function stopHeartbeat() {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer)
    heartbeatTimer = null
  }
}

/* Пересинхронизация — только когда события могли потеряться: после обрыва
   (её делает обработчик connect) и после долгой скрытости — мобильный браузер
   замораживает фоновую вкладку, и «подключённый» сокет мог молча пропустить
   кадры. Возврат фокуса при живом сокете (alt-tab на столе) ничего не теряет,
   а стоил четыре запроса на каждое переключение окна. */
const RESYNC_AFTER_HIDDEN_MS = 30_000
let hiddenAt = 0

/* Сон сокета Android-обёртки в фоне. Пинги шлюза каждые 25 с будят радио
   телефона, хотя свёрнутому приложению события и так приходят пушами: скрытая
   вкладка для presence офлайн, и pushsvc шлёт такому получателю FCM (звонок —
   тоже). Возврат будит сокет, а обработчик connect сверяет пропущенное, в том
   числе звонок, который ещё звонит. Рабочему столу сон не положен: в трее
   Electron события доходят только через сокет. */
const SLEEP_AFTER_HIDDEN_MS = 3 * 60_000
let sleepTimer = null

function scheduleSleep(visible) {
  if (!isNativeApp()) return
  clearTimeout(sleepTimer)
  sleepTimer = null
  if (visible) {
    socket?.wake()
    return
  }
  sleepTimer = setTimeout(() => socket?.sleep(), SLEEP_AFTER_HIDDEN_MS)
}

function installVisibilityResync() {
  if (visibilityHookInstalled || typeof document === 'undefined') return
  visibilityHookInstalled = true
  document.addEventListener('visibilitychange', () => {
    const visible = document.visibilityState === 'visible'
    emitVisibility(visible)
    scheduleSleep(visible)
    if (!visible) {
      hiddenAt = Date.now()
      return
    }
    if (!socket?.connected) return
    if (hiddenAt && Date.now() - hiddenAt > RESYNC_AFTER_HIDDEN_MS) resyncMessenger()
    markActiveReadOnFocus()
  })
  window.addEventListener('focus', () => {
    emitVisibility(true)
    if (socket?.connected) markActiveReadOnFocus()
  })
  // pagehide — последний надёжный момент на мобильных, чтобы пометить «ушёл».
  window.addEventListener('pagehide', () => emitVisibility(false))
  // Сеть вернулась после обрыва: фоном перечитываем список диалогов и ПЕРВУЮ
  // страницу активного чата по HTTP — не полагаясь на состояние сокета (после
  // кратковременного обрыва WS может «залипнуть» и не прислать connect, из-за
  // чего в ленте оставалось лишь последнее дошедшее сообщение).
  window.addEventListener('online', () => {
    resyncMessenger()
    markActiveReadOnFocus()
  })
}

export function connectSocket() {
  const auth = useAuthStore()
  if (!auth.token || socket) return socket

  // В dev подключаемся напрямую к gatewaysvc (:8096), минуя Vite proxy;
  // хост берём из адреса страницы (не хардкодим localhost) — тогда заход с
  // другого устройства по http://<IP-машины>:5173 даёт ws://<IP-машины>:8096.
  // В проде — /ws через nginx (схема по текущему протоколу страницы).
  const target = import.meta.env.DEV
    ? `ws://${window.location.hostname}:8096/ws`
    : (window.location.protocol === 'https:' ? 'wss://' : 'ws://')
      + window.location.host + '/ws'
  socket = new GatewaySocket(target, {
    auth: { token: auth.token },
    // Протухший access: обновляем так же, как HTTP-клиент на 401. Сессию
    // отозвали (4xx) — выходим, иначе сокет пробовал бы вечно.
    refreshAuth: async () => {
      try {
        return (await refreshSession()).access_token
      } catch (e) {
        const status = e?.status ?? 0
        if (status >= 400 && status < 500) useAuthStore().clearAuth()
        throw e
      }
    },
  })

  installVisibilityResync()

  let hadConnected = false

  socket.on('connect', () => {
    // Свежий снимок онлайн-статусов при каждом (пере)подключении — события
    // presence:update, прошедшие до коннекта, мы не услышали.
    try { useMessengerStore().fetchPresence() } catch {}
    // Heartbeat presence — пока вкладка видима, шлём пинг каждые 25с;
    // sweep на сервере опускает «зависшие» сокеты в офлайн через 60с.
    if (typeof document === 'undefined' || document.visibilityState === 'visible') {
      startHeartbeat()
    }
    // При переподключении локальное состояние могло разойтись с сервером
    // (пропущенные события за время обрыва) — перечитываем активный юнит,
    // список диалогов и сообщения активного чата.
    if (hadConnected) {
      const units = useUnitsStore()
      units.fetchActiveUnit()
      resyncMessenger()
    }
    hadConnected = true
    // Синхронизируем состояние звонка с сервером: лечит зависший phase после
    // обрыва (пропущенный call:ended) и предлагает вернуться к звонку, если он
    // ещё идёт на сервере.
    try { useCallStore().checkRejoin() } catch {}
  })

  socket.on('connect_error', (err) => {
    console.warn('Socket connection error:', err.message)
  })

  socket.on('disconnect', () => { stopHeartbeat() })

  registerTaskSocketHandlers(socket)
  registerMessengerSocketHandlers(socket)
  registerCallSocketHandlers(socket)
  registerPetsSocketHandlers(socket)
  registerRegistrySocketHandlers(socket)
  registerFormsSocketHandlers(socket)
  registerCalendarSocketHandlers(socket)
  registerDiarySocketHandlers(socket)
  registerScheduleSocketHandlers(socket)
  registerNotesSocketHandlers(socket)
  registerBoardsSocketHandlers(socket)
  registerDriveSocketHandlers(socket)
  registerRemindersSocketHandlers(socket)
  registerPortalSocketHandlers(socket)
  registerBillingSocketHandlers(socket)
}

export function disconnectSocket() {
  stopHeartbeat()
  clearTimeout(sleepTimer)
  sleepTimer = null
  resyncPromise = null
  if (socket) {
    socket.disconnect()
    socket = null
  }
}
