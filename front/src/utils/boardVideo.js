/* Выгрузка покадровой анимации доски: видео и GIF.

   Кадры рисует тот же движок сцены, что и холст, — кодирует их браузер:
   WebCodecs (VideoEncoder) даёт готовые H.264-кадры, а контейнер собирает свой
   муксер (utils/mp4.js). Где WebCodecs нет — пишем через MediaRecorder с
   ручной подачей кадров; он отдаёт MP4 не везде, поэтому расширение файла
   приходит вместе с содержимым, а не назначается заранее. GIF собирает свой
   кодировщик (utils/gif.js) — он же умеет зацикливание и прозрачный фон.

   По умолчанию кадр рисуется на БЕЛОМ листе фиксированной палитрой
   (EXPORT_COLORS): файл уезжает наружу, где темы приложения нет. Прозрачный
   фон просят намеренно — тогда листа нет вовсе (видео H.264 прозрачности не
   знает, поэтому там его и не предлагают). */
import { EXPORT_COLORS, FPS_RANGE, normalizeScene, orderedObjects, sceneBounds } from '@/utils/boardScene.js'
import { renderScene } from '@/utils/boardRender.js'
import { loadSceneImages } from '@/utils/boardExport.js'

const PAD = 32
const MAX_SIDE = 1920
// GIF хранит кадры целиком и без межкадрового сжатия — гигантский ролик из
// него делать незачем, поэтому своя, меньшая граница стороны.
const GIF_MAX_SIDE = 900

// Кодеки по убыванию качества профиля: baseline открывается везде, main/high
// лучше жмут. Берём первый, который согласился взять наше разрешение.
const CODECS = ['avc1.640028', 'avc1.4d0028', 'avc1.42e01e']

// Типы для запасного пути: MP4 умеют не все браузеры, WebM — почти все.
const RECORDER_TYPES = [
  { mime: 'video/mp4;codecs=avc1.42E01E', ext: 'mp4' },
  { mime: 'video/mp4', ext: 'mp4' },
  { mime: 'video/webm;codecs=vp9', ext: 'webm' },
  { mime: 'video/webm', ext: 'webm' },
]

const even = (n) => Math.max(2, Math.round(n / 2) * 2)
const sleep = (ms) => new Promise((resolve) => { setTimeout(resolve, ms) })

/** Сводка об анимации для диалога выгрузки. */
export function animationInfo(scene) {
  const s = normalizeScene(scene)
  const frames = s.animation?.frames || []
  return { frames: frames.map((f, i) => ({ id: f.id, name: f.name || `Кадр ${i + 1}` })), fps: s.animation?.fps || FPS_RANGE.def }
}

/** Подготовить холст и функцию отрисовки кадра по его id. */
async function prepare(scene, { maxSide = MAX_SIDE, transparent = false, from = 0, to = null, fps } = {}) {
  const s = normalizeScene(scene)
  const all = s.animation?.frames || []
  // Общая рамка ВСЕХ кадров, чтобы картинка не «прыгала» от кадра к кадру.
  const box = sceneBounds(orderedObjects(s, { onlyVisible: true }))
  if (!all.length || !box) return null

  const start = Math.max(0, Math.min(from, all.length - 1))
  const end = to == null ? all.length - 1 : Math.max(start, Math.min(to, all.length - 1))
  const frames = all.slice(start, end + 1)

  const rawW = box.w + PAD * 2
  const rawH = box.h + PAD * 2
  const scale = Math.min(2, maxSide / Math.max(rawW, rawH))
  const width = even(Math.min(maxSide, rawW * scale))
  const height = even(Math.min(maxSide, rawH * scale))

  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d', { willReadFrequently: true })
  if (!ctx) return null

  const images = await loadSceneImages(s)
  const camera = { x: box.x - PAD, y: box.y - PAD, scale }

  const drawFrame = (frameId) => {
    ctx.setTransform(1, 0, 0, 1, 0, 0)
    ctx.clearRect(0, 0, width, height)
    if (!transparent) {
      ctx.fillStyle = '#ffffff'
      ctx.fillRect(0, 0, width, height)
    }
    renderScene(ctx, s, { width, height, camera, images, background: false, colors: EXPORT_COLORS, frame: frameId })
  }

  return {
    canvas, ctx, width, height, frames, drawFrame,
    fps: Math.min(FPS_RANGE.max, Math.max(FPS_RANGE.min, Math.round(fps || s.animation.fps || FPS_RANGE.def))),
  }
}

/** Кодировщик WebCodecs: возвращает Blob или null, если путь недоступен. */
async function encodeWithCodecs({ canvas, width, height, frames, fps, drawFrame }, onProgress, signal) {
  if (typeof window === 'undefined' || typeof window.VideoEncoder === 'undefined') return null
  const { createMp4Muxer } = await import('@/utils/mp4.js')

  const bitrate = Math.round(Math.min(20e6, Math.max(1.5e6, width * height * fps * 0.07)))
  let codec = null
  for (const candidate of CODECS) {
    try {
      const support = await window.VideoEncoder.isConfigSupported({ codec: candidate, width, height, bitrate, framerate: fps })
      if (support?.supported) { codec = candidate; break }
    } catch { /* следующий кандидат */ }
  }
  if (!codec) return null

  const muxer = createMp4Muxer({ width, height, fps })
  let failure = null
  const encoder = new window.VideoEncoder({
    output: (chunk, meta) => {
      const description = meta?.decoderConfig?.description
      if (description) muxer.setDescription(description)
      const buffer = new ArrayBuffer(chunk.byteLength)
      chunk.copyTo(buffer)
      muxer.addSample(buffer, chunk.type === 'key')
    },
    error: (err) => { failure = err },
  })
  encoder.configure({ codec, width, height, bitrate, framerate: fps, avc: { format: 'avc' } })

  const step = Math.round(1e6 / fps)
  for (let i = 0; i < frames.length && !failure; i += 1) {
    if (signal?.aborted) throw new DOMException('Отменено', 'AbortError')
    drawFrame(frames[i].id)
    const frame = new window.VideoFrame(canvas, { timestamp: i * step, duration: step })
    // Ключевой кадр раз в секунду: перемотка ролика остаётся быстрой.
    encoder.encode(frame, { keyFrame: i % fps === 0 })
    frame.close()
    onProgress?.((i + 1) / frames.length)
    // Очередь не должна расти без предела — иначе память съедает сам буфер.
    if (encoder.encodeQueueSize > 8) await sleep(0)
  }
  await encoder.flush()
  encoder.close()
  if (failure) throw failure
  const blob = muxer.finish()
  return blob ? { blob, ext: 'mp4' } : null
}

/** Запасной путь: запись потока холста в реальном времени. */
async function recordFallback({ canvas, frames, fps, drawFrame }, onProgress, signal) {
  if (typeof window === 'undefined' || typeof window.MediaRecorder === 'undefined' || !canvas.captureStream) return null
  const type = RECORDER_TYPES.find((t) => window.MediaRecorder.isTypeSupported?.(t.mime))
  if (!type) return null

  // Поток без автозахвата: кадр отдаём сами — так ролик не зависит от того,
  // успел ли браузер снять холст между отрисовками.
  const stream = canvas.captureStream(0)
  const track = stream.getVideoTracks()[0]
  const chunks = []
  const recorder = new window.MediaRecorder(stream, { mimeType: type.mime })
  recorder.ondataavailable = (e) => { if (e.data?.size) chunks.push(e.data) }
  const finished = new Promise((resolve) => { recorder.onstop = resolve })
  recorder.start()

  const hold = 1000 / fps
  try {
    for (let i = 0; i < frames.length; i += 1) {
      if (signal?.aborted) throw new DOMException('Отменено', 'AbortError')
      drawFrame(frames[i].id)
      track.requestFrame?.()
      onProgress?.((i + 1) / frames.length)
      await sleep(hold)
    }
  } finally {
    recorder.stop()
    track.stop()
  }
  await finished
  if (!chunks.length) return null
  return { blob: new Blob(chunks, { type: type.mime }), ext: type.ext }
}

/** Ролик покадровой анимации: { blob, ext } либо null, если кадров нет.
    from/to — диапазон кадров, fps — частота (по умолчанию как у доски).
    Бросает ошибку, если браузер не умеет кодировать видео. */
export async function sceneToVideo(scene, { onProgress, maxSide, signal, from, to, fps } = {}) {
  const prepared = await prepare(scene, { maxSide, from, to, fps })
  if (!prepared) return null
  const encoded = await encodeWithCodecs(prepared, onProgress, signal)
  if (encoded) return encoded
  const recorded = await recordFallback(prepared, onProgress, signal)
  if (recorded) return recorded
  throw new Error('VIDEO_UNSUPPORTED')
}

/** Анимированный GIF: { blob, ext } либо null, если кадров нет.
    loop — сколько раз повторить (0 — бесконечно), transparent — без листа. */
export async function sceneToGif(scene, {
  onProgress, maxSide = GIF_MAX_SIDE, signal, from, to, fps, loop = 0, transparent = false,
} = {}) {
  const prepared = await prepare(scene, { maxSide, transparent, from, to, fps })
  if (!prepared) return null
  const { createGifEncoder } = await import('@/utils/gif.js')
  const { ctx, width, height, frames, drawFrame } = prepared
  const encoder = createGifEncoder({ width, height, loop, transparent })
  const delay = 1000 / prepared.fps

  for (let i = 0; i < frames.length; i += 1) {
    if (signal?.aborted) throw new DOMException('Отменено', 'AbortError')
    drawFrame(frames[i].id)
    encoder.addFrame(ctx.getImageData(0, 0, width, height), delay)
    onProgress?.((i + 1) / frames.length)
    // Кодирование кадра синхронное и не быстрое — уступаем поток, иначе
    // страница замирает на всю выгрузку.
    await sleep(0)
  }
  const blob = encoder.finish()
  return blob ? { blob, ext: 'gif' } : null
}
