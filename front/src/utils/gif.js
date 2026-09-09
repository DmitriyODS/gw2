/* Сборка анимированного GIF — своя, без внешних зависимостей: готовый
   кодировщик тянет в бандл сотни килобайт ради одной кнопки, а нужен ровно
   один случай — последовательность кадров одного размера с зацикливанием
   (тот же приём, что в utils/pdf.js и utils/mp4.js).

   GIF знает только 256 цветов на кадр, поэтому у каждого кадра своя палитра
   (медианное сечение по его же пикселям) — так градиенты и полутона держатся
   заметно лучше общей палитры на весь ролик. Прозрачность в GIF — ОДИН
   зарезервированный индекс, полупрозрачных пикселей формат не знает: всё, что
   прозрачнее половины, становится дырой, остальное рисуется плотно. */

const TRANSPARENT_ALPHA = 128

/** Побитовый писатель LZW: коды идут младшими битами вперёд, данные уезжают
    блоками до 255 байт — так их ждёт формат. */
function createBitWriter() {
  const bytes = []
  let current = 0
  let bits = 0
  return {
    write(code, length) {
      current |= code << bits
      bits += length
      while (bits >= 8) {
        bytes.push(current & 0xff)
        current >>= 8
        bits -= 8
      }
    },
    finish() {
      if (bits > 0) bytes.push(current & 0xff)
      return bytes
    },
  }
}

/** LZW-сжатие индексов пикселей (алгоритм из спецификации GIF89a). */
function lzwEncode(indices, minCodeSize) {
  const clearCode = 1 << minCodeSize
  const eoiCode = clearCode + 1
  const writer = createBitWriter()
  let codeSize = minCodeSize + 1
  let next = eoiCode + 1
  let dict = new Map()

  writer.write(clearCode, codeSize)
  let prefix = indices[0]
  for (let i = 1; i < indices.length; i += 1) {
    const k = indices[i]
    const key = prefix * 4096 + k
    if (dict.has(key)) {
      prefix = dict.get(key)
      continue
    }
    writer.write(prefix, codeSize)
    if (next < 4096) {
      dict.set(key, next)
      next += 1
      if (next > (1 << codeSize) && codeSize < 12) codeSize += 1
    } else {
      // Словарь заполнен — сбрасываем его, иначе кодировать дальше нечем.
      writer.write(clearCode, codeSize)
      dict = new Map()
      next = eoiCode + 1
      codeSize = minCodeSize + 1
    }
    prefix = k
  }
  writer.write(prefix, codeSize)
  writer.write(eoiCode, codeSize)
  return writer.finish()
}

/* Медианное сечение: коробка цветов делится по самому широкому каналу, пока
   коробок не станет столько, сколько нужно цветов. Группируем по огрублённым
   до 5 бит цветам (иначе на большом кадре сортируется миллион точек), но
   СУММЫ храним по настоящим значениям — иначе палитра systematically темнее
   картинки на половину шага огрубления. */
function medianCut(pixels, maxColors) {
  const seen = new Map()
  for (let i = 0; i < pixels.length; i += 4) {
    if (pixels[i + 3] < TRANSPARENT_ALPHA) continue
    const key = ((pixels[i] >> 3) << 10) | ((pixels[i + 1] >> 3) << 5) | (pixels[i + 2] >> 3)
    const found = seen.get(key)
    if (found) {
      found.n += 1
      found.sr += pixels[i]; found.sg += pixels[i + 1]; found.sb += pixels[i + 2]
    } else {
      seen.set(key, {
        r: pixels[i], g: pixels[i + 1], b: pixels[i + 2],
        sr: pixels[i], sg: pixels[i + 1], sb: pixels[i + 2], n: 1,
      })
    }
  }
  const colors = [...seen.values()]
  if (!colors.length) return [{ r: 0, g: 0, b: 0 }]
  if (colors.length <= maxColors) return colors

  let boxes = [colors]
  while (boxes.length < maxColors) {
    // Делим самую «широкую» коробку: она вносит больше всего ошибки.
    let target = -1
    let spread = -1
    boxes.forEach((box, i) => {
      if (box.length < 2) return
      const s = channelSpread(box)
      if (s.range > spread) { spread = s.range; target = i }
    })
    if (target < 0) break
    const box = boxes[target]
    const { channel } = channelSpread(box)
    box.sort((a, b) => a[channel] - b[channel])
    const mid = Math.floor(box.length / 2)
    boxes = [...boxes.slice(0, target), box.slice(0, mid), box.slice(mid), ...boxes.slice(target + 1)]
  }
  return boxes.filter((b) => b.length).map(averageColor)
}

function channelSpread(box) {
  const min = { r: 255, g: 255, b: 255 }
  const max = { r: 0, g: 0, b: 0 }
  for (const c of box) {
    for (const ch of ['r', 'g', 'b']) {
      if (c[ch] < min[ch]) min[ch] = c[ch]
      if (c[ch] > max[ch]) max[ch] = c[ch]
    }
  }
  const ranges = { r: max.r - min.r, g: max.g - min.g, b: max.b - min.b }
  const channel = ranges.r >= ranges.g && ranges.r >= ranges.b ? 'r' : (ranges.g >= ranges.b ? 'g' : 'b')
  return { channel, range: ranges[channel] }
}

function averageColor(box) {
  let r = 0; let g = 0; let b = 0; let n = 0
  for (const c of box) {
    r += c.sr; g += c.sg; b += c.sb; n += c.n
  }
  return n ? { r: Math.round(r / n), g: Math.round(g / n), b: Math.round(b / n) } : { r: 0, g: 0, b: 0 }
}

/** Индексы пикселей по палитре: ближайший цвет перебором (палитра мала). */
function quantize(pixels, palette, transparentIndex) {
  const indices = new Uint8Array(pixels.length / 4)
  const cache = new Map()
  for (let i = 0, p = 0; i < pixels.length; i += 4, p += 1) {
    if (transparentIndex >= 0 && pixels[i + 3] < TRANSPARENT_ALPHA) {
      indices[p] = transparentIndex
      continue
    }
    const key = (pixels[i] << 16) | (pixels[i + 1] << 8) | pixels[i + 2]
    const cached = cache.get(key)
    if (cached !== undefined) {
      indices[p] = cached
      continue
    }
    let best = 0
    let bestDist = Infinity
    for (let c = 0; c < palette.length; c += 1) {
      if (c === transparentIndex) continue
      const dr = pixels[i] - palette[c].r
      const dg = pixels[i + 1] - palette[c].g
      const db = pixels[i + 2] - palette[c].b
      const dist = dr * dr + dg * dg + db * db
      if (dist < bestDist) { bestDist = dist; best = c }
    }
    cache.set(key, best)
    indices[p] = best
  }
  return indices
}

/** Кодировщик анимации: кадры добавляются по одному, finish отдаёт файл.
    loop: 0 — бесконечно, N — столько повторов. */
export function createGifEncoder({ width, height, loop = 0, transparent = false }) {
  const bytes = []
  const push = (...values) => bytes.push(...values)
  const pushString = (s) => { for (const ch of s) bytes.push(ch.charCodeAt(0)) }
  const pushShort = (v) => { bytes.push(v & 0xff, (v >> 8) & 0xff) }

  pushString('GIF89a')
  pushShort(width)
  pushShort(height)
  // Глобальной палитры нет: у каждого кадра своя, локальная.
  push(0x70, 0, 0)

  // Расширение Netscape — единственный способ задать зацикливание.
  pushString('!')
  push(0xff, 0x0b)
  pushString('NETSCAPE2.0')
  push(0x03, 0x01)
  pushShort(loop)
  push(0)

  let frames = 0

  return {
    get count() {
      return frames
    },
    /** imageData — пиксели кадра (RGBA), delayMs — сколько его держать. */
    addFrame(imageData, delayMs) {
      const pixels = imageData.data
      const maxColors = transparent ? 255 : 256
      const palette = medianCut(pixels, maxColors)
      const transparentIndex = transparent ? palette.length : -1
      if (transparent) palette.push({ r: 0, g: 0, b: 0 })

      let bits = 1
      while ((1 << bits) < palette.length) bits += 1
      const tableSize = 1 << bits

      // Управляющее расширение: задержка, прозрачный индекс и «очистить кадр»
      // перед следующим — без очистки прозрачные места копили бы прошлые.
      pushString('!')
      push(0xf9, 0x04)
      push((transparent ? 0x09 : 0x08) | 0)
      pushShort(Math.max(1, Math.round(delayMs / 10)))
      push(transparent ? transparentIndex : 0, 0)

      pushString(',')
      pushShort(0)
      pushShort(0)
      pushShort(width)
      pushShort(height)
      push(0x80 | (bits - 1)) // локальная палитра, без чересстрочности

      for (let i = 0; i < tableSize; i += 1) {
        const c = palette[i] || { r: 0, g: 0, b: 0 }
        push(c.r, c.g, c.b)
      }

      const minCodeSize = Math.max(2, bits)
      const indices = quantize(pixels, palette, transparentIndex)
      const data = lzwEncode(indices, minCodeSize)
      push(minCodeSize)
      for (let i = 0; i < data.length; i += 255) {
        const chunk = data.slice(i, i + 255)
        push(chunk.length, ...chunk)
      }
      push(0)
      frames += 1
    },
    finish() {
      if (!frames) return null
      push(0x3b)
      return new Blob([new Uint8Array(bytes)], { type: 'image/gif' })
    },
  }
}
