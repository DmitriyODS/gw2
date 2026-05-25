import { io } from 'socket.io-client'
import { useAuthStore } from '@/stores/auth.js'
import { useTasksStore } from '@/stores/tasks.js'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { showSystemNotification, playNotifySound } from '@/utils/systemNotify.js'

let socket = null

export function connectSocket() {
  const auth = useAuthStore()
  if (!auth.token || socket?.connected) return

  socket = io('/', {
    auth: { token: auth.token },
    transports: ['polling', 'websocket'],
    upgrade: !import.meta.env.DEV,
  })

  let hadConnected = false

  socket.on('connect', () => {
    // При переподключении локальное состояние могло разойтись с сервером
    // (пропущенные события за время обрыва) — перечитываем активный юнит.
    if (hadConnected) {
      const units = useUnitsStore()
      units.fetchActiveUnit()
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
        showSystemNotification(fio, body, () => {
          window.focus()
          window.dispatchEvent(new CustomEvent('messenger:open-conversation', { detail: { conversation_id } }))
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
