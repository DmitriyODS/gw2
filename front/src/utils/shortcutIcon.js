/**
 * Значок ярлыка раздела для домашнего экрана телефона.
 *
 * Рисуем его сами: у разделов иконки — глифы шрифта Material Symbols, и
 * держать двадцать копий картинок в ресурсах обёртки ради ярлыков незачёт.
 * Итог — base64-PNG, который нативный слой отдаёт лаунчеру
 * (`NativeShell.pinShortcut`).
 *
 * Цвета берём из ТЕКУЩЕЙ темы: человек ставит ярлык, глядя на своё оформление,
 * и ждёт такой же значок. Это снимок — ярлык живёт на домашнем экране и после
 * смены темы не перерисовывается; то же исключение из правила токенов, что и у
 * выгрузок досок: файл уезжает наружу, где переменных CSS уже нет.
 */

// Адаптивный значок: лаунчер накладывает свою маску (круг, скруглённый
// квадрат), поэтому содержимое держим в центре с запасом по краям.
const SIZE = 192
const GLYPH_RATIO = 0.44

// Фолбэк, если тема не резолвится (старый WebView без oklch) — цвета марки.
const FALLBACK_BG = '#1195ED'
const FALLBACK_FG = '#ffffff'

const ICON_FONT = '"Material Symbols Outlined"'

/**
 * @param {object} app — раздел: { icon: глиф Material Symbols, title: название }
 * @returns {Promise<string>} base64 PNG без префикса data:
 */
export async function renderShortcutIcon({ icon, title }) {
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = SIZE
  const ctx = canvas.getContext('2d')
  if (!ctx) return ''

  const bg = resolveColor(ctx, themeColor('--color-primary'), FALLBACK_BG)
  const fg = resolveColor(ctx, themeColor('--color-on-primary'), FALLBACK_FG)

  ctx.fillStyle = bg
  ctx.fillRect(0, 0, SIZE, SIZE)

  const px = Math.round(SIZE * GLYPH_RATIO)
  const glyph = (await iconFontReady(px)) ? icon : initial(title)

  ctx.fillStyle = fg
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.font = glyph === icon
    ? `${px}px ${ICON_FONT}`
    : `700 ${px}px "Roboto Flex", system-ui, sans-serif`
  ctx.fillText(glyph, SIZE / 2, SIZE / 2)

  return canvas.toDataURL('image/png').split(',')[1] || ''
}

function themeColor(token) {
  return getComputedStyle(document.documentElement).getPropertyValue(token).trim()
}

/* Токены темы описаны в oklch(): старый WebView его не понимает и молча
   оставляет прежний fillStyle — проверяем присвоение и падаем на цвет марки. */
function resolveColor(ctx, color, fallback) {
  if (!color) return fallback
  ctx.fillStyle = '#000000'
  ctx.fillStyle = color
  return ctx.fillStyle === '#000000' ? fallback : ctx.fillStyle
}

/* Глиф рисуется лигатурой имени иконки — без загруженного шрифта на значке
   оказалось бы само слово «dashboard_customize». Тогда берём первую букву
   названия раздела. */
async function iconFontReady(px) {
  const font = `${px}px ${ICON_FONT}`
  try {
    await document.fonts?.load(font)
    return document.fonts?.check(font) === true
  } catch {
    return false
  }
}

function initial(title) {
  return (title || '?').trim().charAt(0).toUpperCase()
}
