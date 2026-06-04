import { apiRequest } from './client'

export const listCompanies = () => apiRequest('/companies')
export const getCompany = (id) => apiRequest(`/companies/${id}`)
export const createCompany = (payload) => apiRequest('/companies', { method: 'POST', body: payload })
export const updateCompany = (id, payload) => apiRequest(`/companies/${id}`, { method: 'PATCH', body: payload })
export const toggleCompanyActive = (id, isActive) =>
  apiRequest(`/companies/${id}/toggle-active`, { method: 'PATCH', body: { is_active: isActive } })
export const deleteCompany = (id) => apiRequest(`/companies/${id}`, { method: 'DELETE' })
