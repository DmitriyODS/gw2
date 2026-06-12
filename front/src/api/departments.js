// Ведётся вручную: REST отделов живёт в tasksvc (back-go/tasks), в Flask-spec
// его больше нет (MANUAL_TAGS в scripts/gen-api.mjs).
import { apiRequest } from './client.js'

export const getDepartments = () => apiRequest('/departments')

export const createDepartment = (data) => apiRequest('/departments', { method: 'POST', body: data })

export const deleteDepartment = (deptId) => apiRequest(`/departments/${deptId}`, { method: 'DELETE' })

export const updateDepartment = (deptId, data) => apiRequest(`/departments/${deptId}`, { method: 'PATCH', body: data })
