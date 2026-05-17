import { ref, computed } from 'vue'

function formatDate(d) {
  return d.toISOString().split('T')[0]
}

export function useStatsPeriod() {
  const now = new Date()
  const periodFrom = ref(new Date(now.getFullYear(), 0, 1))
  const periodTo = ref(new Date(now.getFullYear(), 11, 31))

  let mode = 'year'

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
      const d = dir > 0 ? new Date(periodFrom.value) : new Date(periodFrom.value)
      d.setDate(d.getDate() + (dir < 0 ? -1 : 0))
      periodFrom.value = d
      periodTo.value = new Date(d)
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
      const d = new Date(periodFrom.value)
      if (dir < 0) d.setDate(d.getDate() - 7)
      periodFrom.value = d
      const e = new Date(d)
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
