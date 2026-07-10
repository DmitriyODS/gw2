import { defineStore } from 'pinia'
import { ref, reactive, watch, computed } from 'vue'
import * as tasksApi from '@/api/tasks.js'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'
import { storageGet, storageSet } from '@/utils/storage.js'

const STORAGE_KEY = 'gw2_tasks_filters'

function loadSavedFilters() {
  const raw = storageGet(STORAGE_KEY, '')
  if (!raw) return {}
  try {
    return JSON.parse(raw)
  } catch {
    return {}
  }
}

export const useTasksStore = defineStore('tasks', () => {
  const tasks = ref([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref(null)
  const activeTask = ref(null)
  const taskById = computed(() => {
    const map = new Map()
    for (const task of tasks.value) map.set(task.id, task)
    return map
  })

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
    created_by_me: saved.created_by_me ?? false,
    tag_ids: saved.tag_ids ?? [],
    colors: saved.colors ?? [],
    page: 1,
    per_page: 30,
  })

  // Карта комментариев: task_id → массив комментариев (упорядочены по created_at).
  const commentsByTask = reactive({})
  // Карта контрибьюторов: task_id → массив { id, fio, avatar_path }.
  const contributorsByTask = reactive({})
  let fetchSeq = 0
  let fetchCtrl = null

  watch(filters, () => {
    // eslint-disable-next-line no-unused-vars
    const { page, per_page, ...toSave } = { ...filters }
    storageSet(STORAGE_KEY, JSON.stringify(toSave))
  }, { deep: true })

  function _hasCompanyScope() {
    const auth = useAuthStore()
    if (auth.companyId != null) return true
    const companies = useCompaniesStore()
    return companies.activeCompanyId != null
  }

  // Бейдж навигации: сколько активных задач, где я ответственный.
  // Отдельный лёгкий запрос (per_page=1, читаем только total) — основная
  // выборка живёт своими фильтрами и не годится как источник счётчика.
  const myActiveCount = ref(0)
  let badgeTimer = null

  async function fetchMyActiveCount() {
    const auth = useAuthStore()
    if (!_hasCompanyScope() || auth.userId == null) {
      myActiveCount.value = 0
      return
    }
    try {
      const data = await tasksApi.getTasks({
        tab: 'active', responsible_id: auth.userId, page: 1, per_page: 1,
      })
      myActiveCount.value = data.total ?? 0
    } catch { /* бейдж некритичен — молча пропускаем */ }
  }

  // Сокет-события задач приходят сериями — пересчёт с дебаунсом.
  function refreshMyActiveCount() {
    clearTimeout(badgeTimer)
    badgeTimer = setTimeout(() => { fetchMyActiveCount() }, 1500)
  }

  async function fetchTasks({ silent = false } = {}) {
    if (!_hasCompanyScope()) {
      tasks.value = []
      total.value = 0
      return
    }
    const seq = ++fetchSeq
    fetchCtrl?.abort()
    fetchCtrl = new AbortController()
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
      if (filters.created_by_me) params.created_by_me = '1'
      if (filters.tag_ids?.length) params.tag_ids = filters.tag_ids.join(',')
      if (filters.colors?.length) params.colors = filters.colors.join(',')
      params.page = filters.page
      params.per_page = filters.per_page

      const data = await tasksApi.getTasks(params, { signal: fetchCtrl.signal })
      if (seq !== fetchSeq) return
      tasks.value = data.tasks ?? data.items ?? data
      total.value = data.total ?? tasks.value.length
    } catch (e) {
      if (e?.error === 'ABORTED') return
      error.value = e.message || 'Ошибка загрузки задач'
      throw e
    } finally {
      if (seq === fetchSeq) {
        fetchCtrl = null
        if (!silent) loading.value = false
      }
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
    filters.created_by_me = false
    filters.tag_ids = []
    filters.colors = []
    filters.page = 1
    fetchTasks().catch(() => {})
  }

  // Переключить цвет в мультифильтре (личный цвет карточек).
  function toggleColorFilter(color) {
    const cur = filters.colors || []
    filters.colors = cur.includes(color)
      ? cur.filter((c) => c !== color)
      : [...cur, color]
    filters.page = 1
    fetchTasks().catch(() => {})
  }

  // ── Теги (справочник компании — общий для фильтров/формы/меню) ──
  const tags = ref([])
  let tagsLoaded = false

  async function fetchTags({ force = false } = {}) {
    if (tagsLoaded && !force) return
    try {
      const data = await tasksApi.getTags()
      tags.value = Array.isArray(data) ? data : (data.items ?? [])
      tagsLoaded = true
    } catch { /* справочник некритичен — фильтр просто пуст */ }
  }

  // Переключить один тег У ЗАДАЧИ (ПКМ-меню, модалка) — полная замена набора.
  async function toggleTaskTag(taskId, tagId) {
    const task = taskById.value.get(taskId)
    const cur = (task?.tags || []).map((t) => t.id)
    const next = cur.includes(tagId) ? cur.filter((id) => id !== tagId) : [...cur, tagId]
    return setTaskTags(taskId, next)
  }

  // Переключить тег в мультифильтре (чипы в панели фильтров).
  function toggleTagFilter(tagId) {
    const cur = filters.tag_ids || []
    filters.tag_ids = cur.includes(tagId)
      ? cur.filter((id) => id !== tagId)
      : [...cur, tagId]
    filters.page = 1
    fetchTasks().catch(() => {})
  }

  // Назначить набор тегов задаче (полная замена) — ответ и сокет-событие
  // несут свежие tags, patchTask применит.
  async function setTaskTags(taskId, tagIds) {
    const updated = await tasksApi.setTaskTags(taskId, tagIds)
    patchTask(updated)
    return updated
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
    const prevStageId = taskById.value.get(taskId)?.stage_id ?? null
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
    tasks, taskById, total, loading, error, filters, activeTask,
    commentsByTask, contributorsByTask,
    myActiveCount, fetchMyActiveCount, refreshMyActiveCount,
    tags, fetchTags,
    fetchTasks, setFilter, setTab, resetFilters, toggleTagFilter, toggleColorFilter, openTask, closeTask,
    upsertTask, patchTask, addTaskFromSocket, removeTask, archiveTask, restoreTask,
    setFavorite, addActiveUser, removeActiveUser,
    assignResponsible, setStage, dragMoveStage, setTaskTags, toggleTaskTag,
    loadComments, addComment, editComment, deleteComment, applyCommentSocket,
    loadContributors,
  }
})
