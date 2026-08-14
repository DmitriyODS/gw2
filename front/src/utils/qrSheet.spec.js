import { describe, it, expect, vi, afterEach } from 'vitest'
import { MAX_CODES, PER_PAGE, SHEET, chunkPages, codesToPdf, expandCodes, pageCount } from './qrSheet.js'

describe('qrSheet', () => {
  it('сетка листа — 4 × 6, 24 кода на страницу', () => {
    expect(SHEET.cols * SHEET.rows).toBe(PER_PAGE)
    expect(PER_PAGE).toBe(24)
  })

  it('позиция даёт столько кодов, сколько для неё заказали', () => {
    expect(expandCodes([{ value: 'A', count: 3 }, { value: 'B', count: 1 }]))
      .toEqual(['A', 'A', 'A', 'B'])
  })

  it('ноль копий исключает позицию, не удаляя её из списка', () => {
    expect(expandCodes([{ value: 'A', count: 0 }, { value: 'B', count: 2 }]))
      .toEqual(['B', 'B'])
  })

  it('потолок задания не превышается', () => {
    expect(expandCodes([{ value: 'A', count: 5000 }])).toHaveLength(MAX_CODES)
    expect(expandCodes([{ value: 'A', count: 10 }], 4)).toHaveLength(4)
  })

  it('страниц считается по 24 кода, но не меньше одной', () => {
    expect(pageCount(0)).toBe(1)
    expect(pageCount(24)).toBe(1)
    expect(pageCount(25)).toBe(2)
    expect(pageCount(48)).toBe(2)
  })

  it('ячейки раскладываются по страницам целиком', () => {
    const cells = Array.from({ length: 30 }, (_, i) => ({ value: `V${i}`, src: 'x' }))
    const pages = chunkPages(cells)
    expect(pages).toHaveLength(2)
    expect(pages[0]).toHaveLength(PER_PAGE)
    expect(pages[1]).toHaveLength(6)
  })
})

/* Растеризатора в jsdom нет, поэтому canvas и загрузку кодов подменяем
   заглушками: проверяем не пиксели, а раскладку — сколько листов вышло, все ли
   коды нарисованы и куда встала первая ячейка. */
describe('codesToPdf', () => {
  afterEach(() => vi.restoreAllMocks())

  function stubCanvas() {
    const ctx = {
      fillStyle: '', font: '', textAlign: '', textBaseline: '', imageSmoothingEnabled: true,
      fillRect: vi.fn(), drawImage: vi.fn(), fillText: vi.fn(),
      measureText: (t) => ({ width: String(t).length * 12 }),
    }
    const canvas = {
      width: 0,
      height: 0,
      getContext: () => ctx,
      toBlob: (cb) => cb(new Blob([new Uint8Array([0xff, 0xd8, 0xff, 0xd9])], { type: 'image/jpeg' })),
    }
    const native = document.createElement.bind(document)
    vi.spyOn(document, 'createElement').mockImplementation((tag) => (
      tag === 'canvas' ? canvas : native(tag)
    ))
    // Картинка кода «загружается» сразу — иначе сборка ждала бы вечно.
    vi.stubGlobal('Image', class {
      set src(v) { this._src = v; queueMicrotask(() => this.onload?.()) }
      get src() { return this._src }
    })
    return { canvas, ctx }
  }

  const cells = (n) => Array.from({ length: n }, (_, i) => ({ value: `INV-${i + 1}`, src: `qr-${i + 1}` }))

  it('лист остаётся A4, кодов на странице — 24, страниц столько, сколько нужно', async () => {
    const { canvas, ctx } = stubCanvas()
    const blob = await codesToPdf(cells(25))
    expect(blob.type).toBe('application/pdf')

    const pdf = new TextDecoder('latin1').decode(new Uint8Array(await blob.arrayBuffer()))
    expect(pdf).toContain('/Count 2')
    expect(pdf).toContain('/MediaBox [0 0 595 842]')
    // 150 dpi: A4 — 1240 × 1754 px.
    expect([canvas.width, canvas.height]).toEqual([1240, 1754])
    // Все коды нарисованы, каждый — квадрат 30 мм.
    expect(ctx.drawImage).toHaveBeenCalledTimes(25)
    const [, , , w, h] = ctx.drawImage.mock.calls[0]
    expect([w, h]).toEqual([177, 177])
  })

  it('подпись под кодом обрезается многоточием, а не наезжает на соседей', async () => {
    const { ctx } = stubCanvas()
    await codesToPdf([{ value: 'ОЧЕНЬ-ДЛИННЫЙ-ИНВЕНТАРНЫЙ-НОМЕР-КОТОРЫЙ-НЕ-ВЛЕЗЕТ', src: 'qr' }])
    const lines = ctx.fillText.mock.calls.map((c) => c[0])
    expect(lines).toHaveLength(SHEET.capLines)
    expect(lines.at(-1).endsWith('…')).toBe(true)
  })

  it('печатать нечего — файла нет', async () => {
    stubCanvas()
    expect(await codesToPdf([])).toBeNull()
  })
})
