// Ведётся вручную: REST ежедневников живёт в diarysvc (back-go/diary).
import { apiRequest } from './client.js'

function qs(params = {}) {
  const sp = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') sp.set(k, v) })
  return sp.toString()
}

// ── Ежедневники ──
// tab: 'mine' | 'shared'. Возвращает { diaries: [...] }.
export const getDiaries = (tab = 'mine', options = {}) =>
  apiRequest(`/diaries?tab=${tab}`, options)

export const getDiary = (id) => apiRequest(`/diaries/${id}`)

export const createDiary = (name) =>
  apiRequest('/diaries', { method: 'POST', body: { name } })

export const updateDiary = (id, name) =>
  apiRequest(`/diaries/${id}`, { method: 'PATCH', body: { name } })

export const deleteDiary = (id) =>
  apiRequest(`/diaries/${id}`, { method: 'DELETE' })

// ── Записи ──
// params: { archived?: 0|1, from?: 'YYYY-MM-DD', to?, search? }. → { items: [...] }.
export const getEntries = (diaryId, params = {}, options = {}) =>
  apiRequest(`/diaries/${diaryId}/records?${qs(params)}`, options)

export const getEntry = (diaryId, entryId) =>
  apiRequest(`/diaries/${diaryId}/records/${entryId}`)

// body: { entry_date, start_min, end_min, title, description }.
export const createEntry = (diaryId, body) =>
  apiRequest(`/diaries/${diaryId}/records`, { method: 'POST', body })

export const updateEntry = (diaryId, entryId, body) =>
  apiRequest(`/diaries/${diaryId}/records/${entryId}`, { method: 'PATCH', body })

export const setEntryDone = (diaryId, entryId, done) =>
  apiRequest(`/diaries/${diaryId}/records/${entryId}/done`, { method: 'PATCH', body: { done } })

export const linkEntryTask = (diaryId, entryId, taskId) =>
  apiRequest(`/diaries/${diaryId}/records/${entryId}/link`, { method: 'PATCH', body: { task_id: taskId } })

export const deleteEntry = (diaryId, entryId) =>
  apiRequest(`/diaries/${diaryId}/records/${entryId}`, { method: 'DELETE' })

export const bulkDeleteEntries = (diaryId, ids) =>
  apiRequest(`/diaries/${diaryId}/records/bulk-delete`, { method: 'POST', body: { ids } })

// Экспорт записей в xlsx. params: { archived?, from?, to?, search?, ids?: [] }.
export const exportEntries = (diaryId, { archived = 0, from = '', to = '', search = '', ids = [] } = {}) => {
  const sp = new URLSearchParams()
  if (archived) sp.set('archived', '1')
  if (from) sp.set('from', from)
  if (to) sp.set('to', to)
  if (search) sp.set('search', search)
  if (ids.length) sp.set('ids', ids.join(','))
  return apiRequest(`/diaries/${diaryId}/export?${sp}`, { blob: true })
}

// ── Публичные ссылки (владелец) ──
export const getShares = (diaryId) => apiRequest(`/diaries/${diaryId}/shares`)
export const createShare = (diaryId) =>
  apiRequest(`/diaries/${diaryId}/shares`, { method: 'POST' })
export const revokeShare = (diaryId, shareId) =>
  apiRequest(`/diaries/${diaryId}/shares/${shareId}`, { method: 'DELETE' })

// ── Адресный доступ (поделиться с пользователем) ──
export const getMembers = (diaryId) => apiRequest(`/diaries/${diaryId}/members`)
export const addMember = (diaryId, userId) =>
  apiRequest(`/diaries/${diaryId}/members`, { method: 'POST', body: { user_id: userId } })
export const removeMember = (diaryId, userId) =>
  apiRequest(`/diaries/${diaryId}/members/${userId}`, { method: 'DELETE' })

// ── Публичный доступ по коду (без авторизации) ──
export const getSharedDiary = (code) => apiRequest(`/diaries/shared/${code}`)
export const getSharedEntries = (code, params = {}, options = {}) =>
  apiRequest(`/diaries/shared/${code}/records?${qs(params)}`, options)
export const exportSharedEntries = (code, { archived = 0, from = '', to = '', search = '' } = {}) => {
  const sp = new URLSearchParams()
  if (archived) sp.set('archived', '1')
  if (from) sp.set('from', from)
  if (to) sp.set('to', to)
  if (search) sp.set('search', search)
  return apiRequest(`/diaries/shared/${code}/export?${sp}`, { blob: true })
}
