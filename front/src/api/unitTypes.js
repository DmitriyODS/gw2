// Ведётся вручную: REST типов юнитов живёт в tasksvc (back-go/tasks),
// в Flask-spec его больше нет (MANUAL_TAGS в scripts/gen-api.mjs).
import { apiRequest } from './client.js'

export const getUnitTypes = () => apiRequest('/unit-types')

export const createUnitType = (data) => apiRequest('/unit-types', { method: 'POST', body: data })

export const deleteUnitType = (typeId) => apiRequest(`/unit-types/${typeId}`, { method: 'DELETE' })

export const updateUnitType = (typeId, data) => apiRequest(`/unit-types/${typeId}`, { method: 'PATCH', body: data })
