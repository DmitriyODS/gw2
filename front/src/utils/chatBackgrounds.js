// Оформление чатов мессенджера: пресеты градиента (из токенов темы, как фон
// приложения) + бесшовные SVG-узоры-трафареты. Рецепт хранится на бэкенде как
// непрозрачный JSON; форму владеет фронт (эти утилиты). Цвета — ТОЛЬКО токены:
// градиент собирается через color-mix от var(--color-*), узор красится токеном
// через mask-image, поэтому оформление само следует теме и тёмному режиму.

/* ── Градиент ─────────────────────────────────────────────────────
   Пятно: role (токен), позиция x/y в %, радиус spread в % бокса, доля alpha.
   Пресеты — именованные композиции; 'custom' — сгенерированная случайно. */
export const GRADIENT_ROLES = ['primary', 'secondary', 'tertiary']

export const GRADIENT_PRESETS = [
  {
    key: 'plain', label: 'Без градиента',
    blobs: [],
  },
  {
    key: 'aurora', label: 'Сияние',
    blobs: [
      { role: 'primary', x: 12, y: 8, spread: 60, alpha: 22 },
      { role: 'tertiary', x: 92, y: 24, spread: 55, alpha: 18 },
      { role: 'secondary', x: 30, y: 104, spread: 62, alpha: 16 },
    ],
  },
  {
    key: 'sunset', label: 'Закат',
    blobs: [
      { role: 'tertiary', x: 4, y: 96, spread: 66, alpha: 24 },
      { role: 'primary', x: 96, y: 88, spread: 58, alpha: 18 },
      { role: 'secondary', x: 60, y: 4, spread: 50, alpha: 14 },
    ],
  },
  {
    key: 'ocean', label: 'Океан',
    blobs: [
      { role: 'secondary', x: 8, y: 10, spread: 64, alpha: 22 },
      { role: 'primary', x: 88, y: 60, spread: 60, alpha: 18 },
      { role: 'tertiary', x: 40, y: 108, spread: 54, alpha: 14 },
    ],
  },
  {
    key: 'bloom', label: 'Цветение',
    blobs: [
      { role: 'primary', x: 50, y: -6, spread: 56, alpha: 20 },
      { role: 'tertiary', x: 6, y: 70, spread: 52, alpha: 18 },
      { role: 'secondary', x: 100, y: 78, spread: 52, alpha: 18 },
    ],
  },
  {
    key: 'mono', label: 'Спокойный',
    blobs: [
      { role: 'primary', x: 20, y: 4, spread: 70, alpha: 12 },
      { role: 'primary', x: 88, y: 100, spread: 60, alpha: 10 },
    ],
  },
]

const rand = (min, max) => min + Math.random() * (max - min)
const pickOne = (arr) => arr[Math.floor(Math.random() * arr.length)]

/* Случайная композиция пятен — якорятся к краям, центр остаётся спокойным. */
export function randomGradientBlobs() {
  const zones = [
    { x: [-8, 16], y: [-10, 10] }, { x: [84, 108], y: [-8, 14] },
    { x: [-10, 12], y: [40, 66] }, { x: [88, 110], y: [40, 66] },
    { x: [-8, 14], y: [86, 110] }, { x: [86, 110], y: [86, 110] },
    { x: [40, 62], y: [-10, 6] }, { x: [38, 62], y: [96, 112] },
  ]
  const shuffled = [...zones].sort(() => Math.random() - 0.5).slice(0, 3)
  const roles = [...GRADIENT_ROLES].sort(() => Math.random() - 0.5)
  return shuffled.map((z, i) => ({
    role: roles[i % roles.length],
    x: Math.round(rand(z.x[0], z.x[1])),
    y: Math.round(rand(z.y[0], z.y[1])),
    spread: Math.round(rand(50, 68)),
    alpha: Math.round(rand(i === 0 ? 18 : 12, i === 0 ? 26 : 20)),
  }))
}

/* CSS background-image из набора пятен. Доля затемняется в тёмной теме
   множителем --chat-grad-dim (задан в компоненте-слое). */
export function gradientCss(blobs) {
  if (!Array.isArray(blobs) || !blobs.length) return 'none'
  return blobs.map((b) =>
    `radial-gradient(circle at ${b.x}% ${b.y}%, ` +
    `color-mix(in oklch, var(--color-${b.role}) calc(${b.alpha}% * var(--chat-grad-dim, 1)), transparent) 0%, ` +
    `transparent ${b.spread}%)`,
  ).join(', ')
}

/* ── Узоры-трафареты ──────────────────────────────────────────────
   Плитка 64×64 с двумя экземплярами глифа по диагонали (16,16)/(48,48) —
   глиф не касается краёв, поэтому repeat бесшовный. Заливка сплошная #000:
   SVG используется как mask-image, значение имеет только альфа-канал, а цвет
   даёт background-color (токен). */
const tile = (inner) =>
  `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="64" height="64">${inner}</svg>`

const at = (x, y, glyph) => `<g transform="translate(${x},${y})">${glyph}</g>`
const pair = (glyph) => tile(at(16, 16, glyph) + at(48, 48, glyph))

const G = {
  dots: '<circle r="4" fill="#000"/>',
  bubbles: '<circle r="9" fill="none" stroke="#000" stroke-width="2.5"/>',
  rhombus: '<rect x="-7" y="-7" width="14" height="14" rx="2" fill="#000" transform="rotate(45)"/>',
  sparkle: '<path d="M0 -11 C1.5 -3 3 -1.5 11 0 C3 1.5 1.5 3 0 11 C-1.5 3 -3 1.5 -11 0 C-3 -1.5 -1.5 -3 0 -11 Z" fill="#000"/>',
  plus: '<path d="M-2.5 -9 h5 v6.5 h6.5 v5 h-6.5 v6.5 h-5 v-6.5 h-6.5 v-5 h6.5 z" fill="#000"/>',
  hex: '<path d="M0 -10 L8.66 -5 L8.66 5 L0 10 L-8.66 5 L-8.66 -5 Z" fill="none" stroke="#000" stroke-width="2.5"/>',
  triangle: '<path d="M0 -9 L9 7 L-9 7 Z" fill="none" stroke="#000" stroke-width="2.5" stroke-linejoin="round"/>',
  heart: '<path d="M0 8 C-9 1 -9 -6 -4 -8 C-1 -9 0 -6 0 -5 C0 -6 1 -9 4 -8 C9 -6 9 1 0 8 Z" fill="#000"/>',
}

export const PATTERNS = [
  { key: null, label: 'Нет' },
  { key: 'dots', label: 'Точки' },
  { key: 'bubbles', label: 'Пузыри' },
  { key: 'rhombus', label: 'Ромбы' },
  { key: 'sparkle', label: 'Искры' },
  { key: 'plus', label: 'Плюсы' },
  { key: 'hex', label: 'Соты' },
  { key: 'triangle', label: 'Треугольники' },
  { key: 'heart', label: 'Сердечки' },
]

const PATTERN_SVG = Object.fromEntries(
  Object.entries(G).map(([k, glyph]) => [k, pair(glyph)]),
)

/* data:URI маски узора (кодируем только необходимое — компактнее base64). */
export function patternDataUri(key) {
  const svg = PATTERN_SVG[key]
  if (!svg) return ''
  return `url("data:image/svg+xml,${encodeURIComponent(svg)}")`
}

/* Плитка из эмодзи — цветной глиф, поэтому это background-image (НЕ mask):
   SVG <text> рендерится системным эмодзи-шрифтом, цвет сохраняется. */
const emojiTile = (emoji) => {
  const t = (x, y) =>
    `<text x="${x}" y="${y}" font-size="26" text-anchor="middle" ` +
    `dominant-baseline="central" ` +
    `font-family="Apple Color Emoji,Segoe UI Emoji,Noto Color Emoji,sans-serif">${emoji}</text>`
  return tile(t(16, 16) + t(48, 48))
}

export function emojiPatternDataUri(emoji) {
  if (!emoji) return ''
  return `url("data:image/svg+xml,${encodeURIComponent(emojiTile(emoji))}")`
}

/* Цвет узора — токен основного текста (существует во всех темах). */
export const PATTERN_ROLE = 'text'

/* ── Рецепт ───────────────────────────────────────────────────────
   Целостный объект оформления: пресет/пятна градиента + узор с
   насыщённостью и масштабом. Нормализация — на чтении с бэкенда. */
export const DEFAULT_RECIPE = {
  gradient: { preset: 'aurora', blobs: null },
  pattern: { key: 'dots', emoji: null, alpha: 6, size: 128 },
  // Своя картинка-фон (url из общего uploads-тома) со степенью размытия.
  // null — картинки нет, работает градиент.
  image: null,
}

// Диапазон размытия картинки-фона, px.
export const IMAGE_BLUR_MAX = 40

// Глубокая копия рецепта — рабочая копия редактора не должна делить ссылки
// с сохранённым состоянием стора.
export function cloneRecipe(r) {
  return {
    gradient: {
      preset: r.gradient.preset,
      blobs: r.gradient.blobs ? r.gradient.blobs.map((b) => ({ ...b })) : null,
    },
    pattern: { ...r.pattern },
    image: r.image ? { ...r.image } : null,
  }
}

const clamp = (v, min, max, dflt) =>
  Number.isFinite(v) ? Math.min(Math.max(v, min), max) : dflt

export function normalizeRecipe(raw) {
  if (!raw || typeof raw !== 'object') return null
  const g = raw.gradient || {}
  const preset = GRADIENT_PRESETS.some((p) => p.key === g.preset) ? g.preset
    : (Array.isArray(g.blobs) ? 'custom' : 'aurora')
  const blobs = Array.isArray(g.blobs)
    ? g.blobs.map((b) => ({
      role: GRADIENT_ROLES.includes(b?.role) ? b.role : 'primary',
      x: clamp(b?.x, -30, 130, 50), y: clamp(b?.y, -30, 130, 50),
      spread: clamp(b?.spread, 30, 80, 60), alpha: clamp(b?.alpha, 4, 40, 18),
    })).slice(0, 4)
    : null
  const p = raw.pattern || {}
  const pattern = {
    key: PATTERNS.some((x) => x.key === p.key) ? p.key : null,
    // Эмодзи-узор (взаимоисключим с key). Ограничиваем длину — один глиф.
    emoji: (typeof p.emoji === 'string' && p.emoji) ? [...p.emoji].slice(0, 4).join('') : null,
    alpha: clamp(p.alpha, 0, 30, 6),
    size: clamp(p.size, 64, 240, 128),
  }
  const img = raw.image
  const image = (img && typeof img.url === 'string' && img.url)
    ? { url: img.url, blur: clamp(img.blur, 0, IMAGE_BLUR_MAX, 12) }
    : null
  return { gradient: { preset, blobs }, pattern, image }
}

/* Пустой рецепт — нечего показывать: нет картинки, градиент «Без градиента»
   (пресет plain / нет пятен) и нет узора. Тогда обложку не рисуем вовсе —
   виден обычный фон (градиент приложения у портала). */
export function isBlankRecipe(recipe) {
  if (!recipe) return true
  const g = recipe.gradient || {}
  const hasGradient = (g.preset && g.preset !== 'plain') ||
    (Array.isArray(g.blobs) && g.blobs.length > 0)
  const p = recipe.pattern || {}
  const hasPattern = (p.key || p.emoji) && p.alpha > 0
  const hasImage = !!(recipe.image && recipe.image.url)
  return !hasGradient && !hasPattern && !hasImage
}

/* Пятна для рендера: у именованного пресета — из каталога, у 'custom' — свои. */
export function recipeBlobs(recipe) {
  const g = recipe?.gradient
  if (!g) return []
  if (g.preset === 'custom' && Array.isArray(g.blobs)) return g.blobs
  const preset = GRADIENT_PRESETS.find((p) => p.key === g.preset)
  return preset ? preset.blobs : []
}

/* Инлайн-стили для слоя фона: {gradient} — фон градиента, {pattern} — узор
   (или null). Используют и превью-диалог, и боевой слой. */
export function chatBgStyles(recipe) {
  const blobs = recipeBlobs(recipe)
  const gradient = { backgroundImage: gradientCss(blobs) }

  // Картинка-фон (поверх градиента, под узором). Лёгкий upscale прячет
  // прозрачные края, которые даёт размытие.
  let image = null
  const im = recipe?.image
  if (im && im.url) {
    const blur = im.blur || 0
    image = {
      backgroundImage: `url("${im.url}")`,
      backgroundSize: 'cover',
      backgroundPosition: 'center',
      filter: blur ? `blur(${blur}px)` : 'none',
      transform: `scale(${(1 + blur / 90).toFixed(3)})`,
    }
  }

  const p = recipe?.pattern
  let pattern = null
  if (p && p.alpha > 0 && (p.emoji || p.key)) {
    if (p.emoji) {
      // Цветной эмодзи — обычный background-image, без mask.
      pattern = {
        backgroundImage: emojiPatternDataUri(p.emoji),
        backgroundRepeat: 'repeat',
        backgroundSize: `${p.size}px ${p.size}px`,
        opacity: p.alpha / 100,
      }
    } else {
      const uri = patternDataUri(p.key)
      pattern = {
        backgroundColor: `var(--color-${PATTERN_ROLE})`,
        maskImage: uri, WebkitMaskImage: uri,
        maskRepeat: 'repeat', WebkitMaskRepeat: 'repeat',
        maskSize: `${p.size}px ${p.size}px`, WebkitMaskSize: `${p.size}px ${p.size}px`,
        opacity: p.alpha / 100,
      }
    }
  }
  return { gradient, image, pattern }
}
