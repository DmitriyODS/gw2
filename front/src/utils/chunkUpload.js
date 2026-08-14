/* Загрузка файлов — общий механизм платформы.

   Файл крупнее CHUNK_THRESHOLD уезжает ЧАСТЯМИ в любом разделе: одним запросом
   он упирается в таймауты прокси, не даёт показать честный прогресс и после
   обрыва сети начинается заново. Мелкое отправляется как раньше — одной формой,
   лишний круг запросов ему ни к чему.

   Серверная сторона — pkg/chunkupload: три ручки на раздел (init/chunk/finish)
   с одинаковым контрактом, поэтому здесь один загрузчик на всех.

   Порог и размер части держать в паре с pkg/chunkupload (Threshold/ChunkSize).*/
import { apiRequest } from '@/api/client.js'

// С какого размера файл обязан ехать частями.
export const CHUNK_THRESHOLD = 10 * 1024 * 1024
// Размер части по умолчанию; сервер вправе назвать свой при заведении загрузки.
export const CHUNK_SIZE = 5 * 1024 * 1024
// Часть загружается дольше обычного запроса — свой потолок ожидания.
const CHUNK_TIMEOUT = 120000
// Сколько раз повторяем часть, прежде чем сдаться: моргнувшая сеть не должна
// стоить человеку всей загрузки.
const CHUNK_RETRIES = 3

/**
 * Загрузить файл: сам выберет одиночный или чанковый путь.
 *
 * @param {object} opts
 * @param {File|Blob} opts.file — что грузим.
 * @param {string} opts.directUrl — ручка одиночной загрузки (multipart «file»).
 * @param {string} opts.chunkBase — префикс чанковых ручек раздела
 *        (`<base>/init`, `<base>/:code/chunk`, `<base>/:code/finish`).
 * @param {object} [opts.scope] — контекст раздела, уезжает в init (id реестра,
 *        папки, переписки).
 * @param {(p: number) => void} [opts.onProgress] — доля загруженного, 0..1.
 * @param {AbortSignal} [opts.signal] — отмена.
 * @returns {Promise<object>} метаданные файла от раздела.
 */
export async function uploadFileTo({ file, directUrl, chunkBase, scope = {}, onProgress, signal }) {
  const report = (p) => onProgress?.(Math.max(0, Math.min(1, p)))

  if (file.size <= CHUNK_THRESHOLD) {
    report(0)
    const form = new FormData()
    form.append('file', file)
    const res = await apiRequest(directUrl, {
      method: 'POST', body: form, signal, timeout: CHUNK_TIMEOUT,
    })
    report(1)
    return res
  }

  const session = await apiRequest(`${chunkBase}/init`, {
    method: 'POST',
    signal,
    body: { ...scope, file_name: file.name, mime: file.type || '', size: file.size },
  })
  const chunkSize = session.chunk_size || CHUNK_SIZE
  const total = Math.ceil(file.size / chunkSize)

  try {
    // Докачка: сервер помнит, сколько частей уже принял, и повторять их незачем.
    for (let index = session.received || 0; index < total; index++) {
      const chunk = file.slice(index * chunkSize, (index + 1) * chunkSize)
      await sendChunk(chunkBase, session.code, index, chunk, signal)
      report((index + 1) / total)
    }
    return await apiRequest(`${chunkBase}/${session.code}/finish`, {
      method: 'POST', signal, timeout: CHUNK_TIMEOUT,
    })
  } catch (e) {
    // Брошенные части убирает и фоновая уборка сервера, но отменённую человеком
    // загрузку честнее закрыть сразу — место освободится, не дожидаясь её.
    apiRequest(`${chunkBase}/${session.code}`, { method: 'DELETE' }).catch(() => {})
    throw e
  }
}

// sendChunk — часть с повтором. Сеть моргает чаще, чем ломается: одна неудачная
// часть не повод терять уже отправленные сотни мегабайт.
async function sendChunk(chunkBase, code, index, chunk, signal) {
  let lastError
  for (let attempt = 0; attempt < CHUNK_RETRIES; attempt++) {
    if (signal?.aborted) throw { status: 0, error: 'ABORTED', message: 'Загрузка отменена' }
    try {
      return await apiRequest(`${chunkBase}/${code}/chunk?index=${index}`, {
        method: 'POST', body: chunk, signal, timeout: CHUNK_TIMEOUT,
      })
    } catch (e) {
      // Отмена и отказ сервера повтора не заслуживают — только сбой связи.
      if (e?.error === 'ABORTED' || (e?.status > 0 && e.status !== 408)) throw e
      lastError = e
      await new Promise((r) => setTimeout(r, 500 * (attempt + 1)))
    }
  }
  throw lastError
}

// humanSize — размер файла для подписи под индикатором.
export function humanSize(bytes) {
  if (!bytes) return '0 Б'
  const units = ['Б', 'КБ', 'МБ', 'ГБ']
  const i = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)))
  const value = bytes / 1024 ** i
  return `${i === 0 ? value : value.toFixed(1)} ${units[i]}`
}
