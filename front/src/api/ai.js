import { apiRequest } from './client'

export const getAiSettings = (companyId) =>
  apiRequest(`/companies/${companyId}/ai-settings`)

export const updateAiSettings = (companyId, payload) =>
  apiRequest(`/companies/${companyId}/ai-settings`, { method: 'PUT', body: payload })

export const testAiSettings = (companyId) =>
  apiRequest(`/companies/${companyId}/ai-settings/test`, { method: 'POST' })

export const getAiIndexingStatus = (companyId) =>
  apiRequest(`/companies/${companyId}/ai-settings/indexing`)

export const reindexAiTasks = (companyId) =>
  apiRequest(`/companies/${companyId}/ai-settings/reindex-tasks`, { method: 'POST' })

export const getTvFact = () => apiRequest('/ai/tv-fact')
