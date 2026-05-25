// Сгенерировано из /apispec.json — не редактировать вручную
// Перегенерировать: npm run gen:api
import { apiRequest } from './client.js'

export const getTasks = (params = {}) => {
  const qs = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') qs.set(k, v) })
  return apiRequest('/tasks' + '?' + qs)
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
