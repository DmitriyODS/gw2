// Ведётся вручную: REST «Диска» живёт в drivesvc (back-go/drive).
import { apiRequest } from './client.js'
import { uploadFileTo } from '@/utils/chunkUpload.js'

function qs(params = {}) {
  const sp = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') sp.set(k, v) })
  return sp.toString()
}

// ── Обзор ────────────────────────────────────────────────────────────────────

// params: { folder_id?, search?, view? ('trash'|'starred'|'recent') }.
// Ответ: { folder, path, folders, files }.
export const browse = (params = {}, options = {}) =>
  apiRequest(`/drive?${qs(params)}`, options)

export const sharedWithMe = () => apiRequest('/drive/shared-with-me')

export const searchUsers = (q) => apiRequest(`/drive/users?${qs({ q })}`)

// ── Папки ────────────────────────────────────────────────────────────────────

export const createFolder = (name, parentId = null) =>
  apiRequest('/drive/folders', { method: 'POST', body: { name, parent_id: parentId } })

export const updateFolder = (id, body) =>
  apiRequest(`/drive/folders/${id}`, { method: 'PATCH', body })

export const moveFolder = (id, parentId) =>
  apiRequest(`/drive/folders/${id}/move`, { method: 'POST', body: { parent_id: parentId } })

export const trashFolder = (id) => apiRequest(`/drive/folders/${id}/trash`, { method: 'POST' })
export const restoreFolder = (id) => apiRequest(`/drive/folders/${id}/restore`, { method: 'POST' })
export const purgeFolder = (id) => apiRequest(`/drive/folders/${id}`, { method: 'DELETE' })

// ── Файлы ────────────────────────────────────────────────────────────────────

/* Загрузка.

   Мелкое уходит одним multipart-запросом, крупное — ЧАСТЯМИ: сотни мегабайт
   одним телом упираются в таймауты прокси и не переживают обрыв сети. Порог и
   вся механика общие для платформы (utils/chunkUpload.js) — раздел лишь
   называет свои ручки и контекст: папку назначения. */
export { CHUNK_THRESHOLD, CHUNK_SIZE } from '@/utils/chunkUpload.js'

export function uploadFile(file, folderId = null, { onProgress, signal } = {}) {
  return uploadFileTo({
    file,
    onProgress,
    signal,
    directUrl: `/drive/files?${qs({ folder_id: folderId })}`,
    chunkBase: '/drive/uploads',
    scope: String(folderId ?? ''),
  })
}

export const getFile = (id) => apiRequest(`/drive/files/${id}`)

export const renameFile = (id, name) =>
  apiRequest(`/drive/files/${id}`, { method: 'PATCH', body: { name } })

export const moveFile = (id, folderId) =>
  apiRequest(`/drive/files/${id}/move`, { method: 'POST', body: { folder_id: folderId } })

export const starFile = (id, starred) =>
  apiRequest(`/drive/files/${id}/star`, { method: 'POST', body: { starred } })

export const trashFile = (id) => apiRequest(`/drive/files/${id}/trash`, { method: 'POST' })
export const restoreFile = (id) => apiRequest(`/drive/files/${id}/restore`, { method: 'POST' })
export const purgeFile = (id) => apiRequest(`/drive/files/${id}`, { method: 'DELETE' })

export const emptyTrash = () => apiRequest('/drive/trash', { method: 'DELETE' })

// Адреса скачивания и просмотра: их отдаёт сам сервис (заголовки имени и
// расположения), поэтому это обычные ссылки, а не fetch.
export const downloadURL = (id) => `/api/drive/files/${id}/download`
export const previewURL = (id) => `/api/drive/files/${id}/download?inline=1`

// ── Доступ ───────────────────────────────────────────────────────────────────

const scope = (kind) => (kind === 'folder' ? 'folders' : 'files')

export const getAccess = (kind, id) => apiRequest(`/drive/${scope(kind)}/${id}/access`)

export const shareTo = (kind, id, body) =>
  apiRequest(`/drive/${scope(kind)}/${id}/access`, { method: 'POST', body })

export const createLink = (kind, id) =>
  apiRequest(`/drive/${scope(kind)}/${id}/links`, { method: 'POST' })

export const revokeAccess = (accessId) =>
  apiRequest(`/drive/access/${accessId}`, { method: 'DELETE' })

export const deleteLink = (linkId) =>
  apiRequest(`/drive/links/${linkId}`, { method: 'DELETE' })

// ── Публичная ссылка ─────────────────────────────────────────────────────────

export const getShared = (code) => apiRequest(`/drive/shared/${code}`)
export const getSharedList = (code) => apiRequest(`/drive/shared/${code}/list`)
export const sharedDownloadURL = (code) => `/api/drive/shared/${code}/download`
