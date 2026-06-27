// Ведётся вручную: REST бэкапа живёт в authsvc (back-go/auth).
import { apiRequest } from './client.js'

// exportBackup(sections) — ZIP выбранных разделов; пустой список → все разделы.
export const exportBackup = (sections = []) => {
  const qs = sections.length ? `?sections=${encodeURIComponent(sections.join(','))}` : ''
  return apiRequest(`/backup/export${qs}`, { blob: true })
}

// importBackup(file, sections) — восстановление выбранных разделов из архива.
export const importBackup = (file, sections = []) => {
  const form = new FormData()
  form.append('file', file)
  if (sections.length) form.append('sections', JSON.stringify(sections))
  return apiRequest('/backup/import', { method: 'POST', body: form })
}
