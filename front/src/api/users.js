// Сгенерировано из /apispec.json — не редактировать вручную
// Перегенерировать: npm run gen:api
import { apiRequest } from './client.js'

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
