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

import * as api from '@/api/boards.js'
import { saveBlob } from '@/utils/download.js'
import { sceneToPng } from '@/utils/boardExport.js'
import { BOARD_EXPORT_ITEMS, boardExportFormat, useBoardDownload } from './useBoardDownload.js'

const board = { id: 7, title: 'Схема' }

describe('выгрузка доски', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('меню одинаково для редактора и списка — пять форматов', () => {
    expect(BOARD_EXPORT_ITEMS.filter((i) => i.action).map((i) => boardExportFormat(i.action)))
      .toEqual(['png', 'jpg', 'pdf', 'svg', 'json'])
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

  it('на пустой доске файл не сохраняется', async () => {
    const { downloadBoard } = useBoardDownload()
    await downloadBoard(board, 'pdf', { objects: [] })

    expect(saveBlob).not.toHaveBeenCalled()
  })
})
