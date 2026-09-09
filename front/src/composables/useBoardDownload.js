/* Скачивание доски — ОДИН набор форматов и один код на все входы: меню внутри
   редактора и контекстное меню плитки в списке. Раньше список умел только svg и
   json, и «Скачать» из двух мест давало разное.

   Растр, PDF и видео рисует клиент (в них попадают и картинки сцены), svg/json —
   сервер: он же кладёт в SVG сами файлы картинок. Видео собирает браузер, и
   MP4 умеют не все — расширение приходит вместе с готовым файлом. */
import * as api from '@/api/boards.js'
import { saveBlob, safeFileName } from '@/utils/download.js'
import { sceneToJpeg, sceneToPdf, sceneToPng } from '@/utils/boardExport.js'
import { useNotificationsStore } from '@/stores/notifications.js'

/** Пункты меню выгрузки — общий источник для обоих меню. */
export const BOARD_EXPORT_ITEMS = [
  { label: 'Картинка PNG', icon: 'image', action: 'export:png' },
  { label: 'PNG без фона', icon: 'texture', action: 'export:png-alpha' },
  { label: 'Картинка JPG', icon: 'photo', action: 'export:jpg' },
  { label: 'Документ PDF', icon: 'picture_as_pdf', action: 'export:pdf' },
  { divider: true },
  { label: 'Вектор SVG', icon: 'draft', action: 'export:svg' },
  { label: 'Сцена JSON', icon: 'data_object', action: 'export:json' },
  { divider: true },
  { label: 'Анимация: видео и GIF', icon: 'movie', action: 'export:animation' },
]

/** Формат из действия меню или null, если это не выгрузка. */
export function boardExportFormat(action) {
  return String(action || '').startsWith('export:') ? action.slice('export:'.length) : null
}

const RASTER = {
  png: sceneToPng,
  'png-alpha': (scene) => sceneToPng(scene, { transparent: true }),
  jpg: sceneToJpeg,
  pdf: sceneToPdf,
}

// Расширение файла по формату: у «PNG без фона» оно обычное.
const EXT = { 'png-alpha': 'png' }

export function useBoardDownload() {
  const notify = useNotificationsStore()

  /** board — {id, title}; scene передаёт редактор (у него она уже открыта и
      могла измениться), список её догружает. options — настройки анимации
      (кадры цикла, частота, размер, прозрачный фон). */
  async function downloadBoard(board, format, scene = null, options = {}) {
    const name = safeFileName(board?.title, 'Доска')
    try {
      if (format === 'mp4' || format === 'gif') {
        const data = scene || (await api.getBoard(board.id)).scene
        // Кодировщики грузятся только здесь: анимацию выгружают редко, а в
        // чанк редактора муксер и GIF попадать не должны.
        const { sceneToGif, sceneToVideo } = await import('@/utils/boardVideo.js')
        const made = format === 'gif' ? await sceneToGif(data, options) : await sceneToVideo(data, options)
        if (!made) {
          notify.warn('Для анимации нужны кадры с рисунком')
          return
        }
        await saveBlob(made.blob, `${name}.${made.ext}`)
        return
      }
      const make = RASTER[format]
      if (!make) {
        await saveBlob(await api.exportBoard(board.id, format), `${name}.${format}`)
        return
      }
      const data = scene || (await api.getBoard(board.id)).scene
      const blob = await make(data)
      if (!blob) {
        notify.warn('На доске пока нечего сохранять')
        return
      }
      await saveBlob(blob, `${name}.${EXT[format] || format}`)
    } catch (e) {
      if (e?.name === 'AbortError') return
      if (e?.message === 'VIDEO_UNSUPPORTED') notify.error('Этот браузер не умеет собирать видео')
      else notify.error('Не удалось выгрузить доску')
    }
  }

  return { downloadBoard }
}
