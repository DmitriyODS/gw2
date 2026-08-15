import { reactive, watch } from 'vue'
import { storageGetJSON, storageSetJSON } from '@/utils/storage.js'

/* Личные настройки всплывающих уведомлений: откуда выезжают, сколько висят,
   видны ли поверх экрана блокировки и от каких разделов вообще приходят.
 *
 * Хранение — localStorage, как у звука и «не беспокоить» рядом: это настройка
 * УСТРОЙСТВА. На рабочем компьютере угол и срок жизни выбирают одни, на
 * домашнем ноутбуке — другие, и синхронизировать их через аккаунт значило бы
 * навязывать одному месту привычки другого.
 *
 * Состояние модульное (одно на приложение): его читают и панель настроек, и
 * сама стопка тостов, а смонтированы они порознь. */

const KEY = 'gw_notify_prefs'

/** Углы, откуда выезжает стопка на рабочем столе. На телефоне всегда сверху:
    снизу карточку закрыла бы панель задач, а места по бокам там нет. */
export const TOAST_CORNERS = [
  { key: 'top-right', label: 'Сверху справа' },
  { key: 'top-left', label: 'Сверху слева' },
  { key: 'bottom-right', label: 'Снизу справа' },
  { key: 'bottom-left', label: 'Снизу слева' },
]

/** Сколько карточка висит без внимания (под курсором отсчёт замирает). */
export const TOAST_LIVES = [
  { key: 3000, label: '3 с' },
  { key: 5000, label: '5 с' },
  { key: 10000, label: '10 с' },
]

/* Разделы, события которых порождают уведомления. Звонков здесь нет
   намеренно: пропущенный звонок — это не «шум», глушить его настройкой
   нельзя (для тишины есть «не беспокоить», и та звонки не трогает). */
export const NOTIFY_SOURCES = [
  { key: 'messenger', label: 'Сообщения', hint: 'Новые сообщения и упоминания в чатах' },
  { key: 'tasks', label: 'Задачи', hint: 'Назначенные задачи и комментарии' },
  { key: 'portal', label: 'Портал', hint: 'Новые публикации и обсуждения' },
  { key: 'reminders', label: 'Напоминания', hint: 'Сработавшие напоминания' },
  { key: 'pets', label: 'Питомцы', hint: 'Кудосы от коллег, болезнь и побег грувика' },
  { key: 'billing', label: 'Счета и покупки', hint: 'Счета, платежи и заказы' },
]

const DEFAULTS = {
  corner: 'top-right',
  life: 5000,
  onLockScreen: false,
  // Раздела нет в наборе → уведомления принимаем: новый источник не должен
  // молчать из-за того, что настройки записаны прежней версией.
  sources: {},
}

export const notifyPrefs = reactive({ ...DEFAULTS, ...(storageGetJSON(KEY, {}) || {}) })

watch(notifyPrefs, (v) => storageSetJSON(KEY, { ...v }), { deep: true })

/** Принимаем ли уведомления этого раздела (без источника — всегда да: это
    ответ на действие самого человека, а не событие раздела). */
export function isSourceEnabled(source) {
  if (!source) return true
  return notifyPrefs.sources[source] !== false
}

export function setSourceEnabled(source, on) {
  notifyPrefs.sources = { ...notifyPrefs.sources, [source]: !!on }
}
