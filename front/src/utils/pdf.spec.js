import { describe, it, expect } from 'vitest'
import { jpegPagesToPdf } from './pdf.js'

const page = (n = 1) => ({ bytes: new Uint8Array([0xff, 0xd8, n, 0xff, 0xd9]), width: 96, height: 192 })

async function text(blob) {
  // Байты картинки в тексте не важны — проверяем каркас документа.
  return new TextDecoder('latin1').decode(new Uint8Array(await blob.arrayBuffer()))
}

describe('jpegPagesToPdf', () => {
  it('одностраничный документ: каталог, страница и картинка на всю страницу', async () => {
    const pdf = await text(jpegPagesToPdf([page()]))
    expect(pdf.startsWith('%PDF-1.4')).toBe(true)
    expect(pdf).toContain('/Type /Catalog')
    expect(pdf).toContain('/Kids [3 0 R] /Count 1')
    expect(pdf).toContain('/Filter /DCTDecode')
    expect(pdf.endsWith('%%EOF\n')).toBe(true)
  })

  it('страниц может быть несколько — у каждой свои три объекта', async () => {
    const pdf = await text(jpegPagesToPdf([page(1), page(2), page(3)]))
    expect(pdf).toContain('/Kids [3 0 R 6 0 R 9 0 R] /Count 3')
    // 2 служебных объекта + 3 на страницу; в таблице xref столько же записей.
    expect(pdf).toContain('xref\n0 12\n')
    expect(pdf).toContain('/Size 12')
  })

  it('размер листа задаётся в пунктах — A4 остаётся A4 при любом разрешении растра', async () => {
    const a4 = { ...page(), width: 1240, height: 1754, ptWidth: 595, ptHeight: 842 }
    const pdf = await text(jpegPagesToPdf([a4]))
    expect(pdf).toContain('/MediaBox [0 0 595 842]')
    expect(pdf).toContain('/Width 1240 /Height 1754')
  })

  it('без явных пунктов размер считается из пикселей по dpi', async () => {
    const pdf = await text(jpegPagesToPdf([page()]))
    // 96 px при 96 dpi = 1 дюйм = 72 pt.
    expect(pdf).toContain('/MediaBox [0 0 72 144]')
  })

  /* Смещения объектов в xref обязаны указывать на их начало: иначе документ
     открывается не во всякой читалке. */
  it('таблица xref указывает на настоящие смещения объектов', async () => {
    const pdf = await text(jpegPagesToPdf([page()]))
    const offsets = [...pdf.matchAll(/^(\d{10}) 00000 n $/gm)].map((m) => Number(m[1]))
    expect(offsets).toHaveLength(5)
    offsets.forEach((off, i) => {
      expect(pdf.slice(off, off + 8)).toContain(`${i + 1} 0 obj`)
    })
  })
})
