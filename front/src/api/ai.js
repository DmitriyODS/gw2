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

// Включён ли ИИ в активной компании — один флаг (полные ai-настройки читает
// только администратор компании). На нём держатся компанийные фичи: поиск по
// задачам и заметкам, ИИ-инструменты текста, ТВ-факт дня.
export const getAiStatus = () => apiRequest('/ai/status')

// ── Личные ИИ-настройки: ключ ИИ-ассистента ───────────────────────
// Ассистент подключён к КОНКРЕТНОМУ ПОЛЬЗОВАТЕЛЮ, а не к компании: ключ
// заводит себе каждый сам, он переезжает за человеком между компаниями и
// работает, когда активной компании нет вовсе. Сырого ключа сервер не
// отдаёт — только маска key_hint.
export const getMyAiSettings = () => apiRequest('/ai/my-settings')

// payload: { enabled?, api_key?, clear_key?, model_chat?, api_base_url?,
// feat_assistant?, feat_notes? }. Пустой api_key — «не менять»; удаление
// ключа — clear_key: true (тогда ИИ снова работает на платформенном ключе).
export const updateMyAiSettings = (payload) =>
  apiRequest('/ai/my-settings', { method: 'PATCH', body: payload })

// Реальная проверка связи личным ключом: один tiny-chat. { chat, error, latency_ms }.
export const testMyAiSettings = () =>
  apiRequest('/ai/my-settings/test', { method: 'POST' })

// ── Платформенный ИИ (супер-админ, «Аудит платформы») ─────────────
// Глобальный ключ proxy-api, адрес API, модели по умолчанию и каталог с
// ценами: цена модели задаёт стоимость обращения в токенах доступа.
export const getPlatformAi = () => apiRequest('/ai/platform')

export const updatePlatformAi = (payload) =>
  apiRequest('/ai/platform', { method: 'PATCH', body: payload })

export const testPlatformAi = () => apiRequest('/ai/platform/test', { method: 'POST' })
