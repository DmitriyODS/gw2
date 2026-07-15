// Спаривание устройств: вход по QR и авторизация ТВ-киоска по коду/QR.
// Контракт authsvc (back-go/auth), ведётся вручную.
import { apiRequest } from './client.js'

// Инициатор (устройство без входа / ТВ-киоск) заводит спаривание.
// kind: 'login' (по умолчанию) | 'tv'. Возвращает {code, secret, kind, expires_in_sec}.
export const linkStart = (kind) =>
  apiRequest('/auth/link/start', { method: 'POST', body: { kind } })

// Тип и статус кода — для экрана подтверждения (без секрета).
export const linkInfo = (code) =>
  apiRequest(`/auth/link/info?code=${encodeURIComponent(code)}`, { method: 'GET' })

// Подтверждение спаривания авторизованным пользователем (для ТВ — под активной
// компанией). Требует авторизации.
export const linkApprove = (code) =>
  apiRequest('/auth/link/approve', { method: 'POST', body: { code } })

// Опрос статуса инициатором; после approve вернёт {status:'ok', session}.
export const linkClaim = (code, secret) =>
  apiRequest('/auth/link/claim', { method: 'POST', body: { code, secret } })
