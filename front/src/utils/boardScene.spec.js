import { describe, expect, it } from 'vitest'
import {
  OBJ, boolMembers, editableLayerIds, hitTest, isHexColor, isObjectEditable, moveObject,
  normalizeScene, objectAABB, objectBounds, objectLabel, objectsForFrame, orderedObjects,
  pointInPolygon, resolveColor, rotateObject, scaleObject, sceneBounds, sceneText,
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

describe('произвольный цвет', () => {
  it('узнаёт #rrggbb и #rgb, ключи палитры цветом не считает', () => {
    expect(isHexColor('#a1b2c3')).toBe(true)
    expect(isHexColor('#ABC')).toBe(true)
    expect(isHexColor('red')).toBe(false)
    expect(isHexColor('#zzz')).toBe(false)
  })

  it('отдаёт значение как есть — теме оно не следует', () => {
    expect(resolveColor('#A1B2C3')).toBe('#A1B2C3')
    // Белый задан значением: им рисуют и в тёмной теме.
    expect(resolveColor('white')).toBe('#ffffff')
  })
})

describe('слои', () => {
  it('достраивает свойства слоёв прежних сцен', () => {
    const scene = normalizeScene({ layers: [{ id: 'l1', name: 'Низ' }], objects: [] })
    expect(scene.layers[0]).toMatchObject({ opacity: 1, blend: 'normal', clip: false, mask: null })
  })

  it('снимает обтравку с нижнего слоя — ей не по чему работать', () => {
    const scene = normalizeScene({ layers: [{ id: 'l1', clip: true }], objects: [] })
    expect(scene.layers[0].clip).toBe(false)
  })

  it('выбрасывает выдуманный режим наложения и куцую маску', () => {
    const scene = normalizeScene({
      layers: [{ id: 'l1' }, { id: 'l2', blend: 'радуга', mask: { points: [0, 0] } }],
      objects: [],
    })
    expect(scene.layers[1].blend).toBe('normal')
    expect(scene.layers[1].mask).toBeNull()
  })
})

describe('кадры анимации', () => {
  const scene = {
    animation: { fps: 12, frames: [{ id: 'f1' }, { id: 'f2' }] },
    layers: [{ id: 'l1' }],
    objects: [
      { id: 'a', type: OBJ.rect, layer: 'l1', frame: 'f1' },
      { id: 'b', type: OBJ.rect, layer: 'l1', frame: 'f2' },
      { id: 'c', type: OBJ.rect, layer: 'l1' },
    ],
  }

  it('в кадре видны его объекты и общие', () => {
    expect(orderedObjects(scene, { frame: 'f1' }).map((o) => o.id)).toEqual(['a', 'c'])
    expect(objectsForFrame(scene, 'f2').map((o) => o.id)).toEqual(['b', 'c'])
  })

  it('без кадров анимации нет вовсе', () => {
    expect(normalizeScene({ animation: { fps: 12, frames: [] }, objects: [] }).animation).toBeNull()
  })

  it('объект удалённого кадра становится общим, а не пропадает', () => {
    const fixed = normalizeScene({
      animation: { fps: 12, frames: [{ id: 'f1' }] },
      objects: [{ id: 'a', type: OBJ.rect, frame: 'нет-такого' }],
    })
    expect(fixed.objects).toHaveLength(1)
    expect(fixed.objects[0].frame).toBeUndefined()
  })

  it('частота держится в разумных границах', () => {
    const fast = normalizeScene({ animation: { fps: 999, frames: [{ id: 'f1' }] }, objects: [] })
    expect(fast.animation.fps).toBe(60)
  })
})

describe('поворот', () => {
  const rect = { type: OBJ.rect, x: 0, y: 0, w: 100, h: 20, angle: 90 }

  it('габарит считается по повёрнутым углам', () => {
    const b = objectAABB(rect)
    expect(Math.round(b.w)).toBe(20)
    expect(Math.round(b.h)).toBe(100)
  })

  it('клик попадает по объекту в его повёрнутом виде', () => {
    // После поворота на 90° полоса стоит вертикально: точка ниже центра — в ней.
    expect(hitTest(rect, 50, 50)).toBe(true)
    expect(hitTest(rect, 95, 10)).toBe(false)
  })

  it('поворот группы крутит объекты вокруг общего центра', () => {
    const moved = rotateObject({ type: OBJ.rect, x: 100, y: 0, w: 10, h: 10 }, 180, { x: 0, y: 0 })
    expect(Math.round(moved.x)).toBe(-110)
    expect(moved.angle).toBe(180)
  })
})

describe('лассо', () => {
  it('точка внутри и снаружи контура', () => {
    const square = [0, 0, 100, 0, 100, 100, 0, 100]
    expect(pointInPolygon(square, 50, 50)).toBe(true)
    expect(pointInPolygon(square, 150, 50)).toBe(false)
  })
})

describe('дерево слоёв', () => {
  it('скрытые объекты не идут в отрисовку, но остаются в сцене', () => {
    const scene = {
      layers: [{ id: 'l1' }],
      objects: [
        { id: 'a', type: OBJ.rect, layer: 'l1' },
        { id: 'b', type: OBJ.rect, layer: 'l1', hidden: true },
      ],
    }
    expect(orderedObjects(scene).map((o) => o.id)).toEqual(['a'])
    expect(normalizeScene(scene).objects).toHaveLength(2)
  })

  it('запертый и скрытый объект не редактируется', () => {
    const layers = new Set(['l1'])
    expect(isObjectEditable({ layer: 'l1' }, layers)).toBe(true)
    expect(isObjectEditable({ layer: 'l1', locked: true }, layers)).toBe(false)
    expect(isObjectEditable({ layer: 'l1', hidden: true }, layers)).toBe(false)
    expect(isObjectEditable({ layer: 'нет' }, layers)).toBe(false)
  })

  it('подпись строки — свой текст, иначе название типа', () => {
    expect(objectLabel({ type: OBJ.text, text: 'План\nвторая строка' })).toBe('План')
    expect(objectLabel({ type: OBJ.ellipse })).toBe('Овал')
    expect(objectLabel({ type: OBJ.polygon, star: true })).toBe('Звезда')
    expect(objectLabel({ type: OBJ.path, erase: true })).toBe('Ластик')
  })
})

describe('булевы группы', () => {
  it('группа из одного участника распадается', () => {
    const scene = normalizeScene({
      objects: [{ id: 'a', type: OBJ.rect, bool: 'g1', boolOp: 'subtract' }],
    })
    expect(scene.objects[0].bool).toBeUndefined()
    expect(scene.objects[0].boolOp).toBeUndefined()
  })

  it('выдуманная операция становится объединением', () => {
    const scene = normalizeScene({
      objects: [
        { id: 'a', type: OBJ.rect, bool: 'g1', boolOp: 'union' },
        { id: 'b', type: OBJ.rect, bool: 'g1', boolOp: 'вычесть-как-нибудь' },
      ],
    })
    expect(scene.objects[1].boolOp).toBe('union')
  })

  it('участники группы собираются в порядке отрисовки', () => {
    const objects = [
      { id: 'a', type: OBJ.rect, bool: 'g1' },
      { id: 'x', type: OBJ.rect },
      { id: 'b', type: OBJ.rect, bool: 'g1' },
    ]
    expect(boolMembers(objects, 'g1').map((o) => o.id)).toEqual(['a', 'b'])
  })
})

describe('градиент и эффекты в сцене', () => {
  it('битый градиент и пустые эффекты не сохраняются', () => {
    const scene = normalizeScene({
      objects: [{ id: 'a', type: OBJ.rect, gradient: { stops: [] }, effects: { blur: 0 } }],
    })
    expect(scene.objects[0].gradient).toBeUndefined()
    expect(scene.objects[0].effects).toBeUndefined()
  })

  it('годный градиент остаётся с отсортированными точками', () => {
    const scene = normalizeScene({
      objects: [{
        id: 'a',
        type: OBJ.rect,
        gradient: { type: 'radial', stops: [{ color: 'red', at: 1 }, { color: 'blue', at: 0 }] },
      }],
    })
    expect(scene.objects[0].gradient.type).toBe('radial')
    expect(scene.objects[0].gradient.stops[0].color).toBe('blue')
  })
})
