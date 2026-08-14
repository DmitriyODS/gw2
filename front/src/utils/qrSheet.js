/* Лист QR-кодов: ОДНА раскладка на два выхода — печать (HTML во временном
   iframe) и файл PDF. Числа сетки живут здесь, поэтому лист и предпросмотр не
   разъезжаются: печатный документ и сохранённый файл обязаны выглядеть
   одинаково — их сверяют, приложив распечатку к экрану. */
import { canvasToJpegBytes, jpegPagesToPdf } from '@/utils/pdf.js'

// A4 книжной ориентации, всё в миллиметрах.
export const SHEET = {
  cols: 4,
  rows: 6,
  pageW: 210,
  pageH: 297,
  margin: 10,
  code: 30,
  gap: 2,
  // Подпись под кодом: кегль в мм (8pt) и предел в две строки.
  capSize: 2.8,
  capLines: 2,
}

export const PER_PAGE = SHEET.cols * SHEET.rows

// Потолок одного задания: дальше браузер захлёбывается на data-URI кодов.
export const MAX_CODES = 1000

/**
 * Позиции с количеством копий → плоский список значений.
 * Одной вещи хватает ярлыка, партии нужен ярлык на штуку, поэтому позиция
 * повторяется столько раз, сколько для неё заказали.
 * @param {Array<{value: string, count: number}>} items
 * @returns {string[]}
 */
export function expandCodes(items, max = MAX_CODES) {
  const out = []
  for (const it of items || []) {
    for (let i = 0; i < (it.count || 0) && out.length < max; i++) out.push(it.value)
  }
  return out
}

export function pageCount(total) {
  return Math.max(1, Math.ceil(total / PER_PAGE))
}

/** Разложить ячейки по страницам (по PER_PAGE на лист). */
export function chunkPages(cells) {
  const pages = []
  for (let i = 0; i < cells.length; i += PER_PAGE) pages.push(cells.slice(i, i + PER_PAGE))
  return pages
}

/* ── PDF ───────────────────────────────────────────────────────────
   Лист рисуется в canvas и уходит в PDF картинкой: так подпись печатается
   системным шрифтом с кириллицей, а встраивать шрифт в документ не нужно. */

const MM_PER_INCH = 25.4

// Разрешение листа: при 150 dpi код 30 мм — это 177 px, модули остаются
// крупными и читаются сканером, а файл не распухает.
const DPI = 150

/**
 * Собрать PDF из готовых кодов.
 * @param {Array<{value: string, src: string}>} cells — код и его подпись.
 * @returns {Promise<Blob|null>}
 */
export async function codesToPdf(cells) {
  if (!cells?.length) return null
  const images = await loadImages(cells)
  const pages = []
  for (const page of chunkPages(cells)) {
    const bytes = await canvasToJpegBytes(renderPage(page, images))
    if (bytes) {
      pages.push({
        bytes,
        width: px(SHEET.pageW),
        height: px(SHEET.pageH),
        // Лист остаётся ровно A4 независимо от разрешения растра.
        ptWidth: Math.round((SHEET.pageW * 72) / MM_PER_INCH),
        ptHeight: Math.round((SHEET.pageH * 72) / MM_PER_INCH),
      })
    }
  }
  return pages.length ? jpegPagesToPdf(pages) : null
}

const px = (mm) => Math.round((mm * DPI) / MM_PER_INCH)

async function loadImages(cells) {
  const sources = [...new Set(cells.map((c) => c.src))]
  const pairs = await Promise.all(sources.map((src) => new Promise((resolve) => {
    const img = new Image()
    img.onload = () => resolve([src, img])
    // Не нарисовавшийся код не повод ронять весь лист — ячейка останется пустой.
    img.onerror = () => resolve(null)
    img.src = src
  })))
  return new Map(pairs.filter(Boolean))
}

function renderPage(cells, images) {
  const canvas = document.createElement('canvas')
  canvas.width = px(SHEET.pageW)
  canvas.height = px(SHEET.pageH)
  const ctx = canvas.getContext('2d')

  // Код обязан быть чёрным по белому: тема приложения к печатному листу
  // неприменима, сканеру нужен контраст.
  ctx.fillStyle = '#ffffff'
  ctx.fillRect(0, 0, canvas.width, canvas.height)
  // Сглаживание размывает модули кода — выключаем.
  ctx.imageSmoothingEnabled = false

  const cellW = (SHEET.pageW - SHEET.margin * 2) / SHEET.cols
  const cellH = (SHEET.pageH - SHEET.margin * 2) / SHEET.rows
  const capSize = px(SHEET.capSize)
  const lineH = Math.round(capSize * 1.15)

  ctx.font = `${capSize}px Arial, Helvetica, sans-serif`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'top'

  cells.forEach((cell, i) => {
    const col = i % SHEET.cols
    const row = Math.floor(i / SHEET.cols)
    const cx = px(SHEET.margin + cellW * col + cellW / 2)
    // Код и подпись — единым блоком по центру ячейки, как в печатном листе.
    const capH = lineH * SHEET.capLines
    const blockH = px(SHEET.code) + px(SHEET.gap) + capH
    const top = px(SHEET.margin + cellH * row) + Math.round((px(cellH) - blockH) / 2)

    const img = images.get(cell.src)
    if (img) ctx.drawImage(img, cx - px(SHEET.code) / 2, top, px(SHEET.code), px(SHEET.code))

    ctx.fillStyle = '#000000'
    drawCaption(ctx, cell.value, cx, top + px(SHEET.code) + px(SHEET.gap), px(cellW) - px(4), lineH)
  })

  return canvas
}

/* Подпись бьётся ПО СИМВОЛАМ (как `word-break: break-all` в печатном листе):
   инвентарный номер — одно длинное «слово», по пробелам его не перенести. */
function drawCaption(ctx, text, cx, y, maxWidth, lineH) {
  const value = String(text ?? '')
  const lines = []
  let line = ''
  for (const ch of value) {
    if (line && ctx.measureText(line + ch).width > maxWidth) {
      lines.push(line)
      line = ch
      if (lines.length === SHEET.capLines) break
    } else {
      line += ch
    }
  }
  if (lines.length < SHEET.capLines && line) lines.push(line)

  const drawn = lines.slice(0, SHEET.capLines)
  // Хвост не поместился — обрываем последнюю строку многоточием.
  const rest = value.length - drawn.join('').length
  if (rest > 0 && drawn.length) {
    let last = drawn[drawn.length - 1]
    while (last && ctx.measureText(`${last}…`).width > maxWidth) last = last.slice(0, -1)
    drawn[drawn.length - 1] = `${last}…`
  }

  drawn.forEach((l, i) => ctx.fillText(l, cx, y + i * lineH))
}
