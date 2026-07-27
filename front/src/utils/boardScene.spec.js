import { describe, expect, it } from 'vitest'
import {
  OBJ, editableLayerIds, hitTest, moveObject, normalizeScene, objectBounds,
  orderedObjects, scaleObject, sceneBounds, sceneText,
} from './boardScene.js'

describe('normalizeScene', () => {
  it('чинит битую сцену вместо падения', () => {
    const fixed = normalizeScene(null)
    expect(fixed.objects).toEqual([])
    expect(fixed.background).toBe('grid')
    // Слой всегда есть — рисовать без него некуда.
    expect(fixed.layers).toHaveLength(1)
    expect(normalizeScene({ objects: 'нет' }).objects).toEqual([])
    // Неизвестный фон откатывается к сетке — иначе холст остался бы пустым.
    expect(normalizeScene({ background: 'радуга' }).background).toBe('grid')
  })

  it('поднимает сцену без слоёв: объекты уезжают в базовый слой', () => {
    const scene = normalizeScene({ objects: [{ id: 'a', type: OBJ.rect, x: 0, y: 0, w: 10, h: 10 }] })
    expect(scene.objects[0].layer).toBe(scene.layers[0].id)
  })

  it('объект с несуществующим слоем не теряется', () => {
    const scene = normalizeScene({
      layers: [{ id: 'l1', name: 'Низ' }],
      objects: [{ id: 'a', type: OBJ.rect, layer: 'нет-такого' }],
    })
    expect(scene.objects[0].layer).toBe('l1')
  })

  it('выбрасывает объекты без типа', () => {
    const scene = normalizeScene({ objects: [{ id: 'a' }, { id: 'b', type: OBJ.rect }] })
    expect(scene.objects).toHaveLength(1)
  })
})

describe('objectBounds', () => {
  it('считает рамку свободного пера по точкам', () => {
    const b = objectBounds({ type: OBJ.path, points: [10, 10, 40, 30, 20, 50] })
    expect(b).toEqual({ x: 10, y: 10, w: 30, h: 40 })
  })

  it('нормализует линию, нарисованную справа налево', () => {
    const b = objectBounds({ type: OBJ.line, x: 100, y: 80, x2: 20, y2: 10 })
    expect(b).toEqual({ x: 20, y: 10, w: 80, h: 70 })
  })
})

describe('sceneBounds', () => {
  it('объединяет рамки всех объектов', () => {
    const b = sceneBounds([
      { type: OBJ.rect, x: 0, y: 0, w: 50, h: 50 },
      { type: OBJ.rect, x: 100, y: 20, w: 40, h: 10 },
    ])
    expect(b).toEqual({ x: 0, y: 0, w: 140, h: 50 })
  })

  it('пустая сцена рамки не имеет', () => {
    expect(sceneBounds([])).toBeNull()
  })
})

describe('hitTest', () => {
  const rect = { type: OBJ.rect, x: 0, y: 0, w: 100, h: 50 }

  it('ловит клик внутри фигуры и мимо неё', () => {
    expect(hitTest(rect, 50, 25)).toBe(true)
    expect(hitTest(rect, 300, 300)).toBe(false)
  })

  it('по линии попадает с допуском на толщину', () => {
    const line = { type: OBJ.line, x: 0, y: 0, x2: 100, y2: 0, width: 4 }
    expect(hitTest(line, 50, 3)).toBe(true)
    expect(hitTest(line, 50, 60)).toBe(false)
  })
})

describe('moveObject', () => {
  it('сдвигает перо целиком по точкам', () => {
    const moved = moveObject({ type: OBJ.path, points: [0, 0, 10, 10] }, 5, -5)
    expect(moved.points).toEqual([5, -5, 15, 5])
  })

  it('сдвигает оба конца стрелки', () => {
    const moved = moveObject({ type: OBJ.arrow, x: 0, y: 0, x2: 10, y2: 10 }, 3, 4)
    expect([moved.x, moved.y, moved.x2, moved.y2]).toEqual([3, 4, 13, 14])
  })
})

describe('scaleObject', () => {
  it('растягивает фигуру в новую рамку', () => {
    const from = { x: 0, y: 0, w: 100, h: 100 }
    const to = { x: 0, y: 0, w: 200, h: 50 }
    const scaled = scaleObject({ type: OBJ.rect, x: 50, y: 50, w: 50, h: 50 }, from, to)
    expect(scaled.x).toBe(100)
    expect(scaled.y).toBe(25)
    expect(scaled.w).toBe(100)
    expect(scaled.h).toBe(25)
  })

  it('надписи меняют кегль, а не ширину', () => {
    const scaled = scaleObject(
      { type: OBJ.text, x: 0, y: 100, size: 20, text: 'привет' },
      { x: 0, y: 0, w: 100, h: 100 },
      { x: 0, y: 0, w: 100, h: 200 },
    )
    expect(scaled.size).toBe(40)
  })
})

describe('слои', () => {
  const scene = {
    layers: [
      { id: 'l1', name: 'Низ', visible: true, locked: false },
      { id: 'l2', name: 'Верх', visible: false, locked: false },
      { id: 'l3', name: 'Замок', visible: true, locked: true },
    ],
    objects: [
      { id: 'b', type: OBJ.rect, layer: 'l2' },
      { id: 'a', type: OBJ.rect, layer: 'l1' },
      { id: 'c', type: OBJ.rect, layer: 'l3' },
    ],
  }

  it('рисует снизу вверх и пропускает скрытые слои', () => {
    expect(orderedObjects(scene).map((o) => o.id)).toEqual(['a', 'c'])
  })

  it('правке доступны только видимые и незаблокированные слои', () => {
    const ids = editableLayerIds(scene)
    expect([...ids]).toEqual(['l1'])
  })
})

describe('sceneText', () => {
  it('собирает надписи и стикеры для поиска', () => {
    const text = sceneText({
      objects: [
        { type: OBJ.text, text: 'План' },
        { type: OBJ.path, points: [0, 0, 1, 1] },
        { type: OBJ.sticky, text: 'Созвон' },
      ],
    })
    expect(text).toBe('План\nСозвон')
  })
})
