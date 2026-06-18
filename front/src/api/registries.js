// Ведётся вручную: REST реестров живёт в registrysvc (back-go/registry).
import { apiRequest } from './client.js'

// ── Реестры (структура) ──
export const getRegistries = (options = {}) => apiRequest('/registries', options)

export const getRegistry = (id) => apiRequest(`/registries/${id}`)

export const createRegistry = (name) =>
  apiRequest('/registries', { method: 'POST', body: { name } })

export const updateRegistry = (id, name) =>
  apiRequest(`/registries/${id}`, { method: 'PATCH', body: { name } })

export const deleteRegistry = (id) =>
  apiRequest(`/registries/${id}`, { method: 'DELETE' })

// Полная замена набора полей реестра (добавление/удаление/реордер/раскладка).
export const replaceFields = (id, fields) =>
  apiRequest(`/registries/${id}/fields`, { method: 'PUT', body: { fields } })

// ── Записи ──
export const getRecords = (registryId, params = {}, options = {}) => {
  const qs = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') qs.set(k, v) })
  return apiRequest(`/registries/${registryId}/records?${qs}`, options)
}

export const getRecord = (registryId, recordId) =>
  apiRequest(`/registries/${registryId}/records/${recordId}`)

export const createRecord = (registryId, data) =>
  apiRequest(`/registries/${registryId}/records`, { method: 'POST', body: { data } })

export const updateRecord = (registryId, recordId, data) =>
  apiRequest(`/registries/${registryId}/records/${recordId}`, { method: 'PATCH', body: { data } })

export const deleteRecord = (registryId, recordId) =>
  apiRequest(`/registries/${registryId}/records/${recordId}`, { method: 'DELETE' })

export const bulkDeleteRecords = (registryId, ids) =>
  apiRequest(`/registries/${registryId}/records/bulk-delete`, { method: 'POST', body: { ids } })

// Экспорт записей в xlsx. params: { fields: [ids], search?, ids?: [recordIds] }.
// Возвращает Response (blob:true) для скачивания файла.
export const exportRecords = (registryId, { fields = [], search = '', ids = [] } = {}) => {
  const qs = new URLSearchParams()
  if (fields.length) qs.set('fields', fields.join(','))
  if (search) qs.set('search', search)
  if (ids.length) qs.set('ids', ids.join(','))
  return apiRequest(`/registries/${registryId}/export?${qs}`, { blob: true })
}

// Загрузка файла/картинки записи (multipart) → { path, name, mime, size }.
export const uploadFile = (file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest('/registries/uploads', { method: 'POST', body: form })
}

// ── Публичные ссылки (управление, требует прав участника) ──
export const getShares = (registryId) => apiRequest(`/registries/${registryId}/shares`)
export const createShare = (registryId) =>
  apiRequest(`/registries/${registryId}/shares`, { method: 'POST' })
export const revokeShare = (registryId, shareId) =>
  apiRequest(`/registries/${registryId}/shares/${shareId}`, { method: 'DELETE' })

// ── Публичный доступ по коду (без авторизации) ──
export const getSharedRegistry = (code) => apiRequest(`/registries/shared/${code}`)
export const getSharedRecords = (code, params = {}, options = {}) => {
  const qs = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') qs.set(k, v) })
  return apiRequest(`/registries/shared/${code}/records?${qs}`, options)
}
export const exportSharedRecords = (code, { fields = [], search = '', ids = [] } = {}) => {
  const qs = new URLSearchParams()
  if (fields.length) qs.set('fields', fields.join(','))
  if (search) qs.set('search', search)
  if (ids.length) qs.set('ids', ids.join(','))
  return apiRequest(`/registries/shared/${code}/export?${qs}`, { blob: true })
}
