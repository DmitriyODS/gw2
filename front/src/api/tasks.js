// Ведётся вручную: REST задач живёт в tasksvc (back-go/tasks), в Flask-spec
// его больше нет (MANUAL_TAGS в scripts/gen-api.mjs).
import { apiRequest } from './client.js'

export const getTasks = (params = {}, options = {}) => {
  const qs = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') qs.set(k, v) })
  return apiRequest('/tasks' + '?' + qs, options)
}

export const createTask = (data) => apiRequest('/tasks', { method: 'POST', body: data })

export const deleteTask = (taskId) => apiRequest(`/tasks/${taskId}`, { method: 'DELETE' })

export const getTask = (taskId) => apiRequest(`/tasks/${taskId}`)

export const updateTask = (taskId, data) => apiRequest(`/tasks/${taskId}`, { method: 'PATCH', body: data })

export const archiveTask = (taskId) => apiRequest(`/tasks/${taskId}/archive`, { method: 'POST' })

export const toggleFavorite = (taskId) => apiRequest(`/tasks/${taskId}/favorite`, { method: 'POST' })

export const restoreTask = (taskId) => apiRequest(`/tasks/${taskId}/restore`, { method: 'POST' })

// Индивидуальный цвет карточки для текущего пользователя (передать color=null чтобы снять)
export const setTaskColor = (taskId, color) =>
  apiRequest(`/tasks/${taskId}/color`, { method: 'PUT', body: { color } })

// v3 — ответственный, этап, контрибьюторы, комментарии
export const setTaskResponsible = (taskId, responsibleUserId) =>
  apiRequest(`/tasks/${taskId}/responsible`, {
    method: 'PATCH',
    body: { responsible_user_id: responsibleUserId },
  })

export const setTaskStage = (taskId, stageId) =>
  apiRequest(`/tasks/${taskId}/stage`, { method: 'PATCH', body: { stage_id: stageId } })

export const getTaskContributors = (taskId) =>
  apiRequest(`/tasks/${taskId}/contributors`)

export const listTaskComments = (taskId) =>
  apiRequest(`/tasks/${taskId}/comments`)

export const createTaskComment = (taskId, text) =>
  apiRequest(`/tasks/${taskId}/comments`, { method: 'POST', body: { text } })

export const updateTaskComment = (taskId, commentId, text) =>
  apiRequest(`/tasks/${taskId}/comments/${commentId}`, { method: 'PATCH', body: { text } })

export const deleteTaskComment = (taskId, commentId) =>
  apiRequest(`/tasks/${taskId}/comments/${commentId}`, { method: 'DELETE' })
