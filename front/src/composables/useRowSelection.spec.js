import { describe, it, expect } from 'vitest'
import { nextTick, ref } from 'vue'
import { useRowSelection } from './useRowSelection.js'

const rows = (...ids) => ids.map((id) => ({ id }))

describe('useRowSelection', () => {
  it('выбор переживает смену страницы', () => {
    const page = ref(rows(1, 2, 3))
    const sel = useRowSelection(() => page.value, { total: () => 6 })

    sel.toggle(1)
    sel.toggle(2)
    page.value = rows(4, 5, 6) // ушли на вторую страницу
    expect(sel.count.value).toBe(2)
    expect(sel.payload.value).toEqual({ ids: [1, 2] })

    sel.toggle(4)
    expect(sel.count.value).toBe(3)
  })

  it('галочка в шапке — только про текущую страницу', () => {
    const page = ref(rows(1, 2))
    const sel = useRowSelection(() => page.value, { total: () => 5 })

    sel.toggleAll()
    expect(sel.count.value).toBe(2)
    expect(sel.allSelected.value).toBe(true)

    page.value = rows(3, 4)
    expect(sel.allSelected.value).toBe(false)
    sel.toggleAll()
    expect(sel.count.value).toBe(4)
  })

  it('«выбрать всё» описывается фильтром, а не списком id', () => {
    const page = ref(rows(1, 2))
    const sel = useRowSelection(() => page.value, { total: () => 500 })

    sel.selectAllMatching()
    expect(sel.count.value).toBe(500)
    expect(sel.payload.value).toEqual({ all: true, exclude: [] })
    expect(sel.isSelected(999)).toBe(true) // и записи, которых на экране нет

    // Снятая галочка становится исключением, а не превращает выбор в перечень.
    sel.toggle(2)
    expect(sel.count.value).toBe(499)
    expect(sel.isSelected(2)).toBe(false)
    expect(sel.payload.value).toEqual({ all: true, exclude: [2] })
  })

  it('в режиме «все» галочка шапки снимает страницу целиком', () => {
    const page = ref(rows(1, 2))
    const sel = useRowSelection(() => page.value, { total: () => 10 })

    sel.selectAllMatching()
    sel.toggleAll()
    expect(sel.count.value).toBe(8)
    expect(sel.payload.value).toEqual({ all: true, exclude: [1, 2] })
  })

  it('смена выборки (реестр, поиск, тег) сбрасывает выбор', async () => {
    const page = ref(rows(1, 2))
    const scope = ref('reg1||')
    const sel = useRowSelection(() => page.value, { total: () => 9, scope: () => scope.value })

    sel.selectAllMatching()
    expect(sel.count.value).toBe(9)

    scope.value = 'reg1|болт|'
    await nextTick()
    expect(sel.count.value).toBe(0)
    expect(sel.payload.value).toEqual({ ids: [] })
  })
})
