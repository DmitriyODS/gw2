// Ведётся вручную: REST бэкапа живёт в authsvc (back-go/auth), в Flask-spec
// его больше нет (MANUAL_TAGS в scripts/gen-api.mjs).
import { apiRequest } from './client.js'

export const exportBackup = () => apiRequest('/backup/export', { blob: true })

export const importBackup = (file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest('/backup/import', { method: 'POST', body: form })
}
