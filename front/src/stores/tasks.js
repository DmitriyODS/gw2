import { defineStore } from 'pinia'
import { ref, reactive, watch } from 'vue'
import * as tasksApi from '@/api/tasks.js'

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
    received_from: saved.received_from ?? null,
    received_to: saved.received_to ?? null,
    has_units: saved.has_units ?? null,
    period_preset: saved.period_preset ?? null,
    page: 1,
    per_page: 30,
  })

  watch(filters, () => {
    // eslint-disable-next-line no-unused-vars
    const { page, per_page, ...toSave } = { ...filters }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(toSave))
  }, { deep: true })

  async function fetchTasks() {
    loading.value = true
    error.value = null
    try {
      const params = {}
      if (filters.tab) params.tab = filters.tab
      if (filters.search) params.search = filters.search
      if (filters.sort) params.sort = filters.sort
      if (filters.dept_id) params.dept_id = filters.dept_id
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
      loading.value = false
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

  return {
    tasks, total, loading, error, filters, activeTask,
    fetchTasks, setFilter, setTab, openTask, closeTask,
    upsertTask, patchTask, addTaskFromSocket, removeTask, archiveTask, restoreTask,
    addActiveUser, removeActiveUser
  }
})
