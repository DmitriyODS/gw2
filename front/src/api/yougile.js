import { apiRequest } from './client.js'

// ── Личный коннект ────────────────────────────────────────────────────────

export const getYougileStatus = () =>
  apiRequest('/yougile/status')

export const connectYougile = ({ login, password, yg_company_id = null }) =>
  apiRequest('/yougile/account', {
    method: 'POST',
    body: { login, password, yg_company_id },
  })

export const disconnectYougile = () =>
  apiRequest('/yougile/account', { method: 'DELETE' })

export const rotateYougile = ({ password }) =>
  apiRequest('/yougile/account/rotate', { method: 'POST', body: { password } })

// ── Админ-визард ─────────────────────────────────────────────────────────

export const lookupYougileCompanies = ({ login, password }) =>
  apiRequest('/yougile/companies/lookup', { method: 'POST', body: { login, password } })

export const listYougileProjects = () =>
  apiRequest('/yougile/projects')

export const listYougileBoards = (projectId) =>
  apiRequest(`/yougile/boards?projectId=${encodeURIComponent(projectId)}`)

export const listYougileColumns = (boardId) =>
  apiRequest(`/yougile/columns?boardId=${encodeURIComponent(boardId)}`)

// ── Настройки компании ───────────────────────────────────────────────────

export const getCompanyYougileSettings = () =>
  apiRequest('/yougile/company-settings')

export const updateCompanyYougileSettings = (payload) =>
  apiRequest('/yougile/company-settings', { method: 'PUT', body: payload })

// ── Импорт / экспорт / отвязка задачи ────────────────────────────────────

export const importYougileTask = (payload) =>
  apiRequest('/yougile/import-task', { method: 'POST', body: payload })

export const exportYougileTask = ({ gw_task_id }) =>
  apiRequest('/yougile/export-task', { method: 'POST', body: { gw_task_id } })

export const unlinkYougileTask = (gw_task_id) =>
  apiRequest(`/yougile/tasks/${gw_task_id}/link`, { method: 'DELETE' })
