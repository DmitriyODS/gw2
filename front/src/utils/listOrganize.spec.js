import { describe, expect, it } from 'vitest'
import { flattenGroups, moveKey, organizeList, shiftKey } from './listOrganize.js'

const items = [
  { id: 1, name: 'Личное' },
  { id: 2, name: 'Работа' },
  { id: 3, name: 'Учёба' },
]

describe('organizeList', () => {
  it('без настроек отдаёт один список в порядке сервера', () => {
    const groups = organizeList(items, {})
    expect(groups).toHaveLength(1)
    expect(groups[0].kind).toBe('loose')
    expect(flattenGroups(groups)).toEqual(['1', '2', '3'])
  })

  it('закреплённые идут первой группой и не попадают в папку', () => {
    const groups = organizeList(items, {
      folders: [{ id: 'f1', name: 'Дом' }],
      pinned: ['3'],
      folderOf: { 3: 'f1', 2: 'f1' },
    })
    expect(groups.map((g) => g.kind)).toEqual(['pinned', 'folder', 'loose'])
    expect(flattenGroups(groups)).toEqual(['3', '2', '1'])
  })

  it('ручной порядок важнее серверного, новые записи идут следом', () => {
    const groups = organizeList(items, { order: ['3', '1'] })
    expect(flattenGroups(groups)).toEqual(['3', '1', '2'])
  })

  it('привязка к удалённой папке возвращает запись в общий список', () => {
    const groups = organizeList(items, { folders: [], folderOf: { 2: 'gone' } })
    expect(groups).toHaveLength(1)
    expect(groups[0].kind).toBe('loose')
  })

  it('пустая папка остаётся в списке — в неё перетаскивают', () => {
    const groups = organizeList(items, { folders: [{ id: 'f1', name: 'Дом' }] })
    expect(groups[0]).toMatchObject({ kind: 'folder', id: 'f1', items: [] })
  })

  it('сворачивается любая группа, а не только папка', () => {
    const groups = organizeList(items, { pinned: ['1'], collapsed: { pinned: true, loose: true } })
    expect(groups.map((g) => [g.collapseKey, g.collapsed])).toEqual([['pinned', true], ['loose', true]])
  })
})

describe('moveKey', () => {
  it('вставляет перед целью и после неё', () => {
    expect(moveKey(['1', '2', '3'], '3', '1', true)).toEqual(['3', '1', '2'])
    expect(moveKey(['1', '2', '3'], '1', '3', false)).toEqual(['2', '3', '1'])
  })

  it('незнакомый ключ просто добавляется в конец', () => {
    expect(moveKey(['1', '2'], '9', '', true)).toEqual(['1', '2', '9'])
  })
})

describe('shiftKey', () => {
  const order = ['1', '2', '3', '4']

  it('двигает внутри своей группы, а не по всему списку', () => {
    // Группа — только 2 и 4: «выше» ставит 4 перед 2, минуя чужие 1 и 3.
    expect(shiftKey(order, '4', -1, ['2', '4'])).toEqual(['1', '4', '2', '3'])
  })

  it('на краю группы ничего не меняет', () => {
    expect(shiftKey(order, '2', -1, ['2', '4'])).toBe(order)
  })
})
