/* Раскладка дня на шкале: окна между занятиями, накладки и сводка.

   Чистые функции — из них живёт и колонка недели, и день на телефоне, и
   печать. Занятость и окна сервер считает теми же правилами
   (back-go/schedule/internal/domain/layout.go), поэтому цифра на живой плитке
   совпадает со сводкой на экране. */

/** Склеить пересекающиеся отрезки: накладка не удваивает занятость. */
export function mergeSpans(items) {
  const sorted = items
    .map((it) => ({ s: it.start_min, e: it.end_min }))
    .sort((a, b) => a.s - b.s || a.e - b.e)
  const out = []
  sorted.forEach((span) => {
    const last = out[out.length - 1]
    if (last && span.s <= last.e) last.e = Math.max(last.e, span.e)
    else out.push({ ...span })
  })
  return out
}

/** Свободные окна между занятиями длиной не меньше minGap. */
export function gapsOf(items, minGap = 20) {
  const spans = mergeSpans(items)
  const out = []
  for (let i = 1; i < spans.length; i++) {
    const gap = { s: spans[i - 1].e, e: spans[i].s }
    if (gap.e - gap.s >= minGap) out.push(gap)
  }
  return out
}

/** Сколько минут занято (без двойного счёта накладок). */
export function busyOf(items) {
  return mergeSpans(items).reduce((sum, s) => sum + (s.e - s.s), 0)
}

/** Сколько всего времени приходится на окна. */
export function gapMinutes(items, minGap = 20) {
  return gapsOf(items, minGap).reduce((sum, g) => sum + (g.e - g.s), 0)
}

/* Группы пересекающихся занятий: накладку рисуем ОДНОЙ карточкой в несколько
   строк — так видно оба занятия и их время, и ничего не обрезается (приём
   прототипа: две колонки по половине ширины читаются хуже). */
export function clusters(items) {
  const sorted = items.slice().sort((a, b) => a.start_min - b.start_min || a.end_min - b.end_min)
  const out = []
  let i = 0
  while (i < sorted.length) {
    let j = i
    let end = sorted[i].end_min
    while (j + 1 < sorted.length && sorted[j + 1].start_min < end) {
      j++
      end = Math.max(end, sorted[j].end_min)
    }
    out.push(sorted.slice(i, j + 1))
    i = j + 1
  }
  return out
}

/* Границы шкалы считаются ПО ЗАНЯТИЯМ и округляются до часа: настраивать их
   руками не нужно, а пустое расписание показывает рабочий день. Запас в
   полчаса сверху и снизу оставляет место подписи первого и последнего блока. */
export function scaleBounds(items, fallback = { from: 8 * 60, to: 20 * 60 }) {
  if (!items.length) return { ...fallback }
  const first = Math.min(...items.map((it) => it.start_min))
  const last = Math.max(...items.map((it) => it.end_min))
  const from = Math.max(0, Math.floor((first - 30) / 60) * 60)
  const to = Math.min(24 * 60, Math.ceil((last + 30) / 60) * 60)
  return to - from < 120 ? { from, to: Math.min(24 * 60, from + 120) } : { from, to }
}

/** Доля отрезка на шкале в процентах — позиция блока и его высота. */
export function scalePercent(minute, bounds) {
  const span = Math.max(1, bounds.to - bounds.from)
  return ((minute - bounds.from) / span) * 100
}

/* Сколько дней недели показывать: до последнего дня, где вообще есть занятия,
   но не меньше пяти — рабочая неделя выглядит рабочей неделей даже у пустого
   расписания. Сегодняшний день показываем всегда, даже если он выходной. */
export function visibleDays(items, todayWeekday = -1) {
  let last = 4
  items.forEach((it) => { if (it.weekday > last) last = it.weekday })
  if (todayWeekday > last) last = todayWeekday
  return Array.from({ length: last + 1 }, (_, i) => i)
}
