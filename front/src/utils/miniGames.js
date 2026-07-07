// Чистая логика мини-игр действий питомца (кормление/лечение/поглаживание) —
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

// Поглаживание («трение ладошкой»): копим пройденную указателем дистанцию;
// каждые thresholdPx — один завершённый «цикл» глажки (= один платный
// StrokePet). add() возвращает прогресс текущего цикла 0..100 и признак
// завершения; после завершения дистанция цикла начинается заново.
export function createRubTracker(thresholdPx = 480) {
  let dist = 0

  return {
    add(dx, dy) {
      dist += Math.abs(dx) + Math.abs(dy)
      if (dist >= thresholdPx) {
        dist -= thresholdPx
        return { progress: Math.min(100, (dist / thresholdPx) * 100), completed: true }
      }
      return { progress: (dist / thresholdPx) * 100, completed: false }
    },
    reset() { dist = 0 },
    get progress() { return Math.min(100, (dist / thresholdPx) * 100) },
  }
}
