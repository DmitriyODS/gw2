import { apiRequest } from './client'

// Список всех компаний — платформенный эндпоинт (только супер-админ).
export const listCompanies = () => apiRequest('/companies')
// «Мои компании» — где пользователь администратор (раздел «Компании»). Поле
// created_by в каждой компании показывает, создатель ли он (полные права).
export const listMyCompanies = () => apiRequest('/companies/mine')
export const getCompany = (id) => apiRequest(`/companies/${id}`)
// Создать компанию может ЛЮБОЙ авторизованный пользователь: тело
// {name, description?, settings?}. Создатель становится администратором;
// чтобы начать в ней работать — switch-company. DTO компании несёт
// creator/created_by (бывшие director/director_id).
export const createCompany = (payload) => apiRequest('/companies', { method: 'POST', body: payload })
export const updateCompany = (id, payload) => apiRequest(`/companies/${id}`, { method: 'PATCH', body: payload })
// Вкл/выкл компании — только супер-админ (PATCH компании is_active не принимает).
export const toggleCompanyActive = (id, isActive) =>
  apiRequest(`/companies/${id}/toggle-active`, { method: 'PATCH', body: { is_active: isActive } })
export const deleteCompany = (id) => apiRequest(`/companies/${id}`, { method: 'DELETE' })

// Выходные дни компании (0=Пн … 6=Вс) — Администратор компании.
export const getWeekendSettings = (companyId) =>
  apiRequest(`/companies/${companyId}/weekend-settings`)
export const updateWeekendSettings = (companyId, weekendDays) =>
  apiRequest(`/companies/${companyId}/weekend-settings`, {
    method: 'PUT', body: { weekend_days: weekendDays },
  })

// Режим «Мой Groove» (геймификация-питомцы) — Администратор компании.
export const getGrooveSettings = (companyId) =>
  apiRequest(`/companies/${companyId}/groove-settings`)
export const updateGrooveSettings = (companyId, enabled) =>
  apiRequest(`/companies/${companyId}/groove-settings`, {
    method: 'PUT', body: { enabled },
  })

// Каталог сотрудников конкретной компании (для селекта в модалках).
// Супер-админ передаёт company_id явно; обычному юзеру бэк берёт компанию из токена.
export const getCompanyDirectory = (companyId, q = '') => {
  const params = new URLSearchParams()
  if (companyId != null) params.set('company_id', String(companyId))
  if (q) params.set('q', q)
  const qs = params.toString()
  return apiRequest(`/users/directory${qs ? '?' + qs : ''}`)
}

// ── Участники компании (управляет Администратор компании / супер-админ) ──
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

// ── Создание/редактирование сотрудников в компании (только создатель компании
// или супер-админ). Создание заводит новый аккаунт + членство в этой компании. ──
export const createCompanyUser = (companyId, payload) =>
  apiRequest(`/companies/${companyId}/users`, { method: 'POST', body: payload })
export const updateCompanyMember = (companyId, userId, payload) =>
  apiRequest(`/companies/${companyId}/users/${userId}`, { method: 'PATCH', body: payload })
export const resetCompanyMemberPassword = (companyId, userId) =>
  apiRequest(`/companies/${companyId}/users/${userId}/reset-password`, { method: 'POST' })

// ── Ссылка-приглашение (Администратор компании или супер-админ) ──
export const getCompanyInvite = (companyId) => apiRequest(`/companies/${companyId}/invite`)
export const regenerateCompanyInvite = (companyId) =>
  apiRequest(`/companies/${companyId}/invite`, { method: 'POST' })
// Вступление по коду (авторизованный пользователь) — возвращает новую сессию.
export const joinCompanyByCode = (code) =>
  apiRequest(`/companies/join/${encodeURIComponent(code)}`, { method: 'POST' })

// ── Email-приглашения в компанию (создатель компании / супер-админ) ──
export const createCompanyInvite = (companyId, email, roleId) =>
  apiRequest(`/companies/${companyId}/invites`, { method: 'POST', body: { email, role_id: roleId } })
// Превью приглашения по токену (что увидит получатель). Требует авторизации.
export const getInvitePreview = (token) =>
  apiRequest(`/companies/invites/${encodeURIComponent(token)}`)
// Принять приглашение — возвращает новую сессию (переключённую на компанию).
export const acceptCompanyInvite = (token) =>
  apiRequest(`/companies/invites/${encodeURIComponent(token)}/accept`, { method: 'POST' })
