import { apiRequest } from './client.js'

// YouGile-эндпоинты под капотом делают несколько последовательных HTTP-запросов
// к ru.yougile.com (auth/companies → auth/keys → users/me) с возможными ретраями
// на 429/5xx. Дефолтных 8 секунд клиента не хватает — фронт abort'ил запрос,
// пока бэк ещё ждал YouGile. Поднимаем планку для всех ручек интеграции.
const YG_TIMEOUT = 60_000

const ygReq = (path, opts = {}) =>
  apiRequest(path, { timeout: YG_TIMEOUT, ...opts })

// ── Личный коннект ────────────────────────────────────────────────────────

export const getYougileStatus = () =>
  ygReq('/yougile/status')

export const connectYougile = ({ login, password, yg_company_id = null }) =>
  ygReq('/yougile/account', {
    method: 'POST',
    body: { login, password, yg_company_id },
  })

export const disconnectYougile = () =>
  ygReq('/yougile/account', { method: 'DELETE' })

export const rotateYougile = ({ password }) =>
  ygReq('/yougile/account/rotate', { method: 'POST', body: { password } })

// ── Админ-визард ─────────────────────────────────────────────────────────

export const lookupYougileCompanies = ({ login, password }) =>
  ygReq('/yougile/companies/lookup', { method: 'POST', body: { login, password } })

export const listYougileProjects = () =>
  ygReq('/yougile/projects')

export const listYougileBoards = (projectId) =>
  ygReq(`/yougile/boards?projectId=${encodeURIComponent(projectId)}`)

export const listYougileColumns = (boardId) =>
  ygReq(`/yougile/columns?boardId=${encodeURIComponent(boardId)}`)

// ── Настройки компании ───────────────────────────────────────────────────

export const getCompanyYougileSettings = () =>
  ygReq('/yougile/company-settings')

export const updateCompanyYougileSettings = (payload) =>
  ygReq('/yougile/company-settings', { method: 'PUT', body: payload })

// ── Импорт / экспорт / отвязка задачи ────────────────────────────────────

export const importYougileTask = (payload) =>
  ygReq('/yougile/import-task', { method: 'POST', body: payload })

export const exportYougileTask = ({ gw_task_id }) =>
  ygReq('/yougile/export-task', { method: 'POST', body: { gw_task_id } })

export const unlinkYougileTask = (gw_task_id) =>
  ygReq(`/yougile/tasks/${gw_task_id}/link`, { method: 'DELETE' })
