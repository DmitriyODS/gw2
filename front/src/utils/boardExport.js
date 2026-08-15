/* Выгрузка доски в растр и PDF плюс миниатюра для плитки списка.

   Рисует сам клиент: холст уже умеет рендерить сцену, поэтому серверу не нужен
   растеризатор. Перед отрисовкой картинки сцены ЗАГРУЖАЮТСЯ явно (иначе в
   файл уходил бы пустой прямоугольник вместо изображения), а PDF собирает
   общий `utils/pdf.js` — одна страница по размеру рисунка.

   Файл и миниатюра рисуются по-разному, и это намеренно. ФАЙЛ уезжает наружу,
   где темы приложения нет: белый лист и фиксированная палитра (та же, что в
   SVG сервера). МИНИАТЮРА — снимок холста: фон и цвета берутся из темы, иначе
   доска, нарисованная в тёмной теме, показывалась в списке белым листом. */
import {
  EXPORT_COLORS, normalizeScene, renderScene, sceneBounds, orderedObjects,
} from '@/utils/boardScene.js'
import { canvasToJpegBytes, jpegPagesToPdf } from '@/utils/pdf.js'

export { saveBlob } from '@/utils/download.js'

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

/** Отрисовать сцену в offscreen-canvas по её содержимому.
    paper=true — белый лист и палитра файлов; иначе фон холста и тема. */
async function renderToCanvas(scene, { scale = 2, paper = true } = {}) {
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
  if (paper) {
    ctx.fillStyle = '#ffffff'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
  }

  const images = await loadSceneImages(s)
  renderScene(ctx, s, {
    width: canvas.width,
    height: canvas.height,
    camera: { x: box.x - PAD, y: box.y - PAD, scale },
    images,
    // Фон-сетку в файл не тащим: она мешает читать рисунок. В миниатюре фон
    // холста, наоборот, нужен — плитка обязана выглядеть как сама доска.
    background: !paper,
    colors: paper ? EXPORT_COLORS : undefined,
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
  const bytes = await canvasToJpegBytes(canvas, 0.92)
  if (!bytes) return null
  return jpegPagesToPdf([{ bytes, width: canvas.width, height: canvas.height }])
}

/** Миниатюра для плитки списка — снимок холста в текущей теме. */
export async function sceneToPreview(scene, { scale = 0.6 } = {}) {
  const canvas = await renderToCanvas(scene, { scale, paper: false })
  if (!canvas) return null
  return new Promise((resolve) => canvas.toBlob(resolve, 'image/png'))
}
