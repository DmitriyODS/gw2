// Сгенерировано из /apispec.json — не редактировать вручную
// Перегенерировать: npm run gen:api
import { apiRequest } from './client.js'

export const getUnits = (taskId) => apiRequest(`/tasks/${taskId}/units`)

export const createUnit = (taskId, data) => apiRequest(`/tasks/${taskId}/units`, { method: 'POST', body: data })

export const getActiveUnit = () => apiRequest('/units/active')

export const deleteUnit = (unitId) => apiRequest(`/units/${unitId}`, { method: 'DELETE' })

export const updateUnit = (unitId, data) => apiRequest(`/units/${unitId}`, { method: 'PATCH', body: data })

export const stopUnit = (unitId) => apiRequest(`/units/${unitId}/stop`, { method: 'POST' })
