import { onBeforeUnmount, onMounted, ref } from 'vue'

/* Общие минутные часы каркаса (панель задач, панель «Виджетов»): один таймер
   на всех, срабатывает ровно на границе минуты — часы «ЧЧ:ММ» не отстают, а
   панели перерисовываются раз в минуту, а не каждые 10 с. На скрытой вкладке
   стоят; возврат сразу подводит время. */

const now = ref(new Date())
let users = 0
let timer = null

function schedule() {
  clearTimeout(timer)
  timer = null
  if (!users || document.hidden) return
  now.value = new Date()
  const toNextMinute = 60_000 - (Date.now() % 60_000)
  timer = setTimeout(schedule, toNextMinute + 50)
}

/** @returns {import('vue').Ref<Date>} */
export function useMinuteClock() {
  onMounted(() => {
    if (users++ === 0) document.addEventListener('visibilitychange', schedule)
    schedule()
  })
  onBeforeUnmount(() => {
    if (--users === 0) {
      document.removeEventListener('visibilitychange', schedule)
      schedule()
    }
  })
  return now
}
