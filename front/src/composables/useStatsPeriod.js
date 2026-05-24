import { ref, computed } from 'vue'

function formatDate(d) {
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function getMonday(d) {
  const day = d.getDay()
  const diff = day === 0 ? -6 : 1 - day
  const monday = new Date(d)
  monday.setDate(d.getDate() + diff)
  monday.setHours(0, 0, 0, 0)
  return monday
}

// Самая ранняя дата для режима «весь срок» — заведомо раньше любых данных.
const ALL_TIME_START = new Date(2000, 0, 1)

export function useStatsPeriod() {
  const mode = ref('week')
  const periodFrom = ref(getMonday(new Date()))
  const periodTo = ref((() => {
    const s = new Date(periodFrom.value)
    s.setDate(s.getDate() + 6)
    return s
  })())

  const fromStr = computed(() => formatDate(periodFrom.value))
  const toStr = computed(() => formatDate(periodTo.value))

  // Устанавливает диапазон по текущей дате для выбранного типа периода.
  function selectMode(m) {
    mode.value = m
    const now = new Date()
    if (m === 'day') {
      const d = new Date(now)
      d.setHours(0, 0, 0, 0)
      periodFrom.value = d
      periodTo.value = new Date(d)
    } else if (m === 'week') {
      const monday = getMonday(now)
      periodFrom.value = monday
      const e = new Date(monday)
      e.setDate(e.getDate() + 6)
      periodTo.value = e
    } else if (m === 'month') {
      periodFrom.value = new Date(now.getFullYear(), now.getMonth(), 1)
      periodTo.value = new Date(now.getFullYear(), now.getMonth() + 1, 0)
    } else if (m === 'year') {
      periodFrom.value = new Date(now.getFullYear(), 0, 1)
      periodTo.value = new Date(now.getFullYear(), 11, 31)
    }
  }

  // Сдвигает текущий диапазон на одну единицу выбранного типа периода.
  function shift(dir = 1) {
    if (mode.value === 'day') {
      const d = new Date(periodFrom.value)
      d.setDate(d.getDate() + dir)
      periodFrom.value = d
      periodTo.value = new Date(d)
    } else if (mode.value === 'week') {
      const d = new Date(periodFrom.value)
      d.setDate(d.getDate() + dir * 7)
      periodFrom.value = d
      const e = new Date(d)
      e.setDate(e.getDate() + 6)
      periodTo.value = e
    } else if (mode.value === 'month') {
      const d = new Date(periodFrom.value)
      d.setMonth(d.getMonth() + dir)
      periodFrom.value = new Date(d.getFullYear(), d.getMonth(), 1)
      periodTo.value = new Date(d.getFullYear(), d.getMonth() + 1, 0)
    } else if (mode.value === 'year') {
      const y = periodFrom.value.getFullYear() + dir
      periodFrom.value = new Date(y, 0, 1)
      periodTo.value = new Date(y, 11, 31)
    }
  }

  // «Весь срок» — диапазон, заведомо охватывающий все задачи в системе.
  function setAllTime() {
    mode.value = 'all'
    periodFrom.value = new Date(ALL_TIME_START)
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    periodTo.value = today
  }

  function setCustom(from, to) {
    mode.value = 'custom'
    periodFrom.value = new Date(from)
    periodTo.value = new Date(to)
  }

  const displayLabel = computed(() => {
    if (mode.value === 'all') return 'За весь срок'
    const fmt = (d) => d.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
    return `${fmt(periodFrom.value)} — ${fmt(periodTo.value)}`
  })

  return {
    mode, periodFrom, periodTo, fromStr, toStr, displayLabel,
    selectMode, shift, setAllTime, setCustom,
  }
}
