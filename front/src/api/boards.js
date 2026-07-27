// Ведётся вручную: REST досок живёт в boardsvc (back-go/board).
import { apiRequest } from './client.js'

function qs(params = {}) {
  const sp = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') sp.set(k, v) })
  return sp.toString()
}

// ── Доски ──
// params: { folder_id? ('root'|id), search?, archived? ('1'), shared? ('1') }.
export const getBoards = (params = {}, options = {}) =>
  apiRequest(`/boards?${qs(params)}`, options)

export const getBoard = (id) => apiRequest(`/boards/${id}`)

export const createBoard = (title = '', folderId = null) =>
  apiRequest('/boards', { method: 'POST', body: { title, folder_id: folderId } })

// Частичная правка: { title?, scene?, color?, archived?, pinned? }.
export const updateBoard = (id, body) =>
  apiRequest(`/boards/${id}`, { method: 'PATCH', body })

export const deleteBoard = (id) =>
  apiRequest(`/boards/${id}`, { method: 'DELETE' })

// folderId: null — в корень.
export const moveBoard = (id, folderId) =>
  apiRequest(`/boards/${id}/move`, { method: 'POST', body: { folder_id: folderId } })

export const copyBoard = (id) =>
  apiRequest(`/boards/${id}/copy`, { method: 'POST' })

// ── Папки ──
// → { folders: [свои плоско], shared: [расшаренные мне корни] }.
export const getFolders = () => apiRequest('/boards/folders')
// → { folders: [подпапки], my_access }.
export const getFolderChildren = (id) => apiRequest(`/boards/folders/${id}/children`)
export const createFolder = (name, parentId = null, color = '') =>
  apiRequest('/boards/folders', { method: 'POST', body: { name, parent_id: parentId, color } })
export const updateFolder = (id, body) =>
  apiRequest(`/boards/folders/${id}`, { method: 'PATCH', body })
export const moveFolder = (id, parentId) =>
  apiRequest(`/boards/folders/${id}/move`, { method: 'POST', body: { parent_id: parentId } })
export const copyFolder = (id) =>
  apiRequest(`/boards/folders/${id}/copy`, { method: 'POST' })
export const deleteFolder = (id) =>
  apiRequest(`/boards/folders/${id}`, { method: 'DELETE' })

// Компании пользователя (любое членство) — для выбора аудитории шаринга.
export const getMyCompanies = () => apiRequest('/boards/companies')

// ── Публичные ссылки (владелец) ──
export const getShares = (boardId) => apiRequest(`/boards/${boardId}/shares`)
export const createShare = (boardId, access) =>
  apiRequest(`/boards/${boardId}/shares`, { method: 'POST', body: { access } })
export const revokeShare = (boardId, shareId) =>
  apiRequest(`/boards/${boardId}/shares/${shareId}`, { method: 'DELETE' })

// ── Адресный шаринг досок (пользователь/компания) ──
export const getBoardMembers = (boardId) => apiRequest(`/boards/${boardId}/members`)
export const shareBoardWithUser = (boardId, userId, canEdit) =>
  apiRequest(`/boards/${boardId}/members`, { method: 'POST', body: { target: 'user', user_id: userId, can_edit: canEdit } })
export const shareBoardWithCompany = (boardId, companyId, canEdit) =>
  apiRequest(`/boards/${boardId}/members`, { method: 'POST', body: { target: 'company', company_id: companyId, can_edit: canEdit } })
export const unshareBoardUser = (boardId, userId) =>
  apiRequest(`/boards/${boardId}/members/user/${userId}`, { method: 'DELETE' })
export const unshareBoardCompany = (boardId, companyId) =>
  apiRequest(`/boards/${boardId}/members/company/${companyId}`, { method: 'DELETE' })

// ── Адресный шаринг папок (пользователь/компания) ──
export const getFolderMembers = (folderId) => apiRequest(`/boards/folders/${folderId}/members`)
export const shareFolderWithUser = (folderId, userId, canEdit) =>
  apiRequest(`/boards/folders/${folderId}/members`, { method: 'POST', body: { target: 'user', user_id: userId, can_edit: canEdit } })
export const shareFolderWithCompany = (folderId, companyId, canEdit) =>
  apiRequest(`/boards/folders/${folderId}/members`, { method: 'POST', body: { target: 'company', company_id: companyId, can_edit: canEdit } })
export const unshareFolderUser = (folderId, userId) =>
  apiRequest(`/boards/folders/${folderId}/members/user/${userId}`, { method: 'DELETE' })
export const unshareFolderCompany = (folderId, companyId) =>
  apiRequest(`/boards/folders/${folderId}/members/company/${companyId}`, { method: 'DELETE' })

// ── Совместное рисование ──
// body: { kind: 'join'|'leave'|'cursor'|'ops'|'scene', cursor?, ops?, scene?, title? }.
export const sendCollab = (boardId, body) =>
  apiRequest(`/boards/${boardId}/collab`, { method: 'POST', body })

// ── Картинки холста и превью плитки ──
export const uploadImage = (boardId, file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest(`/boards/${boardId}/uploads`, { method: 'POST', body: form })
}

// Миниатюра холста (png-снимок делает сам редактор).
export const uploadPreview = (boardId, blob) => {
  const form = new FormData()
  form.append('file', blob, 'preview.png')
  return apiRequest(`/boards/${boardId}/preview`, { method: 'PUT', body: form })
}

// ── Экспорт/импорт ──
// format: 'svg' | 'json' (доска), папка — всегда zip.
export const exportBoard = (id, format = 'svg') =>
  apiRequest(`/boards/${id}/export?${qs({ format })}`, { blob: true })
export const exportFolder = (id, format = 'svg') =>
  apiRequest(`/boards/folders/${id}/export?${qs({ format })}`, { blob: true })
// scope: 'all' | 'archive' | 'shared' — zip всей группировки.
export const exportScope = (scope, format = 'svg') =>
  apiRequest(`/boards/export?${qs({ scope, format })}`, { blob: true })
export const importBoard = (file, folderId = null) => {
  const form = new FormData()
  form.append('file', file)
  if (folderId != null) form.append('folder_id', String(folderId))
  return apiRequest('/boards/import', { method: 'POST', body: form })
}

// ── Публичный доступ по коду (без авторизации) ──
export const getSharedBoard = (code) => apiRequest(`/boards/shared/${code}`)
export const updateSharedBoard = (code, body) =>
  apiRequest(`/boards/shared/${code}`, { method: 'PUT', body })
