// Ведётся вручную: REST заметок живёт в notesvc (back-go/notes).
import { apiRequest } from './client.js'

function qs(params = {}) {
  const sp = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') sp.set(k, v) })
  return sp.toString()
}

// ── Заметки ──
// params: { group_id?, search?, archived? ('1' — архив) }. → { notes: [плитки без doc] }.
export const getNotes = (params = {}, options = {}) =>
  apiRequest(`/notes?${qs(params)}`, options)

// Полная заметка (с doc).
export const getNote = (id) => apiRequest(`/notes/${id}`)

export const createNote = (title = '') =>
  apiRequest('/notes', { method: 'POST', body: { title } })

// Частичная правка: { title?, doc?, color?, archived? } — отсутствующие поля не меняются.
export const updateNote = (id, body) =>
  apiRequest(`/notes/${id}`, { method: 'PATCH', body })

export const deleteNote = (id) =>
  apiRequest(`/notes/${id}`, { method: 'DELETE' })

// Полная замена групп заметки.
export const setNoteGroups = (id, groupIds) =>
  apiRequest(`/notes/${id}/groups`, { method: 'PUT', body: { group_ids: groupIds } })

// ── Группы ──
export const getGroups = () => apiRequest('/notes/groups')
export const createGroup = (name) =>
  apiRequest('/notes/groups', { method: 'POST', body: { name } })
export const renameGroup = (id, name) =>
  apiRequest(`/notes/groups/${id}`, { method: 'PATCH', body: { name } })
export const deleteGroup = (id) =>
  apiRequest(`/notes/groups/${id}`, { method: 'DELETE' })

// ── Публичные ссылки (владелец) ──
export const getShares = (noteId) => apiRequest(`/notes/${noteId}/shares`)
// access: 'view' (только чтение) | 'edit' (чтение и редактирование).
export const createShare = (noteId, access) =>
  apiRequest(`/notes/${noteId}/shares`, { method: 'POST', body: { access } })
export const revokeShare = (noteId, shareId) =>
  apiRequest(`/notes/${noteId}/shares/${shareId}`, { method: 'DELETE' })

// ── Картинки редактора ──
// → { path: '/uploads/notes/…' } — готовый src для вставки в документ.
export const uploadImage = (noteId, file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest(`/notes/${noteId}/uploads`, { method: 'POST', body: form })
}

// ── Экспорт/импорт txt ──
export const exportNote = (id) => apiRequest(`/notes/${id}/export`, { blob: true })
export const importNote = (file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest('/notes/import', { method: 'POST', body: form })
}

// ── Публичный доступ по коду (без авторизации) ──
// → { note, access }.
export const getSharedNote = (code) => apiRequest(`/notes/shared/${code}`)
export const updateSharedNote = (code, body) =>
  apiRequest(`/notes/shared/${code}`, { method: 'PUT', body })
