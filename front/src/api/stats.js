// Сгенерировано из /apispec.json — не редактировать вручную
// Перегенерировать: npm run gen:api
import { apiRequest } from './client.js'

export const getStatsCommon = (from, to) => {
  const qs = new URLSearchParams()
  if (from != null) qs.set('from', from)
  if (to != null) qs.set('to', to)
  const q = qs.toString() ? `?${qs}` : ''
  return apiRequest('/stats/common' + q)
}

export const exportStatsCommon = (from, to) => {
  const qs = new URLSearchParams()
  if (from != null) qs.set('from', from)
  if (to != null) qs.set('to', to)
  const q = qs.toString() ? `?${qs}` : ''
  return apiRequest('/stats/common/export' + q, { blob: true })
}

export const getStatsExtended = (from, to) => {
  const qs = new URLSearchParams()
  if (from != null) qs.set('from', from)
  if (to != null) qs.set('to', to)
  const q = qs.toString() ? `?${qs}` : ''
  return apiRequest('/stats/extended' + q)
}

export const exportStatsExtended = (from, to) => {
  const qs = new URLSearchParams()
  if (from != null) qs.set('from', from)
  if (to != null) qs.set('to', to)
  const q = qs.toString() ? `?${qs}` : ''
  return apiRequest('/stats/extended/export' + q, { blob: true })
}

export const getStatsProfile = (from, to) => {
  const qs = new URLSearchParams()
  if (from != null) qs.set('from', from)
  if (to != null) qs.set('to', to)
  const q = qs.toString() ? `?${qs}` : ''
  return apiRequest('/stats/profile' + q)
}
