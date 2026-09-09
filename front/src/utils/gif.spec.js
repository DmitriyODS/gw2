import { describe, expect, it } from 'vitest'
import { createGifEncoder } from './gif.js'

/** Кадр-заглушка: сплошной цвет, при желании с прозрачной половиной. */
function frame(width, height, [r, g, b], alpha = 255) {
  const data = new Uint8ClampedArray(width * height * 4)
  for (let i = 0; i < data.length; i += 4) {
    data[i] = r; data[i + 1] = g; data[i + 2] = b
    data[i + 3] = i < data.length / 2 ? 255 : alpha
  }
  return { data, width, height }
}

async function bytesOf(blob) {
  return new Uint8Array(await blob.arrayBuffer())
}

const text = (bytes, at, len) => String.fromCharCode(...bytes.slice(at, at + len))

describe('сборка GIF', () => {
  it('без кадров файла нет', () => {
    expect(createGifEncoder({ width: 4, height: 4 }).finish()).toBeNull()
  })

  it('пишет заголовок, зацикливание и завершающий байт', async () => {
    const gif = createGifEncoder({ width: 8, height: 8, loop: 0 })
    gif.addFrame(frame(8, 8, [255, 0, 0]), 100)
    gif.addFrame(frame(8, 8, [0, 0, 255]), 100)
    expect(gif.count).toBe(2)

    const bytes = await bytesOf(gif.finish())
    expect(text(bytes, 0, 6)).toBe('GIF89a')
    // Расширение Netscape — единственный способ задать повторы.
    expect([...bytes].some((_, i) => text(bytes, i, 11) === 'NETSCAPE2.0')).toBe(true)
    expect(bytes[bytes.length - 1]).toBe(0x3b)
    // Два кадра — два дескриптора изображения.
    expect([...bytes].filter((b, i) => b === 0x2c && i > 13).length).toBeGreaterThanOrEqual(2)
  })

  it('прозрачность включает флаг в управляющем расширении', async () => {
    const opaque = createGifEncoder({ width: 8, height: 8 })
    opaque.addFrame(frame(8, 8, [10, 200, 10]), 80)
    const solid = await bytesOf(opaque.finish())

    const clear = createGifEncoder({ width: 8, height: 8, transparent: true })
    clear.addFrame(frame(8, 8, [10, 200, 10], 0), 80)
    const alpha = await bytesOf(clear.finish())

    // Байт упаковки управляющего расширения идёт сразу после «!F9 04».
    const packedAt = (bytes) => bytes.findIndex((b, i) => b === 0x21 && bytes[i + 1] === 0xf9) + 3
    expect(solid[packedAt(solid)] & 0x01).toBe(0)
    expect(alpha[packedAt(alpha)] & 0x01).toBe(1)
  })

  it('кадр с одним цветом жмётся в считаные байты', async () => {
    const gif = createGifEncoder({ width: 64, height: 64 })
    gif.addFrame(frame(64, 64, [0, 0, 0]), 40)
    const bytes = await bytesOf(gif.finish())
    // 64×64 пикселя одного цвета: LZW обязан сжать их в сотни байт, а не тысячи.
    expect(bytes.length).toBeLessThan(1200)
  })
})
