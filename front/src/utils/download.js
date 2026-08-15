import { saveNativeFile } from '@/utils/nativeApp.js'

/** Скачать Blob под именем — ЕДИНСТВЕННЫЙ способ отдать файл пользователю.
 *
 * Своей ссылки-«скачать» разделам заводить нельзя: в мобильной обёртке WebView
 * молча игнорирует blob:-ссылки (DownloadListener видит только http), и файл
 * никуда не сохранялся. Здесь такой файл уходит нативке, а в браузере и в
 * десктоп-обёртке остаётся обычная ссылка. */
export async function saveBlob(blob, name) {
  // Мост либо забирает файл, либо сообщает об отказе (нет доступа к памяти) —
  // ошибку показывает вызвавший раздел. Ссылка остаётся браузеру и старым
  // обёрткам, которые saveFile ещё не знают.
  if (await saveNativeFile(blob, name)) return

  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  // Ссылку кладём в документ: без этого клик игнорируют старые WebView обёрток.
  document.body.appendChild(a)
  a.click()
  a.remove()
  // Адрес освобождаем следующим тиком: отзыв в том же кадре срывал скачивание.
  setTimeout(() => URL.revokeObjectURL(url), 0)
}

/** Скачать файл по адресу (готовый объект хранилища, а не собранный в вебе). */
export function saveUrl(url, name) {
  const a = document.createElement('a')
  a.href = url
  a.download = name || ''
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  a.remove()
}

/** Имя файла из пользовательского текста: без разделителей пути и лишних пробелов. */
export function safeFileName(name, fallback = 'file') {
  const clean = String(name || '').replace(/[\\/:*?"<>|]/g, ' ').replace(/\s+/g, ' ').trim()
  return clean || fallback
}
