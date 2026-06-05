import { apiRequest } from './client.js'

export const getStages = () => apiRequest('/stages')
export const createStage = (data) => apiRequest('/stages', { method: 'POST', body: data })
export const updateStage = (id, data) => apiRequest(`/stages/${id}`, { method: 'PATCH', body: data })
export const deleteStage = (id) => apiRequest(`/stages/${id}`, { method: 'DELETE' })
export const reorderStages = (ids) => apiRequest('/stages/reorder', { method: 'PATCH', body: { ids } })

export const STAGE_COLORS = ['red', 'orange', 'amber', 'green', 'teal', 'blue', 'violet', 'pink']
