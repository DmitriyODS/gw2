// Ведётся вручную: REST «Диска» живёт в drivesvc (back-go/drive).
import { apiRequest } from './client.js'
import { useAuthStore } from '@/stores/auth.js'

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

   Маленькие файлы уходят одним multipart-запросом. Большие — ЧАСТЯМИ: сотни
   мегабайт одним запросом упираются в таймауты прокси и не переживают обрыв
   сети, а прогресс по ним виден только «до отправки», без хода на сервере.
   Куски идут по порядку сырыми байтами; сервер держит позицию и при
   рассинхроне отвечает, с какого места продолжать.

   С onProgress мелкий файл идёт через XHR — только он умеет отдавать ход
   отправки; без него — обычным apiRequest, который проходит общую цепочку
   (refresh токена, обработка ошибок). */

// Больше этого размера — грузим частями.
export const CHUNK_THRESHOLD = 8 * 1024 * 1024
// Размер куска: компромисс между числом запросов и потерями при обрыве.
export const CHUNK_SIZE = 4 * 1024 * 1024

export function uploadFile(file, folderId = null, { onProgress, signal } = {}) {
  if (file.size > CHUNK_THRESHOLD) {
    return uploadInChunks(file, folderId, { onProgress, signal })
  }
  const form = new FormData()
  form.append('file', file)
  const url = `/api/drive/files?${qs({ folder_id: folderId })}`
  if (!onProgress) {
    return apiRequest(`/drive/files?${qs({ folder_id: folderId })}`, { method: 'POST', body: form, signal })
  }
  const token = useAuthStore().token

  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', url)
    xhr.withCredentials = true
    // Токен берём из стора — там же, где его держит apiRequest: своего
    // интерцептора у XHR нет.
    if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded / e.total)
    }
    xhr.onload = () => {
      let data = null
      try { data = JSON.parse(xhr.responseText) } catch { /* пустой ответ */ }
      if (xhr.status >= 200 && xhr.status < 300) resolve(data)
      else reject(Object.assign(new Error(data?.message || 'Не удалось загрузить файл'), {
        status: xhr.status, error: data?.error, ...data,
      }))
    }
    xhr.onerror = () => reject(new Error('Не удалось загрузить файл'))
    if (signal) signal.addEventListener('abort', () => xhr.abort(), { once: true })
    xhr.send(form)
  })
}

// Загрузка частями: завести → куски по порядку → собрать. Отмена и обрыв
// убирают заготовку на сервере, чтобы куски не лежали мёртвым грузом.
async function uploadInChunks(file, folderId, { onProgress, signal } = {}) {
  const { upload_id: uploadId } = await apiRequest('/drive/uploads', {
    method: 'POST',
    body: { name: file.name, mime: file.type || '', size: file.size, folder_id: folderId },
    signal,
  })

  try {
    let offset = 0
    while (offset < file.size) {
      const end = Math.min(offset + CHUNK_SIZE, file.size)
      const chunk = file.slice(offset, end)
      try {
        const res = await apiRequest(`/drive/uploads/${uploadId}?offset=${offset}`, {
          method: 'PUT',
          body: chunk,
          headers: { 'Content-Type': 'application/octet-stream' },
          // Кусок на медленной сети живёт дольше обычного запроса.
          timeout: 120000,
          signal,
        })
        offset = res.received
      } catch (e) {
        // Сервер знает, сколько принял: продолжаем с его отметки, а не с нуля.
        if (e?.error === 'CHUNK_OFFSET' && typeof e.received === 'number') {
          offset = e.received
          continue
        }
        throw e
      }
      onProgress?.(offset / file.size)
    }
    // Сборка большого файла — запись в хранилище целиком: ждём дольше.
    return await apiRequest(`/drive/uploads/${uploadId}/complete`, { method: 'POST', timeout: 180000 })
  } catch (e) {
    apiRequest(`/drive/uploads/${uploadId}`, { method: 'DELETE' }).catch(() => {})
    throw e
  }
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
