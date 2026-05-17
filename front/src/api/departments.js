// Сгенерировано из /apispec.json — не редактировать вручную
// Перегенерировать: npm run gen:api
import { apiRequest } from './client.js'

export const getDepartments = () => apiRequest('/departments')

export const createDepartment = (data) => apiRequest('/departments', { method: 'POST', body: data })

export const deleteDepartment = (deptId) => apiRequest(`/departments/${deptId}`, { method: 'DELETE' })

export const updateDepartment = (deptId, data) => apiRequest(`/departments/${deptId}`, { method: 'PATCH', body: data })
