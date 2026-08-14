/** Скачать Blob под именем — единственный способ отдать файл пользователю. */
export function saveBlob(blob, name) {
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

/** Имя файла из пользовательского текста: без разделителей пути и лишних пробелов. */
export function safeFileName(name, fallback = 'file') {
  const clean = String(name || '').replace(/[\\/:*?"<>|]/g, ' ').replace(/\s+/g, ' ').trim()
  return clean || fallback
}
