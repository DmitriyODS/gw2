import { io } from 'socket.io-client'
import { useAuthStore } from '@/stores/auth.js'
import { useTasksStore } from '@/stores/tasks.js'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'

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
