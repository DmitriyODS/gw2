// Ведётся вручную: REST форм и опросов живёт в formsvc (back-go/forms).
import { apiRequest } from './client.js'
import { uploadFileTo } from '@/utils/chunkUpload.js'

// qs — параметры запроса без пустых значений (сервер трактует их как «не задано»).
function qs(params = {}) {
  const out = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v == null || v === '') continue
    out.set(k, v)
  }
  return out
}

// ── Формы ──
// scope: all | mine | assigned | shared — вкладки левой панели.
export const getForms = (scope = 'all', options = {}) =>
  apiRequest(`/forms?${qs({ scope })}`, options)

export const getForm = (id) => apiRequest(`/forms/${id}`)

// Глобальный поиск по доступным формам (строка Hola).
export const searchForms = (q, limit = 5, options = {}) =>
  apiRequest(`/forms/search?${qs({ q, limit })}`, options)

export const createForm = (title, quiz = false) =>
  apiRequest('/forms', { method: 'POST', body: { title, quiz } })

/* patch: любые настройки формы. Ключи шлём ТОЛЬКО те, что меняем: сервер
   отличает «не трогать» от явного значения, иначе переименование сбрасывало бы
   настройки приёма ответов. */
export const updateForm = (id, patch) =>
  apiRequest(`/forms/${id}`, { method: 'PATCH', body: patch })

export const deleteForm = (id) => apiRequest(`/forms/${id}`, { method: 'DELETE' })

// Копия формы со структурой, но без ответов.
export const duplicateForm = (id) => apiRequest(`/forms/${id}/duplicate`, { method: 'POST' })

/* Полная замена структуры. Ветвление уезжает ПОЗИЦИЯМИ разделов (next_index у
   раздела и "#<позиция>" в config.targets вопроса): у только что добавленного
   раздела id ещё нет — его выдаёт сервер. */
export const saveStructure = (id, sections) =>
  apiRequest(`/forms/${id}/structure`, { method: 'PUT', body: { sections } })

// ── Заполнение ──
export const getFill = (id) => apiRequest(`/forms/${id}/fill`)

// body: { answers: {question_id: value}, email?, name? }
export const submitResponse = (id, body) =>
  apiRequest(`/forms/${id}/responses`, { method: 'POST', body })

export const updateMyResponse = (id, body) =>
  apiRequest(`/forms/${id}/responses/mine`, { method: 'PATCH', body })

// ── Собранные ответы ──
// params: { search, sort: created_at|score, order, page, per_page }
export const getResponses = (id, params = {}, options = {}) =>
  apiRequest(`/forms/${id}/responses?${qs(params)}`, options)

export const getResponse = (id, responseId) =>
  apiRequest(`/forms/${id}/responses/${responseId}`)

export const deleteResponse = (id, responseId) =>
  apiRequest(`/forms/${id}/responses/${responseId}`, { method: 'DELETE' })

// selection: { ids: [] } либо { all: true } — «очистить все ответы».
export const deleteResponses = (id, selection = {}) =>
  apiRequest(`/forms/${id}/responses/bulk-delete`, { method: 'POST', body: selection })

// Открыть оценки теста отвечающим (0 — сразу все ответы).
export const publishGrades = (id, responseId = 0) =>
  apiRequest(`/forms/${id}/grades`, { method: 'POST', body: { response_id: responseId } })

export const getSummary = (id) => apiRequest(`/forms/${id}/summary`)

// Контроль исполнения: кто из назначенных ответил, а кто нет.
export const getProgress = (id) => apiRequest(`/forms/${id}/progress`)

// Выгрузка ответов в xlsx. Возвращает Response (blob: true) для скачивания.
export const exportResponses = (id) => apiRequest(`/forms/${id}/export`, { blob: true })

// ── Загрузка файлов ответа ──
/* Мелкое уходит одним запросом, крупное — частями с докачкой и прогрессом
   (общий загрузчик платформы). onProgress(0..1) — для индикатора. */
export const uploadFile = (formId, questionId, file, { onProgress, signal } = {}) =>
  uploadFileTo({
    file,
    onProgress,
    signal,
    directUrl: `/forms/uploads?${qs({ form_id: formId, question_id: questionId })}`,
    chunkBase: '/forms/uploads',
    scope: { form_id: formId, question_id: questionId },
  })

// ── Шаринг: внешние ссылки ──
export const getShares = (id) => apiRequest(`/forms/${id}/shares`)

// params: { name?, require_auth? }
export const createShare = (id, params = {}) =>
  apiRequest(`/forms/${id}/shares`, { method: 'POST', body: params })

export const updateShare = (id, shareId, params = {}) =>
  apiRequest(`/forms/${id}/shares/${shareId}`, { method: 'PATCH', body: params })

export const revokeShare = (id, shareId) =>
  apiRequest(`/forms/${id}/shares/${shareId}`, { method: 'DELETE' })

export const getShareVisits = (id, shareId, limit = 200) =>
  apiRequest(`/forms/${id}/shares/${shareId}/visits?${qs({ limit })}`)

// ── Доступ и назначения ──
export const getAccess = (id) => apiRequest(`/forms/${id}/access`)

// targets: [{ user_id?, company_id?, access: respond|view|edit, due_at? }]
export const shareWith = (id, targets) =>
  apiRequest(`/forms/${id}/access`, { method: 'POST', body: { targets } })

export const unshare = (id, { userId, companyId } = {}) =>
  apiRequest(`/forms/${id}/access?${qs({ user_id: userId, company_id: companyId })}`,
    { method: 'DELETE' })

// Кандидаты в адресаты: коллеги из компаний спрашивающего.
export const getDirectory = (q = '', limit = 50) =>
  apiRequest(`/forms/directory?${qs({ q, limit })}`)

// Компании, которым можно назначить форму (те, где человек состоит).
export const getCompanies = () => apiRequest('/forms/companies')

// ── Публичный доступ по коду ──
export const getSharedForm = (code) => apiRequest(`/forms/shared/${code}`)

export const submitSharedResponse = (code, body) =>
  apiRequest(`/forms/shared/${code}/responses`, { method: 'POST', body })

// По ссылке файл едет одним запросом: гостю чанковые ручки недоступны (они
// требуют входа), а вложения анкет — обычные документы и снимки.
export const uploadSharedFile = (code, file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest(`/forms/shared/${code}/uploads`, { method: 'POST', body: form })
}
