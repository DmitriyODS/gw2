// Чистая логика мини-игр действий питомца (кормление/лечение) —
// вынесена из компонентов, чтобы попадания/промахи проверялись без DOM.

// Кормление/лечение (drag-and-drop): совпадение с круглой hit-zone.
export function distance(x1, y1, x2, y2) {
  return Math.hypot(x1 - x2, y1 - y2)
}

export function isInHitZone(x, y, zoneX, zoneY, radius) {
  return distance(x, y, zoneX, zoneY) <= radius
}

// Лечение (серия точных тапов): маркер бежит 0..100% по полосе, «зелёная
// зона» — отрезок [start, start+width]. Успешные попадания подряд считает
// вызывающая сторона (streak сбрасывается при промахе — см. HealMiniGame).
export function isInGreenZone(markerPercent, zoneStart, zoneWidth) {
  return markerPercent >= zoneStart && markerPercent <= zoneStart + zoneWidth
}

// Позиция маркера в момент времени t (мс) при периоде period (мс), качается
// туда-обратно 0..100 (треугольная волна) — детерминировано и тестируемо.
export function pingPongPercent(t, period) {
  const phase = (t % period) / period // 0..1
  const triangle = phase < 0.5 ? phase * 2 : 2 - phase * 2 // 0..1..0
  return triangle * 100
}
