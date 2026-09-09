import { beforeEach, describe, expect, it, vi } from 'vitest'
import { OBJ } from './boardScene.js'
import { renderScene } from './boardRender.js'

/* jsdom не умеет рисовать, поэтому 2d-контекст подменяется записывающей
   заглушкой: тесты проверяют не картинку, а СПОСОБ отрисовки — в буфер или
   прямо на холст, с каким наложением и сколько раз слои сводятся вместе. */
function fakeContext(canvas, log) {
  const state = { canvas, globalAlpha: 1, globalCompositeOperation: 'source-over' }
  return new Proxy(state, {
    get(target, key) {
      if (key in target) return target[key]
      if (key === 'getTransform') return () => ({ a: 1, b: 0, c: 0, d: 1, e: 0, f: 0 })
      if (key === 'measureText') return () => ({ width: 10 })
      if (key === 'createLinearGradient') return () => ({ addColorStop() {} })
      return (...args) => {
        log.push({ op: key, args, alpha: target.globalAlpha, blend: target.globalCompositeOperation })
      }
    },
    set(target, key, value) {
      target[key] = value
      if (key === 'globalCompositeOperation' || key === 'globalAlpha') log.push({ op: `set:${key}`, value })
      return true
    },
  })
}

function surface(log) {
  const canvas = { width: 200, height: 100, getContext: null }
  canvas.getContext = () => fakeContext(canvas, log)
  return canvas
}

const camera = { x: 0, y: 0, scale: 1 }
const stroke = (id, layer, extra = {}) => ({
  id, type: OBJ.path, layer, points: [0, 0, 10, 10, 20, 0], width: 4, color: 'ink', ...extra,
})

/* Буферы слоёв переиспользуются пулом внутри модуля, поэтому в очередном
   тесте нового canvas может и не появиться: копим их за весь прогон, а перед
   каждым тестом чистим только записи. */
const buffers = []

beforeEach(() => {
  buffers.forEach((b) => { b.log.length = 0 })
  vi.spyOn(document, 'createElement').mockImplementation((tag) => {
    if (tag !== 'canvas') return {}
    const log = []
    const canvas = surface(log)
    buffers.push({ canvas, log })
    return canvas
  })
})

// Был ли вызов в каком-нибудь буфере слоя.
const inBuffers = (predicate) => buffers.some((b) => b.log.some(predicate))

describe('рендер слоёв', () => {
  it('обычный слой рисуется прямо на холст, без буферов', () => {
    const log = []
    const target = surface(log)
    renderScene(target.getContext('2d'), {
      layers: [{ id: 'l1' }],
      objects: [stroke('a', 'l1')],
    }, { width: 200, height: 100, camera, images: new Map(), background: false })

    // Слой без прозрачности, маски и стирания сводить не с чем.
    expect(log.some((e) => e.op === 'stroke')).toBe(true)
    expect(log.some((e) => e.op === 'drawImage')).toBe(false)
  })

  it('стирающий штрих уводит слой в буфер и режет по destination-out', () => {
    const log = []
    const target = surface(log)
    renderScene(target.getContext('2d'), {
      layers: [{ id: 'l1' }],
      objects: [stroke('a', 'l1'), stroke('e', 'l1', { erase: true, width: 20 })],
    }, { width: 200, height: 100, camera, images: new Map(), background: false })

    expect(inBuffers((e) => e.op === 'set:globalCompositeOperation' && e.value === 'destination-out')).toBe(true)
    // Готовый буфер сводится на холст одним drawImage.
    expect(log.filter((e) => e.op === 'drawImage')).toHaveLength(1)
  })

  it('прозрачность и наложение применяются при сведении слоя', () => {
    const log = []
    const target = surface(log)
    renderScene(target.getContext('2d'), {
      layers: [{ id: 'l1' }, { id: 'l2', opacity: 0.5, blend: 'multiply' }],
      objects: [stroke('a', 'l1'), stroke('b', 'l2')],
    }, { width: 200, height: 100, camera, images: new Map(), background: false })

    const compose = log.find((e) => e.op === 'drawImage')
    expect(compose.alpha).toBe(0.5)
    expect(compose.blend).toBe('multiply')
  })

  it('обтравочный слой обрезается по альфе слоя под ним', () => {
    const log = []
    const target = surface(log)
    renderScene(target.getContext('2d'), {
      layers: [{ id: 'base' }, { id: 'clip', clip: true }],
      objects: [stroke('a', 'base'), stroke('b', 'clip')],
    }, { width: 200, height: 100, camera, images: new Map(), background: false })

    // Обтравка живёт в буфере обтравленного слоя: он «оставляет себя» по базе.
    expect(inBuffers((e) => e.op === 'set:globalCompositeOperation' && e.value === 'destination-in')).toBe(true)
    // На холст группа уходит одним изображением, а не двумя слоями подряд.
    expect(log.filter((e) => e.op === 'drawImage')).toHaveLength(1)
  })

  it('маска слоя обрезает содержимое по многоугольнику', () => {
    const log = []
    const target = surface(log)
    renderScene(target.getContext('2d'), {
      layers: [{ id: 'l1', mask: { points: [0, 0, 50, 0, 50, 50] } }],
      objects: [stroke('a', 'l1')],
    }, { width: 200, height: 100, camera, images: new Map(), background: false })

    expect(inBuffers((e) => e.op === 'clip')).toBe(true)
  })
})

describe('булевы операции и маски', () => {
  const shape = (id, extra = {}) => ({
    id, type: OBJ.rect, layer: 'l1', x: 0, y: 0, w: 50, h: 50, color: 'ink', ...extra,
  })

  it('участники булевой группы складываются композитом в одном буфере', () => {
    const log = []
    const target = surface(log)
    renderScene(target.getContext('2d'), {
      layers: [{ id: 'l1' }],
      objects: [
        shape('a', { bool: 'g1', boolOp: 'union' }),
        shape('b', { bool: 'g1', boolOp: 'subtract', x: 25 }),
      ],
    }, { width: 200, height: 100, camera, images: new Map(), background: false })

    expect(inBuffers((e) => e.op === 'set:globalCompositeOperation' && e.value === 'destination-out')).toBe(true)
    // Группа сводится на холст ОДНИМ изображением — это одна фигура.
    expect(log.filter((e) => e.op === 'drawImage')).toHaveLength(1)
  })

  it('исключение идёт режимом xor', () => {
    const log = []
    const target = surface(log)
    renderScene(target.getContext('2d'), {
      layers: [{ id: 'l1' }],
      objects: [
        shape('a', { bool: 'g1', boolOp: 'union' }),
        shape('b', { bool: 'g1', boolOp: 'exclude', x: 25 }),
      ],
    }, { width: 200, height: 100, camera, images: new Map(), background: false })

    expect(inBuffers((e) => e.op === 'set:globalCompositeOperation' && e.value === 'xor')).toBe(true)
  })

  it('объект-маска обрезает то, что лежит над ней', () => {
    const log = []
    const target = surface(log)
    renderScene(target.getContext('2d'), {
      layers: [{ id: 'l1' }],
      objects: [shape('m', { maskObject: true }), shape('a', { x: 20 })],
    }, { width: 200, height: 100, camera, images: new Map(), background: false })

    expect(inBuffers((e) => e.op === 'set:globalCompositeOperation' && e.value === 'destination-in')).toBe(true)
  })

  it('скрытый объект не рисуется вовсе', () => {
    const log = []
    const target = surface(log)
    renderScene(target.getContext('2d'), {
      layers: [{ id: 'l1' }],
      objects: [shape('a'), shape('b', { hidden: true, x: 60 })],
    }, { width: 200, height: 100, camera, images: new Map(), background: false })

    // Один прямоугольник — один путь и одна обводка.
    expect(log.filter((e) => e.op === 'stroke')).toHaveLength(1)
  })
})

describe('кадры и калька', () => {
  const scene = {
    animation: { fps: 12, frames: [{ id: 'f1' }, { id: 'f2' }] },
    layers: [{ id: 'l1' }],
    objects: [
      stroke('a', 'l1', { frame: 'f1' }),
      stroke('b', 'l1', { frame: 'f2' }),
      stroke('c', 'l1'),
    ],
  }

  it('рисует только свой кадр и общие объекты', () => {
    const log = []
    const target = surface(log)
    renderScene(target.getContext('2d'), scene, {
      width: 200, height: 100, camera, images: new Map(), background: false, frame: 'f1',
    })
    // Два штриха кадра «f1»: собственный и общий.
    expect(log.filter((e) => e.op === 'stroke')).toHaveLength(2)
  })

  it('призрачный кадр сводится с общей прозрачностью, а не с прозрачностью объектов', () => {
    const log = []
    const target = surface(log)
    renderScene(target.getContext('2d'), scene, {
      width: 200,
      height: 100,
      camera,
      images: new Map(),
      background: false,
      frame: 'f2',
      onion: [{ frame: 'f1', alpha: 0.35 }],
    })

    const ghost = log.find((e) => e.op === 'drawImage')
    expect(ghost.alpha).toBeCloseTo(0.35)
  })
})
