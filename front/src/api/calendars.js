// Ведётся вручную: REST календарей живёт в calendarsvc (back-go/calendar).
import { apiRequest } from './client.js'

// ── Календари (структура) ──
export const getCalendars = (options = {}) => apiRequest('/calendars', options)

export const getCalendar = (id) => apiRequest(`/calendars/${id}`)

export const createCalendar = (name) =>
  apiRequest('/calendars', { method: 'POST', body: { name } })

export const updateCalendar = (id, name) =>
  apiRequest(`/calendars/${id}`, { method: 'PATCH', body: { name } })

export const deleteCalendar = (id) =>
  apiRequest(`/calendars/${id}`, { method: 'DELETE' })

// Полная замена набора полей календаря (добавление/удаление/реордер/раскладка/
// условная видимость).
export const replaceFields = (id, fields) =>
  apiRequest(`/calendars/${id}/fields`, { method: 'PUT', body: { fields } })

// ── Записи ──
// params: { from?: ISO, to?: ISO, search? }. Возвращает { items: [...] }.
export const getEntries = (calendarId, params = {}, options = {}) => {
  const qs = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') qs.set(k, v) })
  return apiRequest(`/calendars/${calendarId}/records?${qs}`, options)
}

export const getEntry = (calendarId, entryId) =>
  apiRequest(`/calendars/${calendarId}/records/${entryId}`)

export const createEntry = (calendarId, eventAt, data) =>
  apiRequest(`/calendars/${calendarId}/records`, { method: 'POST', body: { event_at: eventAt, data } })

export const updateEntry = (calendarId, entryId, eventAt, data) =>
  apiRequest(`/calendars/${calendarId}/records/${entryId}`, { method: 'PATCH', body: { event_at: eventAt, data } })

export const deleteEntry = (calendarId, entryId) =>
  apiRequest(`/calendars/${calendarId}/records/${entryId}`, { method: 'DELETE' })

export const bulkDeleteEntries = (calendarId, ids) =>
  apiRequest(`/calendars/${calendarId}/records/bulk-delete`, { method: 'POST', body: { ids } })

// Экспорт записей в xlsx. params: { fields: [ids], from?, to?, search?, ids?: [entryIds] }.
// Возвращает Response (blob:true) для скачивания файла.
export const exportEntries = (calendarId, { fields = [], from = '', to = '', search = '', ids = [] } = {}) => {
  const qs = new URLSearchParams()
  if (fields.length) qs.set('fields', fields.join(','))
  if (from) qs.set('from', from)
  if (to) qs.set('to', to)
  if (search) qs.set('search', search)
  if (ids.length) qs.set('ids', ids.join(','))
  return apiRequest(`/calendars/${calendarId}/export?${qs}`, { blob: true })
}

// Загрузка файла/картинки записи (multipart) → { path, name, mime, size }.
export const uploadFile = (file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest('/calendars/uploads', { method: 'POST', body: form })
}

// ── Публичные ссылки (управление, требует прав участника) ──
export const getShares = (calendarId) => apiRequest(`/calendars/${calendarId}/shares`)
export const createShare = (calendarId) =>
  apiRequest(`/calendars/${calendarId}/shares`, { method: 'POST' })
export const revokeShare = (calendarId, shareId) =>
  apiRequest(`/calendars/${calendarId}/shares/${shareId}`, { method: 'DELETE' })

// ── Публичный доступ по коду (без авторизации) ──
export const getSharedCalendar = (code) => apiRequest(`/calendars/shared/${code}`)
export const getSharedEntries = (code, params = {}, options = {}) => {
  const qs = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') qs.set(k, v) })
  return apiRequest(`/calendars/shared/${code}/records?${qs}`, options)
}
export const exportSharedEntries = (code, { fields = [], from = '', to = '', search = '', ids = [] } = {}) => {
  const qs = new URLSearchParams()
  if (fields.length) qs.set('fields', fields.join(','))
  if (from) qs.set('from', from)
  if (to) qs.set('to', to)
  if (search) qs.set('search', search)
  if (ids.length) qs.set('ids', ids.join(','))
  return apiRequest(`/calendars/shared/${code}/export?${qs}`, { blob: true })
}
