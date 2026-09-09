import { describe, expect, it } from 'vitest'
import { createMp4Muxer } from './mp4.js'

const type = (bytes, offset) => String.fromCharCode(...bytes.slice(offset + 4, offset + 8))
const size = (bytes, offset) => new DataView(bytes.buffer, bytes.byteOffset).getUint32(offset)

async function bytesOf(blob) {
  return new Uint8Array(await blob.arrayBuffer())
}

describe('сборка MP4', () => {
  it('без кадров и описания файла не даёт', () => {
    const muxer = createMp4Muxer({ width: 320, height: 240, fps: 12 })
    expect(muxer.finish()).toBeNull()

    muxer.addSample(new Uint8Array([1, 2, 3]), true)
    // Описание кодека (avcC) обязательно: без него дорожку не прочитать.
    expect(muxer.finish()).toBeNull()
  })

  it('кладёт кадры подряд и закрывает файл таблицами', async () => {
    const muxer = createMp4Muxer({ width: 320, height: 240, fps: 12 })
    muxer.setDescription(new Uint8Array([1, 100, 0, 40]))
    muxer.addSample(new Uint8Array(120), true)
    muxer.addSample(new Uint8Array(40), false)

    expect(muxer.count).toBe(2)
    const blob = muxer.finish()
    expect(blob.type).toBe('video/mp4')

    const bytes = await bytesOf(blob)
    expect(type(bytes, 0)).toBe('ftyp')

    const mdat = size(bytes, 0)
    expect(type(bytes, mdat)).toBe('mdat')
    // Заголовок mdat плюс оба кадра.
    expect(size(bytes, mdat)).toBe(8 + 120 + 40)

    const moov = mdat + size(bytes, mdat)
    expect(type(bytes, moov)).toBe('moov')
    expect(moov + size(bytes, moov)).toBe(bytes.length)
  })

  it('описание берётся у первого кадра и потом не подменяется', async () => {
    const muxer = createMp4Muxer({ width: 16, height: 16, fps: 1 })
    muxer.setDescription(new Uint8Array([1, 1, 1, 1]))
    muxer.setDescription(new Uint8Array([9, 9, 9, 9, 9, 9]))
    muxer.addSample(new Uint8Array(8), true)

    const bytes = await bytesOf(muxer.finish())
    // avcC лежит внутри moov; ищем его по сигнатуре и сверяем длину.
    const marker = [...bytes].findIndex((_, i) => type(bytes, i) === 'avcC')
    expect(marker).toBeGreaterThan(0)
    expect(size(bytes, marker)).toBe(8 + 4)
  })
})
