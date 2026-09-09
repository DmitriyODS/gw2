/* Заливки и эффекты доски: градиенты и коррекции (размытие, тень, яркость,
   контраст, насыщенность, оттенок, обесцвечивание, сепия).

   Отдельный модуль, потому что одно и то же нужно и холсту (CanvasGradient +
   ctx.filter), и выгрузке (defs с linearGradient/filter в SVG строит сервер по
   ТЕМ ЖЕ полям). Значения коррекций — доли: 1 — «как есть», как в CSS-функциях
   фильтров, чтобы не переводить их туда-сюда. */

export const GRADIENT_TYPES = [
  { key: 'linear', label: 'Линейный' },
  { key: 'radial', label: 'Радиальный' },
]

// Значения по умолчанию: то же, что «эффекта нет».
export const EFFECT_DEFAULTS = {
  blur: 0,
  brightness: 1,
  contrast: 1,
  saturate: 1,
  hue: 0,
  grayscale: 0,
  sepia: 0,
  shadow: null, // { x, y, blur, color }
}

// Ползунки панели свойств: подпись, границы и шаг — один источник для UI.
export const EFFECT_CONTROLS = [
  { key: 'blur', label: 'Размытие', min: 0, max: 40, step: 1, unit: 'px' },
  { key: 'brightness', label: 'Яркость', min: 0, max: 2, step: 0.05 },
  { key: 'contrast', label: 'Контраст', min: 0, max: 2, step: 0.05 },
  { key: 'saturate', label: 'Насыщенность', min: 0, max: 3, step: 0.05 },
  { key: 'hue', label: 'Оттенок', min: -180, max: 180, step: 5, unit: '°' },
  { key: 'grayscale', label: 'Обесцветить', min: 0, max: 1, step: 0.05 },
  { key: 'sepia', label: 'Сепия', min: 0, max: 1, step: 0.05 },
]

const clamp = (value, min, max, fallback) => {
  const n = Number(value)
  if (!Number.isFinite(n)) return fallback
  return Math.min(max, Math.max(min, n))
}

/** Нормализация градиента; null — заливка обычным цветом. */
export function normalizeGradient(raw) {
  if (!raw || !Array.isArray(raw.stops) || raw.stops.length < 2) return null
  const stops = raw.stops
    .filter((s) => s && typeof s.color === 'string')
    .map((s) => ({ color: s.color, at: clamp(s.at, 0, 1, 0) }))
    .sort((a, b) => a.at - b.at)
  if (stops.length < 2) return null
  return {
    type: raw.type === 'radial' ? 'radial' : 'linear',
    angle: clamp(raw.angle, 0, 360, 90),
    stops,
  }
}

/** Нормализация эффектов; null — эффектов нет (в сцене поле не заводится). */
export function normalizeEffects(raw) {
  if (!raw || typeof raw !== 'object') return null
  const out = {
    blur: clamp(raw.blur, 0, 200, 0),
    brightness: clamp(raw.brightness, 0, 4, 1),
    contrast: clamp(raw.contrast, 0, 4, 1),
    saturate: clamp(raw.saturate, 0, 4, 1),
    hue: clamp(raw.hue, -180, 180, 0),
    grayscale: clamp(raw.grayscale, 0, 1, 0),
    sepia: clamp(raw.sepia, 0, 1, 0),
    shadow: null,
  }
  if (raw.shadow && typeof raw.shadow === 'object') {
    out.shadow = {
      x: clamp(raw.shadow.x, -200, 200, 0),
      y: clamp(raw.shadow.y, -200, 200, 4),
      blur: clamp(raw.shadow.blur, 0, 200, 8),
      color: typeof raw.shadow.color === 'string' && raw.shadow.color ? raw.shadow.color : 'ink',
      opacity: clamp(raw.shadow.opacity, 0, 1, 0.35),
    }
  }
  return hasEffects(out) ? out : null
}

/** Есть ли что применять — пустые эффекты в сцене не храним. */
export function hasEffects(e) {
  if (!e) return false
  return !!e.shadow || e.blur > 0 || e.grayscale > 0 || e.sepia > 0 || e.hue !== 0
    || e.brightness !== 1 || e.contrast !== 1 || e.saturate !== 1
}

/** Строка для ctx.filter и CSS — тот же порядок, что в SVG-фильтре сервера. */
export function cssFilter(e) {
  if (!e) return ''
  const parts = []
  if (e.blur > 0) parts.push(`blur(${e.blur}px)`)
  if (e.brightness !== 1) parts.push(`brightness(${e.brightness})`)
  if (e.contrast !== 1) parts.push(`contrast(${e.contrast})`)
  if (e.saturate !== 1) parts.push(`saturate(${e.saturate})`)
  if (e.hue !== 0) parts.push(`hue-rotate(${e.hue}deg)`)
  if (e.grayscale > 0) parts.push(`grayscale(${e.grayscale})`)
  if (e.sepia > 0) parts.push(`sepia(${e.sepia})`)
  return parts.join(' ')
}

/* applyEffects — коррекции и тень на контекст. Тень задаётся отдельно от
   filter: drop-shadow() в фильтре размывал бы и её вместе со всем остальным,
   а нам нужна тень ОТ объекта, каким он получился. */
export function applyEffects(ctx, effects, resolve) {
  if (!effects) return
  const filter = cssFilter(effects)
  if (filter && 'filter' in ctx) ctx.filter = filter
  if (effects.shadow) {
    const { x, y, blur, color, opacity } = effects.shadow
    ctx.shadowColor = withAlpha(resolve(color), opacity)
    ctx.shadowBlur = blur
    ctx.shadowOffsetX = x
    ctx.shadowOffsetY = y
  }
}

/** Цвет с прозрачностью для тени: «#rrggbb» + доля → rgba(). */
export function withAlpha(color, alpha = 1) {
  const value = String(color || '#000')
  if (alpha >= 1) return value
  if (/^#([0-9a-f]{6})$/i.test(value)) {
    const r = parseInt(value.slice(1, 3), 16)
    const g = parseInt(value.slice(3, 5), 16)
    const b = parseInt(value.slice(5, 7), 16)
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }
  // Токены темы приходят в OKLCH: у CSS-цвета прозрачность задаётся через
  // color-mix, и canvas это понимает.
  return `color-mix(in srgb, ${value} ${Math.round(alpha * 100)}%, transparent)`
}

/** CanvasGradient по рамке объекта; resolve разворачивает ключ палитры. */
export function makeGradient(ctx, gradient, box, resolve) {
  const g = normalizeGradient(gradient)
  if (!g || !box) return null
  let paint
  if (g.type === 'radial') {
    const cx = box.x + box.w / 2
    const cy = box.y + box.h / 2
    const r = Math.max(box.w, box.h) / 2 || 1
    paint = ctx.createRadialGradient(cx, cy, 0, cx, cy, r)
  } else {
    // Угол как в CSS: 0° — снизу вверх, 90° — слева направо.
    const rad = ((g.angle - 90) * Math.PI) / 180
    const cx = box.x + box.w / 2
    const cy = box.y + box.h / 2
    const half = (Math.abs(box.w * Math.cos(rad)) + Math.abs(box.h * Math.sin(rad))) / 2 || 1
    paint = ctx.createLinearGradient(
      cx - Math.cos(rad) * half, cy - Math.sin(rad) * half,
      cx + Math.cos(rad) * half, cy + Math.sin(rad) * half,
    )
  }
  for (const stop of g.stops) paint.addColorStop(stop.at, resolve(stop.color))
  return paint
}

/** Градиент по умолчанию — от текущего цвета к прозрачному белому. */
export function defaultGradient(color = 'blue') {
  return { type: 'linear', angle: 90, stops: [{ color, at: 0 }, { color: 'white', at: 1 }] }
}
