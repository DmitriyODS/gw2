import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/boards.js', () => ({
  exportBoard: vi.fn(() => Promise.resolve(new Blob(['<svg/>']))),
  getBoard: vi.fn(() => Promise.resolve({ id: 7, scene: { objects: [{ type: 'text' }] } })),
}))

vi.mock('@/utils/download.js', () => ({
  saveBlob: vi.fn(() => Promise.resolve()),
  safeFileName: (name, fallback) => String(name || fallback),
}))

vi.mock('@/utils/boardExport.js', () => ({
  sceneToPng: vi.fn(() => Promise.resolve(new Blob(['png']))),
  sceneToJpeg: vi.fn(() => Promise.resolve(new Blob(['jpg']))),
  sceneToPdf: vi.fn(() => Promise.resolve(null)),
}))

vi.mock('@/utils/boardVideo.js', () => ({
  sceneToVideo: vi.fn(() => Promise.resolve({ blob: new Blob(['mp4']), ext: 'mp4' })),
  sceneToGif: vi.fn(() => Promise.resolve({ blob: new Blob(['gif']), ext: 'gif' })),
}))

import * as api from '@/api/boards.js'
import { saveBlob } from '@/utils/download.js'
import { sceneToPng } from '@/utils/boardExport.js'
import { sceneToGif, sceneToVideo } from '@/utils/boardVideo.js'
import { BOARD_EXPORT_ITEMS, boardExportFormat, useBoardDownload } from './useBoardDownload.js'

const board = { id: 7, title: 'Схема' }

describe('выгрузка доски', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('меню одинаково для редактора и списка', () => {
    expect(BOARD_EXPORT_ITEMS.filter((i) => i.action).map((i) => boardExportFormat(i.action)))
      .toEqual(['png', 'png-alpha', 'jpg', 'pdf', 'svg', 'json', 'animation'])
    expect(boardExportFormat('delete')).toBeNull()
  })

  it('svg и json строит сервер', async () => {
    const { downloadBoard } = useBoardDownload()
    await downloadBoard(board, 'svg')

    expect(api.exportBoard).toHaveBeenCalledWith(7, 'svg')
    expect(saveBlob).toHaveBeenCalledWith(expect.any(Blob), 'Схема.svg')
  })

  it('растр рисует клиент, а сцену для него список догружает', async () => {
    const { downloadBoard } = useBoardDownload()
    await downloadBoard(board, 'png')

    expect(api.getBoard).toHaveBeenCalledWith(7)
    expect(sceneToPng).toHaveBeenCalled()
    expect(saveBlob).toHaveBeenCalledWith(expect.any(Blob), 'Схема.png')
  })

  it('открытая доска отдаёт свою сцену — за сервером не ходим', async () => {
    const { downloadBoard } = useBoardDownload()
    const scene = { objects: [{ type: 'rect' }] }
    await downloadBoard(board, 'png', scene)

    expect(api.getBoard).not.toHaveBeenCalled()
    expect(sceneToPng).toHaveBeenCalledWith(scene)
  })

  it('анимацию собирает браузер, расширение приходит с файлом', async () => {
    const { downloadBoard } = useBoardDownload()
    const scene = { animation: { fps: 12, frames: [{ id: 'f1' }] }, objects: [] }
    await downloadBoard(board, 'mp4', scene, { from: 0, to: 0 })

    expect(sceneToVideo).toHaveBeenCalledWith(scene, { from: 0, to: 0 })
    expect(saveBlob).toHaveBeenCalledWith(expect.any(Blob), 'Схема.mp4')
  })

  it('гифку собирает свой кодировщик, а не видеомуксер', async () => {
    const { downloadBoard } = useBoardDownload()
    const scene = { animation: { fps: 12, frames: [{ id: 'f1' }] }, objects: [] }
    await downloadBoard(board, 'gif', scene, { loop: 0, transparent: true })

    expect(sceneToGif).toHaveBeenCalledWith(scene, { loop: 0, transparent: true })
    expect(sceneToVideo).not.toHaveBeenCalled()
    expect(saveBlob).toHaveBeenCalledWith(expect.any(Blob), 'Схема.gif')
  })

  it('PNG без фона сохраняется с обычным расширением', async () => {
    const { downloadBoard } = useBoardDownload()
    await downloadBoard(board, 'png-alpha', { objects: [{ type: 'rect' }] })

    expect(saveBlob).toHaveBeenCalledWith(expect.any(Blob), 'Схема.png')
  })

  it('доска без кадров видео не даёт', async () => {
    sceneToVideo.mockResolvedValueOnce(null)
    const { downloadBoard } = useBoardDownload()
    await downloadBoard(board, 'mp4', { objects: [] })

    expect(saveBlob).not.toHaveBeenCalled()
  })

  it('на пустой доске файл не сохраняется', async () => {
    const { downloadBoard } = useBoardDownload()
    await downloadBoard(board, 'pdf', { objects: [] })

    expect(saveBlob).not.toHaveBeenCalled()
  })
})
