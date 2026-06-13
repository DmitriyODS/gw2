// Контракт Go-микросервиса авторизации (back-go/auth) — ведётся вручную:
// authsvc не публикует Swagger, npm run gen:api этот файл не трогает.
import { apiRequest } from './client.js'

export const changeDefault = (data) => apiRequest('/auth/change-default', { method: 'POST', body: data })

export const login = (data) => apiRequest('/auth/login', { method: 'POST', body: data })

// _isLogout — пометка для client.js: даже во время выхода этот запрос должен
// дойти до сервера (через refresh, если access протух), чтобы погасить
// refresh-cookie. Остальные 401 при выходе подавляются без шума.
export const logout = () => apiRequest('/auth/logout', { method: 'POST', _isLogout: true })

export const refreshToken = () => apiRequest('/auth/refresh', { method: 'POST' })
