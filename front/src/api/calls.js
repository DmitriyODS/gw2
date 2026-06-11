import { apiRequest } from './client.js'

export function getActiveCall() {
  return apiRequest('/calls/active')
}

/** LiveKit-токен для входа/возврата в живой звонок (участник платформы). */
export function getCallToken(callId) {
  return apiRequest(`/calls/${callId}/token`, { method: 'POST' })
}

/** Публичная информация о звонке по ссылке-приглашению (доступна без входа). */
export function getJoinInfo(code) {
  return apiRequest(`/calls/join/${encodeURIComponent(code)}`)
}

/** Вход по ссылке: гость передаёт name; авторизованный входит под собой. */
export function joinCallByCode(code, { name } = {}) {
  return apiRequest(`/calls/join/${encodeURIComponent(code)}`, {
    method: 'POST',
    body: name ? { name } : {},
  })
}
