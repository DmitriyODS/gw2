import { apiRequest } from './client'

export const listCompanies = () => apiRequest('/companies')
export const getCompany = (id) => apiRequest(`/companies/${id}`)
export const createCompany = (payload) => apiRequest('/companies', { method: 'POST', body: payload })
export const updateCompany = (id, payload) => apiRequest(`/companies/${id}`, { method: 'PATCH', body: payload })
export const toggleCompanyActive = (id, isActive) =>
  apiRequest(`/companies/${id}/toggle-active`, { method: 'PATCH', body: { is_active: isActive } })
export const deleteCompany = (id) => apiRequest(`/companies/${id}`, { method: 'DELETE' })

// Каталог сотрудников конкретной компании (для селекта руководителя в модалке).
// company_id передаём явно — это «обходит» автомат-инжекцию в client.js.
export const getCompanyDirectory = (companyId, q = '') => {
  const params = new URLSearchParams()
  if (companyId != null) params.set('company_id', String(companyId))
  if (q) params.set('q', q)
  const qs = params.toString()
  return apiRequest(`/users/directory${qs ? '?' + qs : ''}`)
}
