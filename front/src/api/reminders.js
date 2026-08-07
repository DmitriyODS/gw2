// Ведётся вручную: REST напоминаний живёт в remindersvc (back-go/reminder).
import { apiRequest } from './client.js'

function qs(params = {}) {
  const sp = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') sp.set(k, v) })
  return sp.toString()
}

// scope: 'active' (по умолчанию) | 'done' | 'all'.
export const getReminders = (scope = 'active') =>
  apiRequest(`/reminders?${qs({ scope })}`)

// Ближайшие активные — живая плитка рабочего стола и центр уведомлений.
export const getUpcoming = (limit = 10) =>
  apiRequest(`/reminders/upcoming?${qs({ limit })}`)

// Напоминания, привязанные к записи ежедневника/календаря: раздел, который
// правит запись, обновляет ими свой снимок времени и названия.
export const getLinked = (kind, recordId) =>
  apiRequest(`/reminders/linked?${qs({ kind, record_id: recordId })}`)

export const getReminder = (id) => apiRequest(`/reminders/${id}`)

// body: { title, note?, remind_at (ISO), timezone?, repeat?, link? }.
export const createReminder = (body) =>
  apiRequest('/reminders', { method: 'POST', body })

export const updateReminder = (id, body) =>
  apiRequest(`/reminders/${id}`, { method: 'PATCH', body })

export const deleteReminder = (id) =>
  apiRequest(`/reminders/${id}`, { method: 'DELETE' })

// Отложить на N минут от текущего момента (кнопка в уведомлении).
export const snoozeReminder = (id, minutes = 10) =>
  apiRequest(`/reminders/${id}/snooze`, { method: 'POST', body: { minutes } })

// «Готово»: разовое уходит в журнал, повтор перескакивает на следующий срок.
export const completeReminder = (id) =>
  apiRequest(`/reminders/${id}/done`, { method: 'POST' })
