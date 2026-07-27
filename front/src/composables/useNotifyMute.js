import { computed, ref } from 'vue'
import { muteNotifications, notifyMutedUntil, unmuteNotifications } from '@/utils/systemNotify.js'

/* Реактивная обёртка над «не беспокоить» (само состояние — в localStorage,
   см. utils/systemNotify.js): нужна кнопке уведомлений, чтобы перечёркнутый
   колокольчик и пункт меню менялись сразу, и настройкам, чтобы тумблер не
   расходился с меню.

   Тишина на срок заканчивается сама: пока она идёт, тикает секундный таймер —
   по его истечении состояние гаснет без перезагрузки страницы. */

const until = ref(notifyMutedUntil())
let timer = null

function sync() {
  until.value = notifyMutedUntil()
  clearInterval(timer)
  timer = null
  // Тикать есть смысл только у тишины с концом: «навсегда» само не пройдёт.
  if (until.value !== null && until.value !== Infinity) {
    timer = setInterval(() => {
      until.value = notifyMutedUntil()
      if (until.value === null) { clearInterval(timer); timer = null }
    }, 1000)
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
