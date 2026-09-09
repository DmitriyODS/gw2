import { describe, expect, it } from 'vitest'
import {
  insertNode, moveNode, newNode, nodeHit, outlineMetrics, outlinePoints, pointAtLength,
  removeNode, scaleNodes, shapeToVector, toggleNodeSmooth, vectorOutline,
} from './boardVector.js'

const line = {
  type: 'vector',
  closed: false,
  nodes: [newNode(0, 0), newNode(100, 0)],
}

describe('узлы контура', () => {
  it('новый узел не имеет оттянутых направляющих', () => {
    const n = newNode(10, 20)
    expect(n).toEqual({ x: 10, y: 20, ix: 10, iy: 20, ox: 10, oy: 20 })
  })

  it('прямые сегменты не дробятся на кривые', () => {
    // Две точки без направляющих дают ровно две точки полилинии.
    expect(vectorOutline(line)).toEqual([0, 0, 100, 0])
  })

  it('кривая раскладывается в полилинию', () => {
    const curved = { ...line, nodes: [{ ...line.nodes[0], ox: 20, oy: -40 }, line.nodes[1]] }
    const points = vectorOutline(curved, 8)
    expect(points.length).toBeGreaterThan(4)
    // Кривая уходит вверх от прямой линии между концами.
    expect(Math.min(...points.filter((_, i) => i % 2))).toBeLessThan(0)
  })

  it('перенос узла тянет за собой обе направляющие', () => {
    const withHandles = { ...line, nodes: [{ x: 0, y: 0, ix: -10, iy: 0, ox: 10, oy: 0 }, line.nodes[1]] }
    const moved = moveNode(withHandles, 0, 'node', 5, 5)
    expect(moved.nodes[0]).toEqual({ x: 5, y: 5, ix: -5, iy: 5, ox: 15, oy: 5 })
  })

  it('направляющая по умолчанию ведёт вторую зеркально, с Alt — нет', () => {
    const mirrored = moveNode(line, 0, 'out', 10, 10)
    expect(mirrored.nodes[0].ix).toBe(-10)
    expect(mirrored.nodes[0].iy).toBe(-10)

    const broken = moveNode(line, 0, 'out', 10, 10, { mirror: false })
    expect(broken.nodes[0].ix).toBe(0)
  })

  it('узел добавляется на сегменте и удаляется, пока их больше двух', () => {
    const added = insertNode(line, 50, 0)
    expect(added.nodes).toHaveLength(3)
    expect(removeNode(added, 1).nodes).toHaveLength(2)
    // Двухузловой контур не разбираем — иначе он перестанет быть линией.
    expect(removeNode(line, 0).nodes).toHaveLength(2)
  })

  it('переключение сглаживания добавляет и убирает направляющие', () => {
    const three = insertNode(line, 50, 0)
    const smooth = toggleNodeSmooth(three, 1)
    expect(smooth.nodes[1].ox).not.toBe(smooth.nodes[1].x)
    expect(toggleNodeSmooth(smooth, 1).nodes[1].ox).toBe(smooth.nodes[1].x)
  })

  it('попадание различает узел и его направляющую', () => {
    const withHandles = { ...line, nodes: [{ x: 0, y: 0, ix: -10, iy: 0, ox: 10, oy: 0 }, line.nodes[1]] }
    expect(nodeHit(withHandles, 0, 0, 5)).toEqual({ index: 0, part: 'node' })
    expect(nodeHit(withHandles, 10, 0, 5)).toEqual({ index: 0, part: 'out' })
    expect(nodeHit(withHandles, 400, 400, 5)).toBeNull()
  })

  it('масштабирование двигает и узлы, и направляющие', () => {
    const scaled = scaleNodes([{ x: 10, y: 10, ix: 5, iy: 10, ox: 15, oy: 10 }], (x) => x * 2, (y) => y)
    expect(scaled[0]).toEqual({ x: 20, y: 10, ix: 10, iy: 10, ox: 30, oy: 10 })
  })
})

describe('контуры фигур', () => {
  it('овал раскладывается в замкнутую полилинию', () => {
    const points = outlinePoints({ type: 'ellipse', x: 0, y: 0, w: 100, h: 50 })
    expect(points.length).toBe(96)
  })

  it('фигура превращается в кривые с теми же вершинами', () => {
    const vector = shapeToVector({ id: 'a', type: 'diamond', x: 0, y: 0, w: 100, h: 100, color: 'red' })
    expect(vector.type).toBe('vector')
    expect(vector.closed).toBe(true)
    expect(vector.nodes).toHaveLength(4)
    expect(vector.color).toBe('red')
  })
})

describe('длина контура', () => {
  const points = [0, 0, 100, 0, 100, 100]

  it('считает полную длину по сегментам', () => {
    expect(outlineMetrics(points).total).toBe(200)
  })

  it('точка на длине знает свой наклон', () => {
    const metrics = outlineMetrics(points)
    expect(pointAtLength(points, metrics, 50)).toMatchObject({ x: 50, y: 0, angle: 0 })
    const corner = pointAtLength(points, metrics, 150)
    expect(corner.x).toBe(100)
    expect(corner.y).toBe(50)
  })
})
