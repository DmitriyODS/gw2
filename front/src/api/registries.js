// Ведётся вручную: REST реестров живёт в registrysvc (back-go/registry).
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

/* filterParams — фильтры колонок в query: filter=<field_id>:<op>[:v1|v2].
   Значения разделены «|», а не запятой: в тексте фильтра запятая обычна.
   Зеркало разбора на сервере (transport/http/handlers.go:columnFilters). */
function filterParams(search, filters = []) {
  for (const f of filters) {
    if (!f?.field_id || !f.op) continue
    const values = (f.values || []).filter((v) => v !== '' && v != null)
    search.append('filter', values.length
      ? `${f.field_id}:${f.op}:${values.join('|')}`
      : `${f.field_id}:${f.op}`)
  }
  return search
}

/* selectionParams — набор записей под массовую операцию. Либо перечень id,
   либо «всё по фильтру экрана» за вычетом снятых: выбор переживает страницы, и
   тысяча записей не превращается в тысячу id в адресе. */
function selectionParams(search, selection = {}, filter = {}) {
  if (selection.all) {
    search.set('all', 'true')
    if (filter.search) search.set('search', filter.search)
    if (filter.section) search.set('section', filter.section)
    if (selection.exclude?.length) search.set('exclude', selection.exclude.join(','))
    filterParams(search, filter.filters)
  } else if (selection.ids?.length) {
    search.set('ids', selection.ids.join(','))
  }
  return search
}

// ── Реестры ──
// scope: all | mine | shared | company — вкладки левой панели.
export const getRegistries = (scope = 'all', options = {}) =>
  apiRequest(`/registries?${qs({ scope })}`, options)

export const getRegistry = (id) => apiRequest(`/registries/${id}`)

// Глобальный поиск по записям ВСЕХ доступных реестров (строка Hola): свои,
// расшаренные лично и расшаренные компаниям — одним запросом.
export const searchRecords = (q, limit = 5, options = {}) =>
  apiRequest(`/registries/search?${qs({ q, limit })}`, options)

export const createRegistry = (name, accounting = false) =>
  apiRequest('/registries', { method: 'POST', body: { name, accounting } })

/* patch: { name?, section_field_id?, accounting? }. Ключи section_field_id и
   accounting шлём, ТОЛЬКО когда их меняют: сервер отличает «не трогать» от
   null («выключить подразделы»), иначе переименование сбрасывало бы настройку. */
export const updateRegistry = (id, patch) =>
  apiRequest(`/registries/${id}`, { method: 'PATCH', body: patch })

export const deleteRegistry = (id) =>
  apiRequest(`/registries/${id}`, { method: 'DELETE' })

// Полная замена набора полей реестра (добавление/удаление/реордер/раскладка).
export const replaceFields = (id, fields) =>
  apiRequest(`/registries/${id}/fields`, { method: 'PUT', body: { fields } })

// ── Записи ──
// params: { search, sort, order, section, page, per_page, filters: [{field_id, op, values}] }.
export const getRecords = (registryId, params = {}, options = {}) => {
  const { filters, ...rest } = params
  const search = filterParams(qs(rest), filters)
  return apiRequest(`/registries/${registryId}/records?${search}`, options)
}

export const getRecord = (registryId, recordId) =>
  apiRequest(`/registries/${registryId}/records/${recordId}`)

export const createRecord = (registryId, data) =>
  apiRequest(`/registries/${registryId}/records`, { method: 'POST', body: { data } })

export const updateRecord = (registryId, recordId, data) =>
  apiRequest(`/registries/${registryId}/records/${recordId}`, { method: 'PATCH', body: { data } })

export const deleteRecord = (registryId, recordId) =>
  apiRequest(`/registries/${registryId}/records/${recordId}`, { method: 'DELETE' })

export const bulkDeleteRecords = (registryId, selection, filter = {}) =>
  apiRequest(`/registries/${registryId}/records/bulk-delete`, {
    method: 'POST',
    body: selection.all
      ? {
          all: true,
          exclude: selection.exclude || [],
          search: filter.search || '',
          section: filter.section || '',
          filters: filter.filters || [],
        }
      : { ids: selection.ids || [] },
  })

// Выгрузка записей в xlsx: колонки плюс тот же набор, что показан на экране.
// Возвращает Response (blob: true) для скачивания файла.
export const exportRecords = (registryId, { fields = [], selection = {}, filter = {} } = {}) => {
  const search = selectionParams(qs(), selection, filter)
  if (fields.length) search.set('fields', fields.join(','))
  return apiRequest(`/registries/${registryId}/export?${search}`, { blob: true })
}

// ── Учётный реестр ──
export const getIssues = (registryId, recordId) =>
  apiRequest(`/registries/${registryId}/records/${recordId}/issues`)

// body: { holder_name, holder_phone?, holder_user_id?, due_at?, comment? }
export const issueRecord = (registryId, recordId, body) =>
  apiRequest(`/registries/${registryId}/records/${recordId}/issue`, { method: 'POST', body })

export const extendIssue = (registryId, recordId, body) =>
  apiRequest(`/registries/${registryId}/records/${recordId}/extend`, { method: 'POST', body })

export const returnIssue = (registryId, recordId, comment = '') =>
  apiRequest(`/registries/${registryId}/records/${recordId}/return`, {
    method: 'POST', body: { comment },
  })

// ── Загрузка файлов ──
/* Мелкое уходит одним запросом, крупное — частями с докачкой и прогрессом
   (общий загрузчик платформы). Порог и вся механика — в utils/chunkUpload.js.
   onProgress(0..1) — для индикатора. */
export const uploadFile = (registryId, file, { onProgress, signal } = {}) =>
  uploadFileTo({
    file,
    onProgress,
    signal,
    directUrl: `/registries/uploads?${qs({ registry_id: registryId })}`,
    chunkBase: '/registries/uploads',
    scope: { registry_id: registryId },
  })

// ── Шаринг: внешние ссылки ──
export const getShares = (registryId) => apiRequest(`/registries/${registryId}/shares`)

// params: { name?, access: view|edit|admin, require_auth? }
export const createShare = (registryId, params = {}) =>
  apiRequest(`/registries/${registryId}/shares`, { method: 'POST', body: params })

export const updateShare = (registryId, shareId, params = {}) =>
  apiRequest(`/registries/${registryId}/shares/${shareId}`, { method: 'PATCH', body: params })

export const revokeShare = (registryId, shareId) =>
  apiRequest(`/registries/${registryId}/shares/${shareId}`, { method: 'DELETE' })

// Журнал переходов по ссылке: кто, когда и с какого адреса открывал.
export const getShareVisits = (registryId, shareId, limit = 200) =>
  apiRequest(`/registries/${registryId}/shares/${shareId}/visits?${qs({ limit })}`)

// ── Шаринг: люди и компании ──
export const getAccess = (registryId) => apiRequest(`/registries/${registryId}/access`)

// targets: [{ user_id?, company_id?, access }]
export const shareWith = (registryId, targets) =>
  apiRequest(`/registries/${registryId}/access`, { method: 'POST', body: { targets } })

export const unshare = (registryId, { userId, companyId } = {}) =>
  apiRequest(`/registries/${registryId}/access?${qs({ user_id: userId, company_id: companyId })}`,
    { method: 'DELETE' })

// Кандидаты в адресаты: коллеги из компаний спрашивающего.
export const getDirectory = (q = '', limit = 50) =>
  apiRequest(`/registries/directory?${qs({ q, limit })}`)

// Компании, которым можно отдать реестр (те, где человек состоит).
export const getCompanies = () => apiRequest('/registries/companies')

// ── Публичный доступ по коду ──
export const getSharedRegistry = (code) => apiRequest(`/registries/shared/${code}`)

export const getSharedRecords = (code, params = {}, options = {}) => {
  const { filters, ...rest } = params
  const search = filterParams(qs(rest), filters)
  return apiRequest(`/registries/shared/${code}/records?${search}`, options)
}

export const exportSharedRecords = (code, { fields = [], selection = {}, filter = {} } = {}) => {
  const search = selectionParams(qs(), selection, filter)
  if (fields.length) search.set('fields', fields.join(','))
  return apiRequest(`/registries/shared/${code}/export?${search}`, { blob: true })
}

// Правка по ссылке уровня edit (сервер сам откажет ссылке на просмотр).
export const createSharedRecord = (code, data) =>
  apiRequest(`/registries/shared/${code}/records`, { method: 'POST', body: { data } })

export const updateSharedRecord = (code, recordId, data) =>
  apiRequest(`/registries/shared/${code}/records/${recordId}`, { method: 'PATCH', body: { data } })

// ── Учётный реестр по ссылке (уровень edit и выше) ──
export const getSharedIssues = (code, recordId) =>
  apiRequest(`/registries/shared/${code}/records/${recordId}/issues`)

export const issueSharedRecord = (code, recordId, body) =>
  apiRequest(`/registries/shared/${code}/records/${recordId}/issue`, { method: 'POST', body })

export const extendSharedIssue = (code, recordId, body) =>
  apiRequest(`/registries/shared/${code}/records/${recordId}/extend`, { method: 'POST', body })

export const returnSharedIssue = (code, recordId, comment = '') =>
  apiRequest(`/registries/shared/${code}/records/${recordId}/return`, {
    method: 'POST', body: { comment },
  })

/* Правка самого реестра по ссылке уровня «администрирование»: те же тела, что
   и у своих ручек, — сервер сам сверяет уровень ссылки. */
export const updateSharedRegistry = (code, patch) =>
  apiRequest(`/registries/shared/${code}`, { method: 'PATCH', body: patch })

export const replaceSharedFields = (code, fields) =>
  apiRequest(`/registries/shared/${code}/fields`, { method: 'PUT', body: { fields } })

export const deleteSharedRecord = (code, recordId) =>
  apiRequest(`/registries/shared/${code}/records/${recordId}`, { method: 'DELETE' })

export const uploadSharedFile = (code, file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest(`/registries/shared/${code}/uploads`, { method: 'POST', body: form })
}
