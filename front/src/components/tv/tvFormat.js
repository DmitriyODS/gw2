// Общие утилиты форматирования ТВ-режима: числа, часы, тона, проценты баров.

export function num(v) {
  const n = Number(v)
  return Number.isFinite(n) ? n : 0
}

export function sumHours(list) {
  if (!list) return 0
  return list.reduce((acc, e) => acc + num(e.total_hours), 0)
}

export function barPercent(val, max) {
  const m = num(max)
  if (!m) return 0
  return Math.max(6, Math.round((num(val) / m) * 100))
}

// Семантический тон → CSS-токен цвета.
export function toneColor(tone) {
  switch (tone) {
    case 'primary':   return 'var(--color-primary)'
    case 'secondary': return 'var(--color-secondary)'
    case 'tertiary':  return 'var(--color-tertiary)'
    case 'success':   return 'var(--color-success)'
    case 'warning':   return 'var(--color-warning)'
    case 'error':     return 'var(--color-error)'
    default: return 'var(--color-primary)'
  }
}

// При больших объёмах команды «440 ч» выглядит абстрактно — переводим в
// рабочие дни (длина дня настраивается в настройках табло) с порога в
// 5 рабочих дней: становится «55 дн» или «55 дн 4 ч», что нагляднее на табло.
export function formatHoursShort(val, hoursPerDay = 8) {
  const hours = num(val)
  if (hours <= 0) return '0 ч'

  const perDay = num(hoursPerDay) || 8
  if (hours >= perDay * 5) {
    const days = Math.floor(hours / perDay)
    const remainHours = Math.round(hours - days * perDay)
    if (remainHours === 0) return `${days} дн`
    return `${days} дн ${remainHours} ч`
  }

  const totalMinutes = Math.round(hours * 60)
  const h = Math.floor(totalMinutes / 60)
  const m = totalMinutes % 60
  if (h === 0) return `${m} мин`
  if (m === 0) return `${h} ч`
  return `${h} ч ${m} мин`
}

export function plural(n, one, few, many) {
  const a = Math.abs(n) % 100
  const b = a % 10
  if (a > 10 && a < 20) return many
  if (b > 1 && b < 5) return few
  if (b === 1) return one
  return many
}
