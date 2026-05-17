// Сгенерировано из /apispec.json — не редактировать вручную
// Перегенерировать: npm run gen:api
import { apiRequest } from './client.js'

export const getRoles = () => apiRequest('/roles')

export const createRole = (data) => apiRequest('/roles', { method: 'POST', body: data })

export const deleteRole = (roleId) => apiRequest(`/roles/${roleId}`, { method: 'DELETE' })

export const updateRole = (roleId, data) => apiRequest(`/roles/${roleId}`, { method: 'PATCH', body: data })
