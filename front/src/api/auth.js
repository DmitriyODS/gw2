// Контракт Go-микросервиса авторизации (back-go/auth) — ведётся вручную:
// authsvc не публикует Swagger, npm run gen:api этот файл не трогает.
import { apiRequest } from './client.js'

export const changeDefault = (data) => apiRequest('/auth/change-default', { method: 'POST', body: data })

export const login = (data) => apiRequest('/auth/login', { method: 'POST', body: data })

export const logout = () => apiRequest('/auth/logout', { method: 'POST' })

export const refreshToken = () => apiRequest('/auth/refresh', { method: 'POST' })
