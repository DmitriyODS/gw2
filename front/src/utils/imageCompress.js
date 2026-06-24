// Клиентское сжатие картинок перед загрузкой: даунскейл длинной стороны + JPEG
// с адаптивным качеством, чтобы снимки с телефонов не засоряли хранилище.
// Возвращает новый File (jpeg) либо исходный, если сжатие не требуется/невозможно.

const MB = 1024 * 1024

export async function compressImage(file, { maxBytes = 4 * MB, maxDim = 1920, mime = 'image/jpeg' } = {}) {
  if (!file || !file.type?.startsWith('image/')) return file
  // Векторы и анимации canvas испортит — не трогаем.
  if (file.type === 'image/svg+xml' || file.type === 'image/gif') return file

  let bitmap
  try {
    bitmap = await createImageBitmap(file)
  } catch {
    return file
  }

  const { width, height } = bitmap
  const scale = Math.min(1, maxDim / Math.max(width, height))

  // Уже в пределах размеров и лимита по весу — отдаём оригинал без перекодирования.
  if (scale === 1 && file.size <= maxBytes) {
    bitmap.close?.()
    return file
  }

  const w = Math.max(1, Math.round(width * scale))
  const h = Math.max(1, Math.round(height * scale))
  const canvas = document.createElement('canvas')
  canvas.width = w
  canvas.height = h
  canvas.getContext('2d').drawImage(bitmap, 0, 0, w, h)
  bitmap.close?.()

  let quality = 0.9
  let blob = await toBlob(canvas, mime, quality)
  while (blob && blob.size > maxBytes && quality > 0.4) {
    quality -= 0.1
    blob = await toBlob(canvas, mime, quality)
  }
  if (!blob) return file

  const name = file.name.replace(/\.[^.]+$/, '') + '.jpg'
  return new File([blob], name, { type: mime, lastModified: file.lastModified })
}

function toBlob(canvas, mime, quality) {
  return new Promise((resolve) => canvas.toBlob(resolve, mime, quality))
}
