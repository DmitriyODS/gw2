/**
 * Журнал «Моя активность» — лента в меню «Пуск».
 *
 * Один поток событий: и созданное пользователем (задача, заметка, запись
 * ежедневника/календаря/реестра, публикация на портале), и открытые разделы —
 * всё ложится в общий список со временем. Каждая запись знает свой раздел,
 * название и путь, поэтому из ленты можно вернуться прямо к элементу.
 * Хранится локально (localStorage) — это личная история сеанса работы, а не
 * серверный аудит; на выходе из системы чистится.
 */
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { storageGetJSON, storageSetJSON } from '@/utils/storage.js'

const KEY = 'gw_activity'
const MAX = 40

// Что произошло: подпись строки в ленте.
export const ACTIONS = {
  created: 'Создано',
  published: 'Опубликовано',
  closed: 'Закрыто',
  opened: 'Открыт раздел',
}

/** Ключ раздела — один на раздел: повторный заход поднимает прежнюю строку. */
const sectionKey = (appId) => `open:${appId}`

function normalizeItems(raw) {
  if (!Array.isArray(raw)) return []
  return raw
    // Путь необязателен: у строки раздела его подставляет реестр приложений.
    .filter((e) => e && e.section && e.at)
    .map((e) => ({
      key: String(e.key || `${e.section}:${e.path || ''}`),
      section: String(e.section),
      action: ACTIONS[e.action] ? e.action : 'created',
      title: String(e.title || ''),
      path: String(e.path || ''),
      at: String(e.at),
    }))
}

// Прежде разделы жили отдельным списком — поднимаем их в общий поток.
function normalizeSections(raw) {
  if (!Array.isArray(raw)) return []
  return raw
    .filter((s) => s && s.id)
    .map((s) => ({
      key: sectionKey(s.id),
      section: String(s.id),
      action: 'opened',
      title: '',
      path: '',
      at: String(s.at || new Date().toISOString()),
    }))
}

// Читаем и оба прежних формата: голый массив действий и пару {items, sections}.
function load() {
  const raw = storageGetJSON(KEY, null)
  if (Array.isArray(raw)) return normalizeItems(raw).slice(0, MAX)
  return [...normalizeItems(raw?.items), ...normalizeSections(raw?.sections)]
    .sort((a, b) => (a.at < b.at ? 1 : -1))
    .slice(0, MAX)
}

export const useActivityStore = defineStore('activity', () => {
  const items = ref(load())

  const recent = computed(() => items.value)

  function persist() {
    storageSetJSON(KEY, { items: items.value })
  }

  /** Положить событие наверх, вытеснив прежнее с тем же ключом. */
  function push(entry) {
    items.value = [entry, ...items.value.filter((e) => e.key !== entry.key)].slice(0, MAX)
    persist()
  }

  /**
   * Записать действие над элементом. Повтор по тому же элементу не плодит
   * строки — прежняя поднимается наверх с новым временем (пересохранение
   * заметки не должно забивать ленту).
   */
  function record({ section, action = 'created', title, path, id }) {
    if (!section || !path) return
    push({
      key: `${section}:${id ?? path}`,
      section,
      action: ACTIONS[action] ? action : 'created',
      title: title || '',
      path,
      at: new Date().toISOString(),
    })
  }

  /**
   * Раздел открыт окном. Путь не храним — его знает реестр приложений
   * (`desktop/apps.js`), а второму списку разделов на фронте взяться неоткуда.
   */
  function recordSection(appId) {
    if (!appId) return
    push({ key: sectionKey(appId), section: String(appId), action: 'opened', title: '', path: '', at: new Date().toISOString() })
  }

  function clear() {
    items.value = []
    persist()
  }

  const reset = clear

  return { items, recent, record, recordSection, clear, reset }
})
