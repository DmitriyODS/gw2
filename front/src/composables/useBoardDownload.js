/* Скачивание доски — ОДИН набор форматов и один код на все входы: меню внутри
   редактора и контекстное меню плитки в списке. Раньше список умел только svg и
   json, и «Скачать» из двух мест давало разное.

   Растр и PDF рисует клиент (в них попадают и картинки сцены), svg/json —
   сервер: он же кладёт в SVG сами файлы картинок. */
import * as api from '@/api/boards.js'
import { saveBlob, safeFileName } from '@/utils/download.js'
import { sceneToJpeg, sceneToPdf, sceneToPng } from '@/utils/boardExport.js'
import { useNotificationsStore } from '@/stores/notifications.js'

/** Пункты меню выгрузки — общий источник для обоих меню. */
export const BOARD_EXPORT_ITEMS = [
  { label: 'Картинка PNG', icon: 'image', action: 'export:png' },
  { label: 'Картинка JPG', icon: 'photo', action: 'export:jpg' },
  { label: 'Документ PDF', icon: 'picture_as_pdf', action: 'export:pdf' },
  { divider: true },
  { label: 'Вектор SVG', icon: 'draft', action: 'export:svg' },
  { label: 'Сцена JSON', icon: 'data_object', action: 'export:json' },
]

/** Формат из действия меню или null, если это не выгрузка. */
export function boardExportFormat(action) {
  return String(action || '').startsWith('export:') ? action.slice('export:'.length) : null
}

const RASTER = { png: sceneToPng, jpg: sceneToJpeg, pdf: sceneToPdf }

export function useBoardDownload() {
  const notify = useNotificationsStore()

  /** board — {id, title}; scene передаёт редактор (у него она уже открыта и
      могла измениться), список её догружает. */
  async function downloadBoard(board, format, scene = null) {
    const name = safeFileName(board?.title, 'Доска')
    try {
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
      await saveBlob(blob, `${name}.${format}`)
    } catch {
      notify.error('Не удалось выгрузить доску')
    }
  }

  return { downloadBoard }
}
