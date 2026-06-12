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

// Каталог сотрудников конкретной компании (для селекта руководителя в модалке).
// company_id передаём явно — это «обходит» автомат-инжекцию в client.js.
export const getCompanyDirectory = (companyId, q = '') => {
  const params = new URLSearchParams()
  if (companyId != null) params.set('company_id', String(companyId))
  if (q) params.set('q', q)
  const qs = params.toString()
  return apiRequest(`/users/directory${qs ? '?' + qs : ''}`)
}
