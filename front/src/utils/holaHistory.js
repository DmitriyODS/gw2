/**
 * История запросов Hola — то, что показывает пустая вкладка «Поиск».
 *
 * Личный след работы за этим устройством (как журнал активности рабочего
 * стола), поэтому localStorage, а не сервер: между устройствами не едет и
 * чистится при выходе из системы.
 */
import { storageGetJSON, storageSetJSON } from './storage.js'

const KEY = 'gw_hola_history'
const LIMIT = 12

export function loadHistory() {
  const raw = storageGetJSON(KEY, [])
  if (!Array.isArray(raw)) return []
  return raw
    .filter((r) => r && typeof r.text === 'string' && r.text.trim())
    .slice(0, LIMIT)
}

/**
 * Кладёт запрос наверх списка. Повтор поднимает прежнюю строку с новым
 * временем, а не плодит дубли.
 */
export function pushHistory(text) {
  const value = String(text || '').trim()
  if (!value) return loadHistory()
  const rest = loadHistory().filter((r) => r.text.toLowerCase() !== value.toLowerCase())
  const next = [{ text: value, at: Date.now() }, ...rest].slice(0, LIMIT)
  storageSetJSON(KEY, next)
  return next
}

export function removeHistory(text) {
  const next = loadHistory().filter((r) => r.text !== text)
  storageSetJSON(KEY, next)
  return next
}

export function clearHistory() {
  storageSetJSON(KEY, [])
  return []
}

/** Время строки истории: «19:17» сегодня, «26.07 19:17» — раньше. */
export function historyTime(at) {
  const date = new Date(at || 0)
  if (Number.isNaN(date.getTime())) return ''
  const time = date.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  const today = new Date()
  const sameDay = date.toDateString() === today.toDateString()
  if (sameDay) return time
  return `${date.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' })} ${time}`
}
