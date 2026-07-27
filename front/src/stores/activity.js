/**
 * Журнал последних действий пользователя — лента в меню «Пуск».
 *
 * Пишется там, где действие происходит (создание задачи, заметки, записи
 * ежедневника/календаря/реестра, публикация на портале): каждая запись знает
 * свой раздел, название и путь, поэтому из ленты можно перейти прямо к
 * элементу. Хранится локально (localStorage) — это личная история сеанса
 * работы, а не серверный аудит; на выходе из системы чистится.
 */
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { storageGetJSON, storageSetJSON } from '@/utils/storage.js'

const KEY = 'gw_activity'
const MAX = 24
// Сколько недавно открытых разделов помним (строка чипов над лентой).
const MAX_SECTIONS = 6

// Что произошло: подпись строки в ленте.
export const ACTIONS = {
  created: 'Создано',
  published: 'Опубликовано',
  closed: 'Закрыто',
}

function normalizeItems(raw) {
  if (!Array.isArray(raw)) return []
  return raw
    .filter((e) => e && e.section && e.path && e.at)
    .map((e) => ({
      key: String(e.key || `${e.section}:${e.path}`),
      section: String(e.section),
      action: ACTIONS[e.action] ? e.action : 'created',
      title: String(e.title || ''),
      path: String(e.path),
      at: String(e.at),
    }))
    .slice(0, MAX)
}

function normalizeSections(raw) {
  if (!Array.isArray(raw)) return []
  return raw
    .filter((s) => s && s.id)
    .map((s) => ({ id: String(s.id), at: String(s.at || new Date().toISOString()) }))
    .slice(0, MAX_SECTIONS)
}

// Раньше в хранилище лежал голый массив действий — читаем и такой формат.
function load() {
  const raw = storageGetJSON(KEY, null)
  if (Array.isArray(raw)) return { items: normalizeItems(raw), sections: [] }
  return { items: normalizeItems(raw?.items), sections: normalizeSections(raw?.sections) }
}

export const useActivityStore = defineStore('activity', () => {
  const stored = load()
  const items = ref(stored.items)
  // Недавно открытые разделы — отдельный список: он про навигацию, а не про
  // созданные элементы, и не должен вытеснять действия из ленты.
  const sections = ref(stored.sections)

  const recent = computed(() => items.value)

  function persist() {
    storageSetJSON(KEY, { items: items.value, sections: sections.value })
  }

  /**
   * Записать действие. Повтор по тому же элементу не плодит строки — прежняя
   * поднимается наверх с новым временем (пересохранение заметки не должно
   * забивать ленту).
   */
  function record({ section, action = 'created', title, path, id }) {
    if (!section || !path) return
    const key = `${section}:${id ?? path}`
    const entry = { key, section, action, title: title || '', path, at: new Date().toISOString() }
    items.value = [entry, ...items.value.filter((e) => e.key !== key)].slice(0, MAX)
    persist()
  }

  /** Раздел открыт окном — поднимаем его в начало списка недавних. */
  function recordSection(appId) {
    if (!appId) return
    sections.value = [{ id: String(appId), at: new Date().toISOString() },
      ...sections.value.filter((s) => s.id !== appId)].slice(0, MAX_SECTIONS)
    persist()
  }

  function clear() {
    items.value = []
    sections.value = []
    persist()
  }

  const reset = clear

  return { items, sections, recent, record, recordSection, clear, reset }
})
