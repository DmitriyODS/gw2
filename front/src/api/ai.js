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

// ИИ-инструменты текста (заметки): одна операция над фрагментом.
// action: improve|fix|rephrase|shorten|expand|simplify|summarize|bullets|
// continue|tone|translate; style — тон (formal|friendly|confident|casual)
// или язык перевода (en|ru) для tone/translate. Ответ: { text }.
export const transformText = ({ action, text, style = null }) =>
  apiRequest('/ai/text-tools', { method: 'POST', body: { action, text, style } })

// Корректура орфографии/пунктуации всей заметки: массив текстовых сегментов →
// исправленный массив той же длины (клиент подменяет узлы по индексу). { segments }.
export const proofread = (segments) =>
  apiRequest('/ai/proofread', { method: 'POST', body: { segments } })
