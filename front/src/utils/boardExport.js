/* Выгрузка доски в растр и PDF.

   Рисует сам клиент: холст уже умеет рендерить сцену, поэтому серверу не нужен
   растеризатор. Перед отрисовкой картинки сцены ЗАГРУЖАЮТСЯ явно (иначе в
   файл уходил бы пустой прямоугольник вместо изображения), а PDF собирается
   вручную — одностраничный документ с одним JPEG-объектом внутри, без внешних
   зависимостей. */
import { normalizeScene, renderScene, sceneBounds, orderedObjects } from '@/utils/boardScene.js'

// Поля вокруг рисунка и предел стороны растра (дальше файл распухает зря).
const PAD = 32
const MAX_SIDE = 4096

/** Дождаться загрузки всех картинок сцены: Map<src, HTMLImageElement>. */
export async function loadSceneImages(scene) {
  const sources = [...new Set(orderedObjects(scene, { onlyVisible: false })
    .filter((o) => o.type === 'image' && o.src)
    .map((o) => o.src))]

  const entries = await Promise.all(sources.map((src) => new Promise((resolve) => {
    const img = new Image()
    img.crossOrigin = 'anonymous'
    img.onload = () => resolve([src, img])
    // Не загрузилась — отдаём пустую запись: доска выгрузится без этой картинки.
    img.onerror = () => resolve(null)
    img.src = src
  })))

  return new Map(entries.filter(Boolean))
}

/** Отрисовать сцену в offscreen-canvas по её содержимому. */
async function renderToCanvas(scene, { scale = 2, background = 'light' } = {}) {
  const s = normalizeScene(scene)
  const objects = orderedObjects(s)
  if (!objects.length) return null

  const box = sceneBounds(objects)
  const width = Math.min(MAX_SIDE, Math.round((box.w + PAD * 2) * scale))
  const height = Math.min(MAX_SIDE, Math.round((box.h + PAD * 2) * scale))

  const canvas = document.createElement('canvas')
  canvas.width = Math.max(1, width)
  canvas.height = Math.max(1, height)
  const ctx = canvas.getContext('2d')
  if (!ctx) return null

  // JPEG и PDF не знают прозрачности — подкладываем белый лист.
  if (background) {
    ctx.fillStyle = '#ffffff'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
  }

  const images = await loadSceneImages(s)
  renderScene(ctx, s, {
    width: canvas.width,
    height: canvas.height,
    camera: { x: box.x - PAD, y: box.y - PAD, scale },
    images,
    background: false, // фон-сетку в файл не тащим: она мешает читать рисунок
  })
  return canvas
}

/** PNG (с белым фоном) — Blob или null, если рисовать нечего. */
export async function sceneToPng(scene, opts = {}) {
  const canvas = await renderToCanvas(scene, opts)
  if (!canvas) return null
  return new Promise((resolve) => canvas.toBlob(resolve, 'image/png'))
}

/** JPG — Blob или null. */
export async function sceneToJpeg(scene, opts = {}) {
  const canvas = await renderToCanvas(scene, opts)
  if (!canvas) return null
  return new Promise((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.92))
}

/** PDF — одна страница по размеру рисунка; Blob или null. */
export async function sceneToPdf(scene, opts = {}) {
  const canvas = await renderToCanvas(scene, opts)
  if (!canvas) return null
  const jpeg = await new Promise((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.92))
  if (!jpeg) return null
  return buildPdf(new Uint8Array(await jpeg.arrayBuffer()), canvas.width, canvas.height)
}

/* buildPdf — минимальный PDF 1.4: каталог, страница и картинка-XObject на всю
   страницу. Своё, потому что подключать генератор PDF ради одной страницы с
   растром — лишняя зависимость в бандле. */
function buildPdf(jpegBytes, pxWidth, pxHeight) {
  // 96 dpi: пиксели растра переводим в пункты PDF (1 pt = 1/72 дюйма).
  const ptWidth = Math.round((pxWidth * 72) / 96)
  const ptHeight = Math.round((pxHeight * 72) / 96)

  const encoder = new TextEncoder()
  const chunks = []
  const offsets = []
  let length = 0

  const push = (bytes) => {
    chunks.push(bytes)
    length += bytes.length
  }
  const pushText = (text) => push(encoder.encode(text))
  const startObject = (num) => {
    offsets[num] = length
    pushText(`${num} 0 obj\n`)
  }

  pushText('%PDF-1.4\n')

  startObject(1)
  pushText('<< /Type /Catalog /Pages 2 0 R >>\nendobj\n')

  startObject(2)
  pushText('<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n')

  startObject(3)
  pushText(`<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ${ptWidth} ${ptHeight}] `
    + '/Resources << /XObject << /Im0 4 0 R >> >> /Contents 5 0 R >>\nendobj\n')

  startObject(4)
  pushText(`<< /Type /XObject /Subtype /Image /Width ${pxWidth} /Height ${pxHeight} `
    + `/ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length ${jpegBytes.length} >>\nstream\n`)
  push(jpegBytes)
  pushText('\nendstream\nendobj\n')

  const content = `q\n${ptWidth} 0 0 ${ptHeight} 0 0 cm\n/Im0 Do\nQ\n`
  startObject(5)
  pushText(`<< /Length ${content.length} >>\nstream\n${content}endstream\nendobj\n`)

  const xrefAt = length
  const count = 6
  let xref = `xref\n0 ${count}\n0000000000 65535 f \n`
  for (let i = 1; i < count; i++) {
    xref += `${String(offsets[i]).padStart(10, '0')} 00000 n \n`
  }
  pushText(xref)
  pushText(`trailer\n<< /Size ${count} /Root 1 0 R >>\nstartxref\n${xrefAt}\n%%EOF\n`)

  return new Blob(chunks, { type: 'application/pdf' })
}

/** Скачать Blob под именем (общая для всех форматов выгрузки доски). */
export function saveBlob(blob, name) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  a.click()
  URL.revokeObjectURL(url)
}
