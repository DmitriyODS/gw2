import { ref } from 'vue'
import { getMorningBriefing } from '@/api/groove.js'
import { storageGet, storageSet } from '@/utils/storage.js'

const STORAGE_KEY = 'gw2_morning_briefing_shown_date'

// Module-level singleton: модалка одна на приложение, автопоказ управляется
// из App.vue после входа. Показываем не чаще раза в календарный день.
const isOpen = ref(false)
const briefing = ref(null)

function todayKey() {
  const d = new Date()
  return `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`
}

// Время суток по локальным часам клиента — для приветствия Грувика.
function partOfDay() {
  const h = new Date().getHours()
  if (h >= 5 && h < 12) return 'morning'
  if (h >= 12 && h < 17) return 'day'
  if (h >= 17 && h < 23) return 'evening'
  return 'night'
}

function close() {
  isOpen.value = false
}

// Раз в день запрашивает утренний брифинг и показывает модалку, если Грувику
// есть что сказать (data.show). Иначе тихо помечает день показанным.
async function check() {
  const shown = storageGet(STORAGE_KEY, null)
  if (shown === todayKey()) return

  try {
    const data = await getMorningBriefing(partOfDay())
    // Отмечаем сегодняшний показ независимо от результата — чтобы не дёргать
    // эндпоинт повторно при каждом возврате на вкладку в течение дня.
    storageSet(STORAGE_KEY, todayKey())
    if (data?.show) {
      briefing.value = data
      isOpen.value = true
    }
  } catch {
    // Напоминание некритично — молча пропускаем при ошибке.
  }
}

export function useMorningBriefing() {
  return { isOpen, briefing, check, close }
}
