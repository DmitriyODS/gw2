/* Сборка PDF из готовых растров — своя, без внешних зависимостей: генератор
   PDF ради документа «страница = картинка» тянул бы в бандл сотни килобайт.

   Умеет ровно то, что нужно платформе: многостраничный PDF 1.4, где каждая
   страница — один JPEG во всю страницу (доска, лист QR-кодов). */

/**
 * @param {Array<{bytes: Uint8Array, width: number, height: number,
 *                ptWidth?: number, ptHeight?: number}>} pages — растры страниц.
 *        ptWidth/ptHeight задают размер листа в пунктах (A4 — 595×842); без них
 *        он считается из пикселей по dpi.
 * @param {{dpi?: number}} [opts]
 * @returns {Blob}
 */
export function jpegPagesToPdf(pages, { dpi = 96 } = {}) {
  const encoder = new TextEncoder()
  const chunks = []
  const offsets = []
  let length = 0

  const push = (bytes) => {
    chunks.push(bytes)
    length += bytes.length
  }
  const pushText = (text) => push(encoder.encode(text))
  const startObject = (num) => {
    offsets[num] = length
    pushText(`${num} 0 obj\n`)
  }

  // Нумерация: 1 — каталог, 2 — дерево страниц, дальше по три объекта на лист
  // (страница, картинка-XObject, поток отрисовки).
  const pageObj = (i) => 3 + i * 3

  pushText('%PDF-1.4\n')

  startObject(1)
  pushText('<< /Type /Catalog /Pages 2 0 R >>\nendobj\n')

  startObject(2)
  const kids = pages.map((_, i) => `${pageObj(i)} 0 R`).join(' ')
  pushText(`<< /Type /Pages /Kids [${kids}] /Count ${pages.length} >>\nendobj\n`)

  pages.forEach((page, i) => {
    const num = pageObj(i)
    // 1 pt = 1/72 дюйма.
    const w = page.ptWidth ?? Math.round((page.width * 72) / dpi)
    const h = page.ptHeight ?? Math.round((page.height * 72) / dpi)

    startObject(num)
    pushText(`<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ${w} ${h}] `
      + `/Resources << /XObject << /Im0 ${num + 1} 0 R >> >> /Contents ${num + 2} 0 R >>\nendobj\n`)

    startObject(num + 1)
    pushText(`<< /Type /XObject /Subtype /Image /Width ${page.width} /Height ${page.height} `
      + `/ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length ${page.bytes.length} >>\nstream\n`)
    push(page.bytes)
    pushText('\nendstream\nendobj\n')

    const content = `q\n${w} 0 0 ${h} 0 0 cm\n/Im0 Do\nQ\n`
    startObject(num + 2)
    pushText(`<< /Length ${content.length} >>\nstream\n${content}endstream\nendobj\n`)
  })

  const xrefAt = length
  const count = 3 + pages.length * 3
  let xref = `xref\n0 ${count}\n0000000000 65535 f \n`
  for (let i = 1; i < count; i++) {
    xref += `${String(offsets[i]).padStart(10, '0')} 00000 n \n`
  }
  pushText(xref)
  pushText(`trailer\n<< /Size ${count} /Root 1 0 R >>\nstartxref\n${xrefAt}\n%%EOF\n`)

  return new Blob(chunks, { type: 'application/pdf' })
}

/** Растр canvas → байты JPEG (страница PDF). */
export async function canvasToJpegBytes(canvas, quality = 0.95) {
  const blob = await new Promise((resolve) => canvas.toBlob(resolve, 'image/jpeg', quality))
  if (!blob) return null
  return new Uint8Array(await blob.arrayBuffer())
}
