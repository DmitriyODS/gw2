/* Сборка MP4 из уже сжатых H.264-кадров — своя, без внешних зависимостей:
   готовый муксер тянет в бандл сотни килобайт ради одной кнопки «сохранить
   анимацию», а нужен нам ровно один случай — одна видеодорожка с постоянной
   частотой кадров и без звука (тот же приём, что в utils/pdf.js для PDF).

   Кадры сжимает браузер (WebCodecs VideoEncoder), здесь они лишь укладываются
   в контейнер: ftyp + mdat со всеми кадрами подряд + moov с таблицами. Порядок
   именно такой, потому что смещения кадров в moov известны заранее — размер
   заголовка mdat фиксирован. */

// Шкала времени дорожки: 90 кГц — стандарт для видео, любая частота кадров
// раскладывается по ней целыми числами достаточно точно.
const TIMESCALE = 90000

const text = (s) => Uint8Array.from(s, (ch) => ch.charCodeAt(0))

function u8(...values) {
  return Uint8Array.from(values)
}

function u16(value) {
  return u8((value >> 8) & 0xff, value & 0xff)
}

function u32(value) {
  return u8((value >>> 24) & 0xff, (value >>> 16) & 0xff, (value >>> 8) & 0xff, value & 0xff)
}

function concat(parts) {
  const size = parts.reduce((sum, p) => sum + p.length, 0)
  const out = new Uint8Array(size)
  let offset = 0
  for (const p of parts) {
    out.set(p, offset)
    offset += p.length
  }
  return out
}

/** Бокс MP4: размер, тип, содержимое. */
function box(type, ...parts) {
  const body = concat(parts)
  return concat([u32(body.length + 8), text(type), body])
}

/** Бокс с версией и флагами (полный бокс стандарта). */
function fullBox(type, version, flags, ...parts) {
  return box(type, u8(version, (flags >> 16) & 0xff, (flags >> 8) & 0xff, flags & 0xff), ...parts)
}

// Единичная матрица преобразования — её ждут все боксы дорожки и фильма.
const UNITY_MATRIX = concat([
  u32(0x00010000), u32(0), u32(0),
  u32(0), u32(0x00010000), u32(0),
  u32(0), u32(0), u32(0x40000000),
])

function zeros(count) {
  return new Uint8Array(count)
}

function ftyp() {
  return box('ftyp', text('isom'), u32(0x200), text('isom'), text('iso2'), text('avc1'), text('mp41'))
}

function avc1(width, height, description) {
  const avcC = box('avcC', description)
  return box(
    'avc1',
    zeros(6), u16(1),          // reserved + data_reference_index
    u16(0), u16(0), zeros(12), // pre_defined + reserved
    u16(width), u16(height),
    u32(0x00480000), u32(0x00480000), // 72 dpi
    u32(0), u16(1),            // reserved + frame_count
    zeros(32),                 // compressorname
    u16(0x0018), u16(0xffff),  // depth + pre_defined
    avcC,
  )
}

function stbl(samples, width, height, description, sampleDelta) {
  const sizes = concat(samples.map((s) => u32(s.data.length)))
  const offsets = concat(samples.map((s) => u32(s.offset)))
  const syncs = samples.map((s, i) => (s.key ? i + 1 : 0)).filter(Boolean)

  return box(
    'stbl',
    fullBox('stsd', 0, 0, u32(1), avc1(width, height, description)),
    // Все кадры одной длительности — таблица времён из одной записи.
    fullBox('stts', 0, 0, u32(1), u32(samples.length), u32(sampleDelta)),
    fullBox('stss', 0, 0, u32(syncs.length), concat(syncs.map(u32))),
    // По одному кадру в куске: так смещения совпадают с таблицей stco.
    fullBox('stsc', 0, 0, u32(1), u32(1), u32(1), u32(1)),
    fullBox('stsz', 0, 0, u32(0), u32(samples.length), sizes),
    fullBox('stco', 0, 0, u32(samples.length), offsets),
  )
}

function moov(samples, { width, height, description, duration, sampleDelta }) {
  const mvhd = fullBox('mvhd', 0, 0,
    u32(0), u32(0), u32(TIMESCALE), u32(duration),
    u32(0x00010000), u16(0x0100), zeros(10), UNITY_MATRIX, zeros(24), u32(2))

  const tkhd = fullBox('tkhd', 0, 3,
    u32(0), u32(0), u32(1), u32(0), u32(duration), zeros(8),
    u16(0), u16(0), u16(0), u16(0), UNITY_MATRIX,
    u32(width << 16), u32(height << 16))

  const mdhd = fullBox('mdhd', 0, 0,
    u32(0), u32(0), u32(TIMESCALE), u32(duration), u16(0x55c4), u16(0))

  const hdlr = fullBox('hdlr', 0, 0, u32(0), text('vide'), zeros(12), text('VideoHandler\0'))

  const dinf = box('dinf', fullBox('dref', 0, 0, u32(1), fullBox('url ', 0, 1)))
  const minf = box('minf', fullBox('vmhd', 0, 1, u16(0), zeros(6)), dinf,
    stbl(samples, width, height, description, sampleDelta))

  return box('moov', mvhd, box('trak', tkhd, box('mdia', mdhd, hdlr, minf)))
}

/** Муксер одной видеодорожки: кадры добавляются по мере кодирования, finish
    отдаёт готовый файл. */
export function createMp4Muxer({ width, height, fps }) {
  const samples = []
  const sampleDelta = Math.max(1, Math.round(TIMESCALE / Math.max(1, fps)))
  let description = null
  // Смещение первого кадра: ftyp целиком плюс заголовок mdat.
  const headerSize = ftyp().length + 8
  let cursor = headerSize

  return {
    /** description — avcC из metadata.decoderConfig первого чанка кодировщика. */
    setDescription(data) {
      if (data && !description) description = new Uint8Array(data)
    },
    addSample(data, key) {
      const bytes = new Uint8Array(data)
      samples.push({ data: bytes, offset: cursor, key: !!key })
      cursor += bytes.length
    },
    get count() {
      return samples.length
    },
    /** Готовый файл; null — кодировщик не отдал ни описания, ни кадров. */
    finish() {
      if (!samples.length || !description) return null
      const payload = concat(samples.map((s) => s.data))
      const mdat = concat([u32(payload.length + 8), text('mdat'), payload])
      const duration = samples.length * sampleDelta
      return new Blob([ftyp(), mdat, moov(samples, { width, height, description, duration, sampleDelta })],
        { type: 'video/mp4' })
    },
  }
}
