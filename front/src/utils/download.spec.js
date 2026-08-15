import { describe, it, expect, vi, beforeEach } from 'vitest'

// Мост нативной обёртки: сохранит файл он или нет — решает тест.
const saveNativeFile = vi.fn()
vi.mock('@/utils/nativeApp.js', () => ({ saveNativeFile: (...args) => saveNativeFile(...args) }))

const { saveBlob, safeFileName } = await import('./download.js')

describe('сохранение файла', () => {
  let clicks

  beforeEach(() => {
    saveNativeFile.mockReset()
    clicks = []
    URL.createObjectURL = vi.fn(() => 'blob:test')
    URL.revokeObjectURL = vi.fn()
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function click() {
      clicks.push({ href: this.href, download: this.download })
    })
  })

  it('в браузере уходит ссылкой с именем файла', async () => {
    saveNativeFile.mockResolvedValue(false)
    await saveBlob(new Blob(['x']), 'Отчёт.xlsx')

    expect(clicks).toEqual([{ href: 'blob:test', download: 'Отчёт.xlsx' }])
  })

  it('в мобильной обёртке файл забирает нативка — ссылку не создаём', async () => {
    // WebView не скачивает blob:-ссылки: щёлкнуть по ней значит потерять файл.
    saveNativeFile.mockResolvedValue(true)
    const blob = new Blob(['x'])
    await saveBlob(blob, 'Доска.png')

    expect(saveNativeFile).toHaveBeenCalledWith(blob, 'Доска.png')
    expect(clicks).toEqual([])
    expect(URL.createObjectURL).not.toHaveBeenCalled()
  })

  it('отказ обёртки (нет доступа к памяти) доходит до раздела', async () => {
    saveNativeFile.mockRejectedValue(new Error('Нет доступа к памяти устройства'))
    await expect(saveBlob(new Blob(['x']), 'Доска.png')).rejects.toThrow('Нет доступа')
  })

  it('имя файла чистится от разделителей пути', () => {
    expect(safeFileName('Отчёт 12/2026: итоги')).toBe('Отчёт 12 2026 итоги')
    expect(safeFileName('   ', 'Доска')).toBe('Доска')
  })
})
