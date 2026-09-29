import { computed, ref } from 'vue'
import { muteNotifications, notifyMutedUntil, unmuteNotifications } from '@/utils/systemNotify.js'

/* Реактивная обёртка над «не беспокоить» (само состояние — в localStorage,
   см. utils/systemNotify.js): нужна кнопке уведомлений, чтобы перечёркнутый
   колокольчик и пункт меню менялись сразу, и настройкам, чтобы тумблер не
   расходился с меню.

   Тишина на срок заканчивается сама: один таймер ровно до её конца гасит
   состояние без перезагрузки страницы. */

// Потолок setTimeout (~24,8 суток): дальше таймер сработал бы сразу.
const MAX_DELAY = 2 ** 31 - 1

const until = ref(notifyMutedUntil())
let timer = null

function sync() {
  until.value = notifyMutedUntil()
  clearTimeout(timer)
  timer = null
  // Ждать есть смысл только у тишины с концом: «навсегда» само не пройдёт.
  if (until.value !== null && until.value !== Infinity) {
    timer = setTimeout(sync, Math.min(Math.max(until.value - Date.now(), 0) + 50, MAX_DELAY))
  }
}

sync()

export function useNotifyMute() {
  const muted = computed(() => until.value !== null)
  const forever = computed(() => until.value === Infinity)

  /** «до 14:30» либо «навсегда» — подпись для меню и подсказки кнопки. */
  const untilLabel = computed(() => {
    if (!muted.value) return ''
    if (forever.value) return 'навсегда'
    return `до ${new Date(until.value).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })}`
  })

  function mute(minutes = null) {
    muteNotifications(minutes)
    sync()
  }

  function unmute() {
    unmuteNotifications()
    sync()
  }

  return { muted, forever, untilLabel, mute, unmute }
}
