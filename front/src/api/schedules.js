// Ведётся вручную: REST расписаний живёт в schedulesvc (back-go/schedule).
import { apiRequest } from './client.js'

function qs(params = {}) {
  const sp = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') sp.set(k, v) })
  return sp.toString()
}

// ── Расписания ──
// tab: 'mine' | 'shared'. → { schedules: [...] } (каждое со своими категориями и полями).
export const getSchedules = (tab = 'mine', options = {}) =>
  apiRequest(`/schedules?tab=${tab}`, options)

// → { schedule, items, can_edit } — расписание целиком: занятий десятки, и по
// неделям их фильтрует клиент (utils/scheduleCycle.js).
export const getSchedule = (id) => apiRequest(`/schedules/${id}`)

// body: { name, cycle_weeks?, cycle_anchor?, timezone? }
export const createSchedule = (body) =>
  apiRequest('/schedules', { method: 'POST', body })

// body: любые из { name, cycle_weeks, cycle_anchor, week_labels, timezone, gap_min }.
// Ответ несёт adjusted_items — сколько занятий потеряло номера недель за
// укороченным циклом (они стали еженедельными, о чём раздел говорит вслух).
export const updateSchedule = (id, body) =>
  apiRequest(`/schedules/${id}`, { method: 'PATCH', body })

export const deleteSchedule = (id) =>
  apiRequest(`/schedules/${id}`, { method: 'DELETE' })

// ── Категории (они же раскраска блоков) ──
export const createCategory = (scheduleId, body) =>
  apiRequest(`/schedules/${scheduleId}/categories`, { method: 'POST', body })

export const updateCategory = (scheduleId, categoryId, body) =>
  apiRequest(`/schedules/${scheduleId}/categories/${categoryId}`, { method: 'PATCH', body })

export const deleteCategory = (scheduleId, categoryId) =>
  apiRequest(`/schedules/${scheduleId}/categories/${categoryId}`, { method: 'DELETE' })

// ── Поля карточки занятия ──
// fields: [{ id?, label, type, config, col_span, row_span, show_in_card }].
// Известный id сохраняется: по нему лежат значения в занятиях.
export const replaceFields = (scheduleId, fields) =>
  apiRequest(`/schedules/${scheduleId}/fields`, { method: 'PUT', body: { fields } })

// ── Занятия ──
// body: { weekday, start_min, end_min, title, short, category_id, weeks,
//         repeat_every, repeat_from, repeat_until, data }
export const createItem = (scheduleId, body) =>
  apiRequest(`/schedules/${scheduleId}/items`, { method: 'POST', body })

export const updateItem = (scheduleId, itemId, body) =>
  apiRequest(`/schedules/${scheduleId}/items/${itemId}`, { method: 'PATCH', body })

export const deleteItem = (scheduleId, itemId) =>
  apiRequest(`/schedules/${scheduleId}/items/${itemId}`, { method: 'DELETE' })

// ── Плитка и поиск ──
// Живая плитка рабочего стола: день и текущая минута — клиентские (зона
// расписания сервером не додумывается). → { now, next, busy_min, gap_min, total }.
export const getAgenda = (date, minute, options = {}) =>
  apiRequest(`/schedules/agenda?${qs({ date, minute })}`, options)

export const searchItems = (q, limit = 5, options = {}) =>
  apiRequest(`/schedules/search?${qs({ q, limit })}`, options)

// ── Шаринг (только чтение) ──
export const getShares = (scheduleId) => apiRequest(`/schedules/${scheduleId}/shares`)
export const createShare = (scheduleId) =>
  apiRequest(`/schedules/${scheduleId}/shares`, { method: 'POST' })
export const revokeShare = (scheduleId, shareId) =>
  apiRequest(`/schedules/${scheduleId}/shares/${shareId}`, { method: 'DELETE' })

export const getUserShares = (scheduleId) => apiRequest(`/schedules/${scheduleId}/access`)
// body: { user_id } либо { company_id } — адресат ровно один.
export const shareWith = (scheduleId, body) =>
  apiRequest(`/schedules/${scheduleId}/access`, { method: 'POST', body })
export const unshare = (scheduleId, body) =>
  apiRequest(`/schedules/${scheduleId}/access`, { method: 'DELETE', body })

export const getDirectory = (q = '', limit = 20) =>
  apiRequest(`/schedules/directory?${qs({ q, limit })}`)
export const getCompanies = () => apiRequest('/schedules/companies')

// Публичная ссылка (read-only, без входа): → { schedule, items, can_edit: false }.
export const getSharedSchedule = (code) => apiRequest(`/schedules/shared/${code}`)

// ── Перенос ──
// format: 'xlsx' (таблица, лист на неделю цикла) | 'json' (перенос целиком).
export const exportSchedule = (scheduleId, format = 'xlsx') =>
  apiRequest(`/schedules/${scheduleId}/export?format=${format}`, { blob: true })

// Импорт понимает свой формат и формат прототипа (both/num/den).
export const importInto = (scheduleId, file) =>
  apiRequest(`/schedules/${scheduleId}/import`, { method: 'POST', body: file })

export const importNew = (file, name = '') =>
  apiRequest(`/schedules/import?${qs({ name })}`, { method: 'POST', body: file })
