// Сгенерировано из /apispec.json — не редактировать вручную
// Перегенерировать: npm run gen:api
import { apiRequest } from './client.js'

export const exportBackup = () => apiRequest('/backup/export', { blob: true })

export const importBackup = (file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest('/backup/import', { method: 'POST', body: form })
}
