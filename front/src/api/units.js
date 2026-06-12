// Ведётся вручную: REST юнитов живёт в tasksvc (back-go/tasks), в Flask-spec
// его больше нет (MANUAL_TAGS в scripts/gen-api.mjs).
import { apiRequest } from './client.js'

export const getUnits = (taskId) => apiRequest(`/tasks/${taskId}/units`)

export const createUnit = (taskId, data) => apiRequest(`/tasks/${taskId}/units`, { method: 'POST', body: data })

export const getActiveUnit = () => apiRequest('/units/active')

export const deleteUnit = (unitId) => apiRequest(`/units/${unitId}`, { method: 'DELETE' })

export const updateUnit = (unitId, data) => apiRequest(`/units/${unitId}`, { method: 'PATCH', body: data })

export const stopUnit = (unitId) => apiRequest(`/units/${unitId}/stop`, { method: 'POST' })
