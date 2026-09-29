import { useAuthStore } from '@/stores/auth.js'
import { useTasksStore } from '@/stores/tasks.js'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { pushNotification } from '@/composables/useDesktopNotifications.js'

export function registerTaskSocketHandlers(socket) {
  // Бейдж «моих» активных задач пересчитывается, только если событие его
  // касается (affectsMyCount — до применения события, дебаунс внутри стора).
  const onTaskEvent = (apply) => (payload) => {
    const tasks = useTasksStore()
    const mine = tasks.affectsMyCount(payload)
    apply(tasks, payload)
    if (mine) tasks.refreshMyActiveCount()
  }

  socket.on('task:created', onTaskEvent((tasks, task) => tasks.addTaskFromSocket(task)))
  socket.on('task:updated', onTaskEvent((tasks, data) => tasks.patchTask(data)))
  socket.on('task:archived', onTaskEvent((tasks, { task_id, archived_at }) => tasks.archiveTask(task_id, archived_at)))
  socket.on('task:restored', onTaskEvent((tasks, { task_id }) => tasks.restoreTask(task_id)))
  socket.on('task:deleted', onTaskEvent((tasks, { task_id }) => tasks.removeTask(task_id)))

  socket.on('comment:new', (payload) => {
    useTasksStore().applyCommentSocket('new', payload)
  })

  socket.on('comment:updated', (payload) => {
    useTasksStore().applyCommentSocket('updated', payload)
  })

  socket.on('comment:deleted', (payload) => {
    useTasksStore().applyCommentSocket('deleted', payload)
  })

  // Меня упомянули в комментарии (событие адресовано только мне) — бейдж на
  // карточке задачи + ненавязчивый тост.
  socket.on('task:mention', ({ task_id }) => {
    if (task_id == null) return
    useTasksStore().bumpMention(task_id)
    useNotificationsStore().notify({
      severity: 'info',
      summary: 'Упоминание',
      detail: 'Вас отметили в комментарии к задаче',
      source: 'tasks',
    })
    pushNotification({
      key: `mention-${task_id}`,
      icon: 'alternate_email',
      title: 'Упоминание',
      text: 'Вас отметили в комментарии к задаче',
      path: `/tasks/${task_id}`,
      source: 'tasks',
    })
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
