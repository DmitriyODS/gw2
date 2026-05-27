import { ref } from 'vue'
import { getStaleTasks } from '@/api/tasks.js'

const STORAGE_KEY = 'gw2_stale_reminder_shown_date'

// Module-level singleton: модалка одна на приложение, автопоказ управляется
// из App.vue после входа. Показываем не чаще раза в календарный день.
const isOpen = ref(false)
const tasks = ref([])

function todayKey() {
  const d = new Date()
  return `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`
}

function close() {
  isOpen.value = false
}

// Раз в день проверяет давние задачи и показывает напоминание, если они есть.
async function check() {
  let shown = null
  try { shown = localStorage.getItem(STORAGE_KEY) } catch {}
  if (shown === todayKey()) return

  try {
    const data = await getStaleTasks()
    const items = data?.items || []
    // Отмечаем сегодняшний показ независимо от результата — чтобы не дёргать
    // эндпоинт повторно при каждом возврате на вкладку в течение дня.
    try { localStorage.setItem(STORAGE_KEY, todayKey()) } catch {}
    if (items.length) {
      tasks.value = items
      isOpen.value = true
    }
  } catch {
    // Напоминание некритично — молча пропускаем при ошибке.
  }
}

export function useStaleReminder() {
  return { isOpen, tasks, check, close }
}
