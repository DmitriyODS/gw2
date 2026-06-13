import { apiRequest } from './client'

export const listCompanies = () => apiRequest('/companies')
export const getCompany = (id) => apiRequest(`/companies/${id}`)
export const createCompany = (payload) => apiRequest('/companies', { method: 'POST', body: payload })
export const updateCompany = (id, payload) => apiRequest(`/companies/${id}`, { method: 'PATCH', body: payload })
export const toggleCompanyActive = (id, isActive) =>
  apiRequest(`/companies/${id}/toggle-active`, { method: 'PATCH', body: { is_active: isActive } })
export const deleteCompany = (id) => apiRequest(`/companies/${id}`, { method: 'DELETE' })

// Выходные дни компании (0=Пн … 6=Вс). Руководитель — своей компании,
// Администратор системы — любой.
export const getWeekendSettings = (companyId) =>
  apiRequest(`/companies/${companyId}/weekend-settings`)
export const updateWeekendSettings = (companyId, weekendDays) =>
  apiRequest(`/companies/${companyId}/weekend-settings`, {
    method: 'PUT', body: { weekend_days: weekendDays },
  })

// Режим «Мой Groove» (геймификация-питомцы). Руководитель — своей компании,
// Администратор системы — любой.
export const getGrooveSettings = (companyId) =>
  apiRequest(`/companies/${companyId}/groove-settings`)
export const updateGrooveSettings = (companyId, enabled) =>
  apiRequest(`/companies/${companyId}/groove-settings`, {
    method: 'PUT', body: { enabled },
  })

// Каталог сотрудников конкретной компании (для селекта руководителя в модалке).
// company_id передаём явно — это «обходит» автомат-инжекцию в client.js.
export const getCompanyDirectory = (companyId, q = '') => {
  const params = new URLSearchParams()
  if (companyId != null) params.set('company_id', String(companyId))
  if (q) params.set('q', q)
  const qs = params.toString()
  return apiRequest(`/users/directory${qs ? '?' + qs : ''}`)
}

// ── Участники компании (управляет Администратор системы в карточке компании) ──
export const listCompanyMembers = (companyId) => apiRequest(`/companies/${companyId}/members`)
export const getCompanyCandidates = (companyId, q = '') => {
  const params = new URLSearchParams()
  if (q) params.set('q', q)
  const qs = params.toString()
  return apiRequest(`/companies/${companyId}/members/candidates${qs ? '?' + qs : ''}`)
}
export const addCompanyMember = (companyId, userId, roleId) =>
  apiRequest(`/companies/${companyId}/members`, {
    method: 'POST', body: { user_id: userId, role_id: roleId },
  })
export const setMemberRole = (companyId, userId, roleId) =>
  apiRequest(`/companies/${companyId}/members/${userId}`, {
    method: 'PATCH', body: { role_id: roleId },
  })
export const removeCompanyMember = (companyId, userId) =>
  apiRequest(`/companies/${companyId}/members/${userId}`, { method: 'DELETE' })

// ── Ссылка-приглашение (Админ системы или Руководитель компании) ──
export const getCompanyInvite = (companyId) => apiRequest(`/companies/${companyId}/invite`)
export const regenerateCompanyInvite = (companyId) =>
  apiRequest(`/companies/${companyId}/invite`, { method: 'POST' })
// Вступление по коду (авторизованный пользователь) — возвращает новую сессию.
export const joinCompanyByCode = (code) =>
  apiRequest(`/companies/join/${encodeURIComponent(code)}`, { method: 'POST' })
