import { io } from 'socket.io-client'
import { useAuthStore } from '@/stores/auth.js'
import { useTasksStore } from '@/stores/tasks.js'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useCallStore } from '@/stores/call.js'
import {
  showSystemNotification, showCallNotification, closeCallNotification, playNotifySound,
} from '@/utils/systemNotify.js'

let socket = null
let visibilityHookInstalled = false

export function getSocket() {
  return socket
}

/* Подтягивает состояние мессенджера с сервера. Используется при reconnect
   сокета и при возврате вкладки в фокус — закрывает дыру, если событие
   message:new/message:deleted/conversation:* потерялось при polling-обрыве
   или пока вкладка была в фоне. */
function resyncMessenger() {
  try {
    const messenger = useMessengerStore()
    messenger.fetchConversations()
    if (messenger.activeConversationId) {
      messenger.fetchMessages(messenger.activeConversationId)
    }
    messenger.fetchUnreadCount()
  } catch {}
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
  if (socket?.connected) {
    try { socket.emit('presence:visibility', { visible }) } catch {}
  }
}

function installVisibilityResync() {
  if (visibilityHookInstalled || typeof document === 'undefined') return
  visibilityHookInstalled = true
  document.addEventListener('visibilitychange', () => {
    const visible = document.visibilityState === 'visible'
    emitVisibility(visible)
    if (visible && socket?.connected) {
      resyncMessenger()
      markActiveReadOnFocus()
    }
  })
  window.addEventListener('focus', () => {
    emitVisibility(true)
    if (socket?.connected) {
      resyncMessenger()
      markActiveReadOnFocus()
    }
  })
  // pagehide — последний надёжный момент на мобильных, чтобы пометить «ушёл».
  window.addEventListener('pagehide', () => emitVisibility(false))
}

export function connectSocket() {
  const auth = useAuthStore()
  if (!auth.token || socket?.connected) return

  // В dev подключаемся напрямую к Flask (5001), минуя Vite proxy.
  // Порядок transports важен: ['polling', 'websocket'] = стандартный socket.io
  // flow — сначала HTTP polling устанавливает sid, затем upgrade на WS.
  // Прямой WS без polling часто фейлится handshake'ом.
  const target = import.meta.env.DEV ? 'http://localhost:5001' : '/'
  socket = io(target, {
    auth: { token: auth.token },
    transports: ['polling', 'websocket'],
    upgrade: true,
    reconnection: true,
    reconnectionAttempts: Infinity,
    reconnectionDelay: 1000,
    reconnectionDelayMax: 5000,
  })

  installVisibilityResync()

  let hadConnected = false

  socket.on('connect', () => {
    // Свежий снимок онлайн-статусов при каждом (пере)подключении — события
    // presence:update, прошедшие до коннекта, мы не услышали.
    try { useMessengerStore().fetchPresence() } catch {}
    // При переподключении локальное состояние могло разойтись с сервером
    // (пропущенные события за время обрыва) — перечитываем активный юнит,
    // список диалогов и сообщения активного чата.
    if (hadConnected) {
      const units = useUnitsStore()
      units.fetchActiveUnit()
      resyncMessenger()
    }
    hadConnected = true
  })

  socket.on('connect_error', (err) => {
    console.warn('Socket connection error:', err.message)
  })

  socket.on('disconnect', () => {})

  socket.on('task:created', (task) => {
    const tasks = useTasksStore()
    tasks.addTaskFromSocket(task)
  })

  socket.on('task:updated', (data) => {
    const tasks = useTasksStore()
    tasks.patchTask(data)
  })

  socket.on('task:archived', ({ task_id, archived_at }) => {
    const tasks = useTasksStore()
    tasks.archiveTask(task_id, archived_at)
  })

  socket.on('task:restored', ({ task_id }) => {
    const tasks = useTasksStore()
    tasks.restoreTask(task_id)
  })

  socket.on('task:deleted', ({ task_id }) => {
    const tasks = useTasksStore()
    tasks.removeTask(task_id)
  })

  socket.on('unit:started', (unit) => {
    const units = useUnitsStore()
    const auth = useAuthStore()
    if (unit.user_id === auth.user?.id) {
      units.setActiveUnit(unit)
    }
    const tasks = useTasksStore()
    tasks.patchTask({ id: unit.task_id, has_units: true })
    if (unit.user) {
      tasks.addActiveUser(unit.task_id, {
        id: unit.user.id,
        fio: unit.user.fio,
        avatar_path: unit.user.avatar_path ?? null,
      })
    }
  })

  socket.on('unit:stopped', ({ unit_id, task_id, user_id, datetime_end }) => {
    const units = useUnitsStore()
    if (units.activeUnit?.id === unit_id) {
      units.clearActiveUnit()
    }
    if (task_id && user_id) {
      const tasks = useTasksStore()
      tasks.removeActiveUser(task_id, user_id)
    }
  })

  socket.on('unit:updated', (data) => {
    const units = useUnitsStore()
    if (units.activeUnit?.id === data.unit_id) {
      units.setActiveUnit({ ...units.activeUnit, ...data })
    }
  })

  socket.on('unit:deleted', ({ unit_id, task_id, user_id }) => {
    const units = useUnitsStore()
    if (units.activeUnit?.id === unit_id) {
      units.clearActiveUnit()
    }
    if (task_id && user_id) {
      const tasks = useTasksStore()
      tasks.removeActiveUser(task_id, user_id)
    }
  })

  socket.on('message:new', ({ conversation_id, message, from_user_id }) => {
    const messenger = useMessengerStore()
    const authS = useAuthStore()
    const fromMe = from_user_id === authS.user?.id
    messenger.applyIncomingMessage(conversation_id, message, fromMe)

    if (!fromMe) {
      const isActive = messenger.activeConversationId === conversation_id
                       && document.visibilityState === 'visible'
                       && document.hasFocus()
      if (!isActive) {
        const conv = messenger.conversations.find(c => c.id === conversation_id)
        const fio = conv?.other_user?.fio || 'Сотрудник'
        const body = message.text || (message.attachments?.length ? 'Прислал(а) вложение' : 'Новое сообщение')
        showSystemNotification(fio, body, {
          data: { conversation_id },
          onClick: () => {
            window.focus()
            window.dispatchEvent(new CustomEvent('messenger:open-conversation', { detail: { conversation_id } }))
          },
        })
        playNotifySound()
      }
    }
  })

  socket.on('message:read', ({ conversation_id, reader_id }) => {
    const messenger = useMessengerStore()
    messenger.applyReadReceipt(conversation_id, reader_id)
  })

  socket.on('message:deleted', ({ conversation_id, message_id }) => {
    const messenger = useMessengerStore()
    messenger.applyMessageDeleted(conversation_id, message_id)
  })

  socket.on('conversation:deleted', ({ conversation_id }) => {
    const messenger = useMessengerStore()
    messenger.applyConversationDeleted(conversation_id)
  })

  socket.on('conversation:pin', ({ conversation_id, is_pinned }) => {
    const messenger = useMessengerStore()
    messenger.applyPinChange(conversation_id, is_pinned)
  })

  socket.on('presence:update', (p) => {
    useMessengerStore().applyPresence(p)
  })

  // ── Звонки ────────────────────────────────────────────────────
  socket.on('call:incoming', (call) => {
    // Диагностический лог — видно в DevTools Console у получателя. Если
    // лога нет — пакет не дошёл (получатель не в комнате `user_{id}`).
    console.info('[gw2 call] incoming', call)
    const callStore = useCallStore()
    // Если уже в звонке, store сам пошлёт decline — уведомление не нужно.
    if (callStore.phase !== 'idle') {
      callStore.handleIncoming(call)
      return
    }
    callStore.handleIncoming(call)
    // Системное OS-уведомление о входящем звонке — показываем ВСЕГДА (даже
    // когда вкладка в фокусе, потому что overlay появляется через event loop
    // и пользователь мог быть в другой панели/мониторе). Уведомление с
    // действиями «Принять» / «Отклонить», тег `gw2-call` — чтобы новое
    // входящее перезаписало старое.
    const initiator = call.participants?.find(p => p.role === 'initiator')
    const initiatorName = initiator?.fio || 'Сотрудник'
    const mediaText = call.media === 'audio' ? 'аудиозвонок' : 'видеозвонок'
    showCallNotification(
      `Входящий ${mediaText}`,
      `${initiatorName} звонит вам`,
      {
        callId: call.id,
        onClick: () => window.focus?.(),
      },
    )
    // Короткий «бип» подстраховывает рингтон в overlay: если AudioContext
    // ещё не разогрет жестом, хотя бы вот этот звук пройдёт после первого
    // взаимодействия пользователя (installNotifyUnlock).
    playNotifySound()
  })

  socket.on('call:started', (call) => {
    useCallStore().handleStarted(call)
  })

  socket.on('call:accepted', (data) => {
    closeCallNotification()
    useCallStore().handleAccepted(data)
  })

  socket.on('call:participant-joined', (data) => {
    useCallStore().handleParticipantJoined(data)
  })

  socket.on('call:participant-left', (data) => {
    useCallStore().handleParticipantLeft(data)
  })

  socket.on('call:participant-declined', (data) => {
    useCallStore().handleParticipantDeclined(data)
  })

  socket.on('call:ended', () => {
    closeCallNotification()
    useCallStore().handleEnded()
  })

  socket.on('webrtc:signal', (data) => {
    useCallStore().handleSignal(data)
  })

  socket.on('call:media-state', (data) => {
    useCallStore().handleMediaState(data)
  })

  socket.on('call:error', (data) => {
    closeCallNotification()
    useCallStore().handleError(data)
  })

  socket.on('unit:force_stopped', ({ unit_id, stopped_by_fio }) => {
    const units = useUnitsStore()
    const notif = useNotificationsStore()
    if (units.activeUnit?.id === unit_id) {
      units.clearActiveUnit()
      notif.warn(`Ваш юнит был остановлен пользователем ${stopped_by_fio}`)
    }
  })
}

export function disconnectSocket() {
  if (socket) {
    socket.disconnect()
    socket = null
  }
}
