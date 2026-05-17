import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import * as tasksApi from '@/api/tasks.js'

export const useTasksStore = defineStore('tasks', () => {
  const tasks = ref([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref(null)
  const activeTask = ref(null)

  const filters = reactive({
    tab: 'active',
    search: '',
    sort: 'last_activity',
    dept_id: null,
    received_from: null,
    received_to: null,
    has_units: null,
    page: 1,
    per_page: 30,
  })

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

  function removeTask(taskId) {
    tasks.value = tasks.value.filter(t => t.id !== taskId)
    if (activeTask.value?.id === taskId) activeTask.value = null
  }

  function archiveTask(taskId, archived_at) {
    if (filters.tab === 'active' || filters.tab === 'favorites') {
      removeTask(taskId)
    } else {
      upsertTask({ id: taskId, is_archived: true, archived_at })
    }
  }

  function restoreTask(taskId) {
    if (filters.tab === 'archive') {
      removeTask(taskId)
    } else {
      upsertTask({ id: taskId, is_archived: false, archived_at: null })
    }
  }

  return {
    tasks, total, loading, error, filters, activeTask,
    fetchTasks, setFilter, setTab, openTask, closeTask,
    upsertTask, removeTask, archiveTask, restoreTask
  }
})
