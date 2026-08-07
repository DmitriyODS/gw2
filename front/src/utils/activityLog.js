import { useActivityStore } from '@/stores/activity.js'

/**
 * Записать действие в журнал ленты «Последние действия».
 *
 * Обёртка намеренно глотает ошибки: лента — вспомогательная, и вне
 * Pinia-контекста (например, в юнит-тестах стора) она не должна ронять само
 * действие пользователя.
 */
export function logActivity(entry) {
  try {
    useActivityStore().record(entry)
  } catch {
    /* журнал не критичен */
  }
}

/** Отметить открытый раздел (строка «Недавние разделы» в ленте). */
export function logSection(appId) {
  try {
    useActivityStore().recordSection(appId)
  } catch {
    /* журнал не критичен */
  }
}
