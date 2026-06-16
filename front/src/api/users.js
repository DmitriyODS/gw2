// Контракт Go-микросервиса авторизации (back-go/auth) — ведётся вручную:
// authsvc не публикует Swagger, npm run gen:api этот файл не трогает.
import { apiRequest } from './client.js'

// Список ВСЕХ пользователей платформы — только для супер-админа.
export const getUsers = () => apiRequest('/users')

export const createUser = (data) => apiRequest('/users', { method: 'POST', body: data })

export const getMe = () => apiRequest('/users/me')

export const updateMe = (data) => apiRequest('/users/me', { method: 'PATCH', body: data })

export const deleteAvatar = () => apiRequest('/users/me/avatar', { method: 'DELETE' })

export const uploadAvatar = (file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest('/users/me/avatar', { method: 'POST', body: form })
}

export const deleteUser = (userId) => apiRequest(`/users/${userId}`, { method: 'DELETE' })

export const getUser = (userId) => apiRequest(`/users/${userId}`)

export const updateUser = (userId, data) => apiRequest(`/users/${userId}`, { method: 'PATCH', body: data })

export const assignRole = (userId, data) => apiRequest(`/users/${userId}/role`, { method: 'PATCH', body: data })

// Каталог сотрудников — доступно любому авторизованному. По умолчанию —
// члены АКТИВНОЙ компании (роль/должность из связки, компания берётся из токена).
// global: true (?all=1) — все видимые пользователи платформы (для старта чата/
// звонка с сотрудником другой компании, контакты).
export const getDirectory = (q = '', excludeSelf = false, { global = false } = {}) => {
  const params = new URLSearchParams()
  if (q) params.set('q', q)
  if (excludeSelf) params.set('exclude_self', 'true')
  if (global) params.set('all', '1')
  const qs = params.toString()
  return apiRequest(`/users/directory${qs ? '?' + qs : ''}`)
}

export const getDirectoryUser = (userId) => apiRequest(`/users/directory/${userId}`)

export const resetUserPassword = (userId) =>
  apiRequest(`/users/${userId}/reset-password`, { method: 'POST' })
