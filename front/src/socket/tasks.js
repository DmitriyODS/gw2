import { useAuthStore } from '@/stores/auth.js'
import { useTasksStore } from '@/stores/tasks.js'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'

export function registerTaskSocketHandlers(socket) {
  // refreshMyActiveCount — бейдж «моих» активных задач в навигации: любое
  // событие задач могло изменить счётчик (дебаунс внутри стора).
  socket.on('task:created', (task) => {
    const tasks = useTasksStore()
    tasks.addTaskFromSocket(task)
    tasks.refreshMyActiveCount()
  })

  socket.on('task:updated', (data) => {
    const tasks = useTasksStore()
    tasks.patchTask(data)
    tasks.refreshMyActiveCount()
  })

  socket.on('task:archived', ({ task_id, archived_at }) => {
    const tasks = useTasksStore()
    tasks.archiveTask(task_id, archived_at)
    tasks.refreshMyActiveCount()
  })

  socket.on('task:restored', ({ task_id }) => {
    const tasks = useTasksStore()
    tasks.restoreTask(task_id)
    tasks.refreshMyActiveCount()
  })

  socket.on('task:deleted', ({ task_id }) => {
    const tasks = useTasksStore()
    tasks.removeTask(task_id)
    tasks.refreshMyActiveCount()
  })

  socket.on('comment:new', (payload) => {
    useTasksStore().applyCommentSocket('new', payload)
  })

  socket.on('comment:updated', (payload) => {
    useTasksStore().applyCommentSocket('updated', payload)
  })

  socket.on('comment:deleted', (payload) => {
    useTasksStore().applyCommentSocket('deleted', payload)
  })

  socket.on('unit:started', (unit) => {
    const units = useUnitsStore()
    const auth = useAuthStore()
    if (unit.user_id === auth.user?.id) units.setActiveUnit(unit)

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

  socket.on('unit:stopped', ({ unit_id, task_id, user_id }) => {
    const units = useUnitsStore()
    if (units.activeUnit?.id === unit_id) units.clearActiveUnit()
    if (task_id && user_id) useTasksStore().removeActiveUser(task_id, user_id)
  })

  socket.on('unit:updated', (data) => {
    const units = useUnitsStore()
    if (units.activeUnit?.id === data.unit_id) {
      units.setActiveUnit({ ...units.activeUnit, ...data })
    }
  })

  socket.on('unit:deleted', ({ unit_id, task_id, user_id }) => {
    const units = useUnitsStore()
    if (units.activeUnit?.id === unit_id) units.clearActiveUnit()
    if (task_id && user_id) useTasksStore().removeActiveUser(task_id, user_id)
  })

  socket.on('unit:force_stopped', ({ unit_id, stopped_by_fio }) => {
    const units = useUnitsStore()
    if (units.activeUnit?.id === unit_id) {
      units.clearActiveUnit()
      useNotificationsStore().warn(`Ваш юнит был остановлен пользователем ${stopped_by_fio}`)
    }
  })
}
