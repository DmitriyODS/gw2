import { ref } from 'vue'
import { storageGet, storageRemove, storageSet } from '@/utils/storage.js'

/* Скрытый раздел «DevTools»: инструменты для проверки самого приложения
   (сейчас — показ тестовых уведомлений). В списке настроек его нет, пока не
   позовут — пять быстрых нажатий по номеру сборки в «О приложении», приём из
   мобильных ОС. Прячется он же кнопкой внутри раздела.

   Состояние модульное: флаг читают и каталог настроек, и сам раздел. */

const KEY = 'gw_devtools'
const TAPS = 5
const WINDOW_MS = 1500

export const devToolsOn = ref(storageGet(KEY) === '1')

export function showDevTools() {
  devToolsOn.value = true
  storageSet(KEY, '1')
}

export function hideDevTools() {
  devToolsOn.value = false
  storageRemove(KEY)
}

let taps = 0
let firstTapAt = 0

/** Нажатие по номеру сборки. Возвращает true, когда раздел только что открылся. */
export function tapBuildNumber() {
  const now = Date.now()
  // Пауза дольше окна начинает счёт заново: случайные клики по номеру за день
  // не должны складываться в «пять быстрых».
  if (now - firstTapAt > WINDOW_MS) {
    taps = 0
    firstTapAt = now
  }
  taps += 1
  if (taps < TAPS) return false
  taps = 0
  const wasOff = !devToolsOn.value
  showDevTools()
  return wasOff
}
