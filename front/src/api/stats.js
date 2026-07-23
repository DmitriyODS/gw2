// Тонкая обёртка над /api/stats/*. Все эндпоинты принимают опциональный
// companyId — для Администратора системы (Сотрудники-менеджеры получают
// данные строго своей компании, фильтрация на бэке).
import { apiRequest } from './client.js'

function qs({ from, to, companyId, userId } = {}) {
  const p = new URLSearchParams()
  if (from != null) p.set('from', from)
  if (to != null) p.set('to', to)
  if (companyId != null) p.set('company_id', companyId)
  if (userId != null) p.set('user_id', userId)
  const s = p.toString()
  return s ? `?${s}` : ''
}

export const getStatsCommon = (from, to, companyId = null, options = {}) =>
  apiRequest('/stats/common' + qs({ from, to, companyId }), options)

export const exportStatsCommon = (from, to, companyId = null) =>
  apiRequest('/stats/common/export' + qs({ from, to, companyId }), { blob: true })

export const getStatsExtended = (from, to, companyId = null, options = {}) =>
  apiRequest('/stats/extended' + qs({ from, to, companyId }), options)

export const exportStatsExtended = (from, to, companyId = null) =>
  apiRequest('/stats/extended/export' + qs({ from, to, companyId }), { blob: true })

export const getStatsProfile = (from, to) =>
  apiRequest('/stats/profile' + qs({ from, to }))

export const getStatsUserTasks = (userId, from, to) =>
  apiRequest('/stats/user-tasks' + qs({ userId, from, to }))

export const getStatsEmployees = (companyId = null) =>
  apiRequest('/stats/employees' + qs({ companyId }))

export const getStatsResponsibles = (companyId = null) =>
  apiRequest('/stats/responsibles' + qs({ companyId }))

// ── Активность сотрудника (руководитель компании) ──
export const getEmployeeActivity = (userId, from, to) =>
  apiRequest(`/stats/employee/${userId}/activity` + qs({ from, to }))

export const getEmployeeActivityFeed = (userId, { from, to, page = 1, perPage = 30 } = {}) => {
  const p = new URLSearchParams()
  if (from != null) p.set('from', from)
  if (to != null) p.set('to', to)
  p.set('page', page)
  p.set('per_page', perPage)
  return apiRequest(`/stats/employee/${userId}/feed?${p.toString()}`)
}

export const exportEmployeeActivity = (userId, from, to) =>
  apiRequest(`/stats/employee/${userId}/export` + qs({ from, to }), { blob: true })
