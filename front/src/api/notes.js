// Ведётся вручную: REST заметок живёт в notesvc (back-go/notes).
import { apiRequest } from './client.js'

function qs(params = {}) {
  const sp = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') sp.set(k, v) })
  return sp.toString()
}

// ── Заметки ──
// params: { folder_id? ('root'|id), tag_ids? ('1,2'), search?, archived? ('1'), shared? ('1') }.
export const getNotes = (params = {}, options = {}) =>
  apiRequest(`/notes?${qs(params)}`, options)

export const getNote = (id) => apiRequest(`/notes/${id}`)

export const createNote = (title = '', folderId = null) =>
  apiRequest('/notes', { method: 'POST', body: { title, folder_id: folderId } })

// Частичная правка: { title?, doc?, color?, archived?, pinned? }.
export const updateNote = (id, body) =>
  apiRequest(`/notes/${id}`, { method: 'PATCH', body })

export const deleteNote = (id) =>
  apiRequest(`/notes/${id}`, { method: 'DELETE' })

// folderId: null — в корень.
export const moveNote = (id, folderId) =>
  apiRequest(`/notes/${id}/move`, { method: 'POST', body: { folder_id: folderId } })

export const copyNote = (id) =>
  apiRequest(`/notes/${id}/copy`, { method: 'POST' })

// Полная замена тегов заметки.
export const setNoteTags = (id, tagIds) =>
  apiRequest(`/notes/${id}/tags`, { method: 'PUT', body: { tag_ids: tagIds } })

// ── Папки ──
// → { folders: [свои плоско], shared: [расшаренные мне корни] }.
export const getFolders = () => apiRequest('/notes/folders')
// → { folders: [подпапки], my_access }.
export const getFolderChildren = (id) => apiRequest(`/notes/folders/${id}/children`)
export const createFolder = (name, parentId = null, color = '') =>
  apiRequest('/notes/folders', { method: 'POST', body: { name, parent_id: parentId, color } })
export const updateFolder = (id, body) =>
  apiRequest(`/notes/folders/${id}`, { method: 'PATCH', body })
export const moveFolder = (id, parentId) =>
  apiRequest(`/notes/folders/${id}/move`, { method: 'POST', body: { parent_id: parentId } })
export const copyFolder = (id) =>
  apiRequest(`/notes/folders/${id}/copy`, { method: 'POST' })
export const deleteFolder = (id) =>
  apiRequest(`/notes/folders/${id}`, { method: 'DELETE' })

// Компании пользователя (любое членство) — для выбора аудитории шаринга.
export const getMyCompanies = () => apiRequest('/notes/companies')

// ── Теги ──
export const getTags = () => apiRequest('/notes/tags')
export const createTag = (name, color = '') =>
  apiRequest('/notes/tags', { method: 'POST', body: { name, color } })
export const updateTag = (id, body) =>
  apiRequest(`/notes/tags/${id}`, { method: 'PATCH', body })
export const deleteTag = (id) =>
  apiRequest(`/notes/tags/${id}`, { method: 'DELETE' })

// ── Публичные ссылки (владелец) ──
export const getShares = (noteId) => apiRequest(`/notes/${noteId}/shares`)
export const createShare = (noteId, access) =>
  apiRequest(`/notes/${noteId}/shares`, { method: 'POST', body: { access } })
export const revokeShare = (noteId, shareId) =>
  apiRequest(`/notes/${noteId}/shares/${shareId}`, { method: 'DELETE' })

// ── Адресный шаринг заметок (пользователь/компания) ──
export const getNoteMembers = (noteId) => apiRequest(`/notes/${noteId}/members`)
export const shareNoteWithUser = (noteId, userId, canEdit) =>
  apiRequest(`/notes/${noteId}/members`, { method: 'POST', body: { target: 'user', user_id: userId, can_edit: canEdit } })
export const shareNoteWithCompany = (noteId, companyId, canEdit) =>
  apiRequest(`/notes/${noteId}/members`, { method: 'POST', body: { target: 'company', company_id: companyId, can_edit: canEdit } })
export const unshareNoteUser = (noteId, userId) =>
  apiRequest(`/notes/${noteId}/members/user/${userId}`, { method: 'DELETE' })
export const unshareNoteCompany = (noteId, companyId) =>
  apiRequest(`/notes/${noteId}/members/company/${companyId}`, { method: 'DELETE' })

// ── Адресный шаринг папок (пользователь/компания) ──
export const getFolderMembers = (folderId) => apiRequest(`/notes/folders/${folderId}/members`)
export const shareFolderWithUser = (folderId, userId, canEdit) =>
  apiRequest(`/notes/folders/${folderId}/members`, { method: 'POST', body: { target: 'user', user_id: userId, can_edit: canEdit } })
export const shareFolderWithCompany = (folderId, companyId, canEdit) =>
  apiRequest(`/notes/folders/${folderId}/members`, { method: 'POST', body: { target: 'company', company_id: companyId, can_edit: canEdit } })
export const unshareFolderUser = (folderId, userId) =>
  apiRequest(`/notes/folders/${folderId}/members/user/${userId}`, { method: 'DELETE' })
export const unshareFolderCompany = (folderId, companyId) =>
  apiRequest(`/notes/folders/${folderId}/members/company/${companyId}`, { method: 'DELETE' })

// ── Совместное редактирование ──
export const sendCollab = (noteId, body) =>
  apiRequest(`/notes/${noteId}/collab`, { method: 'POST', body })

// ── Картинки редактора ──
export const uploadImage = (noteId, file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest(`/notes/${noteId}/uploads`, { method: 'POST', body: form })
}

// ── Экспорт/импорт ──
// format: 'txt' | 'docx' (заметка), папка — всегда zip.
export const exportNote = (id, format = 'txt') =>
  apiRequest(`/notes/${id}/export?${qs({ format })}`, { blob: true })
export const exportFolder = (id, format = 'txt') =>
  apiRequest(`/notes/folders/${id}/export?${qs({ format })}`, { blob: true })
// scope: 'all' | 'archive' | 'shared' — zip всей группировки.
export const exportScope = (scope, format = 'txt') =>
  apiRequest(`/notes/export?${qs({ scope, format })}`, { blob: true })
export const importNote = (file, folderId = null) => {
  const form = new FormData()
  form.append('file', file)
  if (folderId != null) form.append('folder_id', String(folderId))
  return apiRequest('/notes/import', { method: 'POST', body: form })
}

// ── Публичный доступ по коду (без авторизации) ──
export const getSharedNote = (code) => apiRequest(`/notes/shared/${code}`)
export const updateSharedNote = (code, body) =>
  apiRequest(`/notes/shared/${code}`, { method: 'PUT', body })
