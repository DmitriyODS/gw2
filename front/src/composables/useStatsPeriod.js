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

export function useStatsPeriod() {
  const currentMonday = getMonday(new Date())
  const currentSunday = new Date(currentMonday)
  currentSunday.setDate(currentMonday.getDate() + 6)

  const periodFrom = ref(currentMonday)
  const periodTo = ref(currentSunday)

  let mode = 'week'

  const fromStr = computed(() => formatDate(periodFrom.value))
  const toStr = computed(() => formatDate(periodTo.value))

  function setDay(dir = 1) {
    if (mode === 'day') {
      const d = new Date(periodFrom.value)
      d.setDate(d.getDate() + dir)
      periodFrom.value = d
      periodTo.value = new Date(d)
    } else {
      mode = 'day'
      const today = new Date()
      today.setHours(0, 0, 0, 0)
      if (dir < 0) today.setDate(today.getDate() - 1)
      periodFrom.value = today
      periodTo.value = new Date(today)
    }
  }

  function setWeek(dir = 1) {
    if (mode === 'week') {
      const d = new Date(periodFrom.value)
      d.setDate(d.getDate() + dir * 7)
      periodFrom.value = d
      const e = new Date(d)
      e.setDate(e.getDate() + 6)
      periodTo.value = e
    } else {
      mode = 'week'
      const monday = getMonday(new Date())
      if (dir < 0) monday.setDate(monday.getDate() - 7)
      periodFrom.value = monday
      const e = new Date(monday)
      e.setDate(e.getDate() + 6)
      periodTo.value = e
    }
  }

  function setMonth(dir = 1) {
    if (mode === 'month') {
      const d = new Date(periodFrom.value)
      d.setMonth(d.getMonth() + dir)
      periodFrom.value = new Date(d.getFullYear(), d.getMonth(), 1)
      periodTo.value = new Date(d.getFullYear(), d.getMonth() + 1, 0)
    } else {
      mode = 'month'
      const d = new Date()
      d.setMonth(d.getMonth() + (dir < 0 ? -1 : 0))
      periodFrom.value = new Date(d.getFullYear(), d.getMonth(), 1)
      periodTo.value = new Date(d.getFullYear(), d.getMonth() + 1, 0)
    }
  }

  function setCustom(from, to) {
    mode = 'custom'
    periodFrom.value = new Date(from)
    periodTo.value = new Date(to)
  }

  const displayLabel = computed(() => {
    const f = periodFrom.value
    const t = periodTo.value
    const fmt = (d) => d.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
    return `${fmt(f)} — ${fmt(t)}`
  })

  return { periodFrom, periodTo, fromStr, toStr, displayLabel, setDay, setWeek, setMonth, setCustom }
}
