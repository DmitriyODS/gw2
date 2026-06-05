import { defineStore } from 'pinia'
import { ref, reactive, watch } from 'vue'
import * as tasksApi from '@/api/tasks.js'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'

const STORAGE_KEY = 'gw2_tasks_filters'

function loadSavedFilters() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return JSON.parse(raw)
  } catch {}
  return {}
}

export const useTasksStore = defineStore('tasks', () => {
  const tasks = ref([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref(null)
  const activeTask = ref(null)

  const saved = loadSavedFilters()

  const filters = reactive({
    tab: saved.tab ?? 'active',
    search: saved.search ?? '',
    sort: saved.sort ?? 'last_activity',
    dept_id: saved.dept_id ?? null,
    stage_id: saved.stage_id ?? null,
    responsible_id: saved.responsible_id ?? null,
    received_from: saved.received_from ?? null,
    received_to: saved.received_to ?? null,
    has_units: saved.has_units ?? null,
    period_preset: saved.period_preset ?? null,
    page: 1,
    per_page: 30,
  })

  // Карта комментариев: task_id → массив комментариев (упорядочены по created_at).
  const commentsByTask = reactive({})
  // Карта контрибьюторов: task_id → массив { id, fio, avatar_path }.
  const contributorsByTask = reactive({})

  watch(filters, () => {
    // eslint-disable-next-line no-unused-vars
    const { page, per_page, ...toSave } = { ...filters }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(toSave))
  }, { deep: true })

  function _hasCompanyScope() {
    const auth = useAuthStore()
    if (auth.companyId != null) return true
    const companies = useCompaniesStore()
    return companies.activeCompanyId != null
  }

  async function fetchTasks({ silent = false } = {}) {
    if (!_hasCompanyScope()) {
      tasks.value = []
      total.value = 0
      return
    }
    if (!silent) loading.value = true
    error.value = null
    try {
      const params = {}
      if (filters.tab) params.tab = filters.tab
      if (filters.search) params.search = filters.search
      if (filters.sort) params.sort = filters.sort
      if (filters.dept_id) params.dept_id = filters.dept_id
      if (filters.stage_id) params.stage_id = filters.stage_id
      if (filters.responsible_id) params.responsible_id = filters.responsible_id
      if (filters.received_from) params.received_from = filters.received_from
      if (filters.received_to) params.received_to = filters.received_to
      if (filters.has_units) params.has_units = filters.has_units
      params.page = filters.page
      params.per_page = filters.per_page

      const data = await tasksApi.getTasks(params)
      tasks.value = data.tasks ?? data.items ?? data
      total.value = data.total ?? tasks.value.length
    } catch (e) {
      error.value = e.message || 'Ошибка загрузки задач'
      throw e
    } finally {
      if (!silent) loading.value = false
    }
  }

  function setFilter(key, value) {
    filters[key] = value
    if (key !== 'page') filters.page = 1
    fetchTasks().catch(() => {})
  }

  function setTab(tab) {
    filters.tab = tab
    filters.page = 1
    fetchTasks().catch(() => {})
  }

  // Сбрасывает сортировки и фильтры (поиск/вкладку не трогаем — это другая
  // ось состояния) к значениям по умолчанию.
  function resetFilters() {
    filters.sort = 'last_activity'
    filters.dept_id = null
    filters.stage_id = null
    filters.responsible_id = null
    filters.received_from = null
    filters.received_to = null
    filters.has_units = null
    filters.period_preset = null
    filters.page = 1
    fetchTasks().catch(() => {})
  }

  // === v3: ответственный, этап, контрибьюторы, комментарии ===
  async function assignResponsible(taskId, userId) {
    const updated = await tasksApi.setTaskResponsible(taskId, userId)
    patchTask(updated)
    return updated
  }

  async function setStage(taskId, stageId) {
    const updated = await tasksApi.setTaskStage(taskId, stageId)
    patchTask(updated)
    return updated
  }

  // Оптимистичный drag-drop между колонками канбана.
  async function dragMoveStage(taskId, newStageId) {
    const idx = tasks.value.findIndex((t) => t.id === taskId)
    const prevStageId = idx >= 0 ? tasks.value[idx].stage_id : null
    patchTask({ id: taskId, stage_id: newStageId })
    try {
      await tasksApi.setTaskStage(taskId, newStageId)
    } catch (e) {
      patchTask({ id: taskId, stage_id: prevStageId })
      throw e
    }
  }

  async function loadComments(taskId) {
    const data = await tasksApi.listTaskComments(taskId)
    commentsByTask[taskId] = data.items || []
    return commentsByTask[taskId]
  }

  async function addComment(taskId, text) {
    const created = await tasksApi.createTaskComment(taskId, text)
    if (!commentsByTask[taskId]) commentsByTask[taskId] = []
    if (!commentsByTask[taskId].find((c) => c.id === created.id)) {
      commentsByTask[taskId].push(created)
    }
    return created
  }

  async function editComment(taskId, commentId, text) {
    const updated = await tasksApi.updateTaskComment(taskId, commentId, text)
    const list = commentsByTask[taskId] || []
    const i = list.findIndex((c) => c.id === commentId)
    if (i >= 0) list[i] = { ...list[i], ...updated }
    return updated
  }

  async function deleteComment(taskId, commentId) {
    await tasksApi.deleteTaskComment(taskId, commentId)
    commentsByTask[taskId] = (commentsByTask[taskId] || []).filter((c) => c.id !== commentId)
  }

  // Применить сокет-событие комментария (приходит из socket/index.js).
  function applyCommentSocket(kind, payload) {
    if (!payload) return
    const taskId = payload.task_id ?? payload.id // у delete-payload — { task_id, comment_id }
    if (!commentsByTask[taskId]) return // в кэше нет — пропускаем (загрузится при openTask)
    const list = commentsByTask[taskId]
    if (kind === 'new') {
      if (!list.find((c) => c.id === payload.id)) list.push(payload)
    } else if (kind === 'updated') {
      const i = list.findIndex((c) => c.id === payload.id)
      if (i >= 0) list[i] = { ...list[i], ...payload }
    } else if (kind === 'deleted') {
      commentsByTask[taskId] = list.filter((c) => c.id !== payload.comment_id)
    }
  }

  async function loadContributors(taskId) {
    const data = await tasksApi.getTaskContributors(taskId)
    contributorsByTask[taskId] = data.items || []
    return contributorsByTask[taskId]
  }

  function openTask(task) { activeTask.value = task }
  function closeTask() { activeTask.value = null }

  function upsertTask(task) {
    const idx = tasks.value.findIndex(t => t.id === task.id)
    if (idx >= 0) tasks.value[idx] = { ...tasks.value[idx], ...task }
    else tasks.value.unshift(task)
    if (activeTask.value?.id === task.id) {
      activeTask.value = { ...activeTask.value, ...task }
    }
  }

  // Точечное обновление: применяется только если задача уже в списке/открыта.
  // Никогда не вставляет новую запись — это защищает от «пустых карточек»,
  // когда по сокету приходит частичный патч задачи не из текущей выборки.
  function patchTask(patch) {
    const idx = tasks.value.findIndex(t => t.id === patch.id)
    if (idx >= 0) tasks.value[idx] = { ...tasks.value[idx], ...patch }
    if (activeTask.value?.id === patch.id) {
      activeTask.value = { ...activeTask.value, ...patch }
    }
  }

  // Вставка полноценной задачи, пришедшей по сокету (task:created).
  // Добавляем только на вкладке активных задач и только если её ещё нет —
  // новые задачи всегда активные, в избранном/архиве им не место.
  function addTaskFromSocket(task) {
    if (!task || !task.id) return
    const idx = tasks.value.findIndex(t => t.id === task.id)
    if (idx >= 0) {
      tasks.value[idx] = { ...tasks.value[idx], ...task }
      return
    }
    if (filters.tab === 'active' && !task.is_archived) {
      tasks.value.unshift(task)
    }
  }

  function removeTask(taskId) {
    tasks.value = tasks.value.filter(t => t.id !== taskId)
    if (activeTask.value?.id === taskId) activeTask.value = null
  }

  function addActiveUser(taskId, user) {
    const idx = tasks.value.findIndex(t => t.id === taskId)
    if (idx >= 0) {
      const existing = tasks.value[idx].active_users || []
      if (!existing.find(u => u.id === user.id)) {
        tasks.value[idx] = { ...tasks.value[idx], active_users: [...existing, user] }
      }
    }
  }

  function removeActiveUser(taskId, userId) {
    const idx = tasks.value.findIndex(t => t.id === taskId)
    if (idx >= 0) {
      const existing = tasks.value[idx].active_users || []
      tasks.value[idx] = { ...tasks.value[idx], active_users: existing.filter(u => u.id !== userId) }
    }
  }

  function archiveTask(taskId, archived_at) {
    if (filters.tab === 'active' || filters.tab === 'favorites') {
      removeTask(taskId)
    } else {
      patchTask({ id: taskId, is_archived: true, archived_at })
    }
  }

  function restoreTask(taskId) {
    if (filters.tab === 'archive') {
      removeTask(taskId)
    } else {
      patchTask({ id: taskId, is_archived: false, archived_at: null })
    }
  }

  // Учитывает вкладку: снятие отметки на вкладке «Избранное» сразу убирает
  // карточку из списка. activeTask не трогаем (фильтруем только массив),
  // чтобы открытая модалка задачи не закрывалась при переключении отметки.
  function setFavorite(taskId, isFav) {
    patchTask({ id: taskId, is_favorite: isFav })
    if (!isFav && filters.tab === 'favorites') {
      tasks.value = tasks.value.filter(t => t.id !== taskId)
    }
  }

  return {
    tasks, total, loading, error, filters, activeTask,
    commentsByTask, contributorsByTask,
    fetchTasks, setFilter, setTab, resetFilters, openTask, closeTask,
    upsertTask, patchTask, addTaskFromSocket, removeTask, archiveTask, restoreTask,
    setFavorite, addActiveUser, removeActiveUser,
    assignResponsible, setStage, dragMoveStage,
    loadComments, addComment, editComment, deleteComment, applyCommentSocket,
    loadContributors,
  }
})
