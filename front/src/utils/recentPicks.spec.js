import { beforeEach, describe, expect, it } from 'vitest'
import { pickScore, rememberPick, splitByFrequency } from './recentPicks.js'

const DEPTS = [
  { id: 1, name: 'Бухгалтерия' },
  { id: 2, name: 'Автоматизация' },
  { id: 3, name: 'Снабжение' },
]

describe('recentPicks', () => {
  beforeEach(() => localStorage.clear())

  it('без истории список остаётся алфавитным', () => {
    const { frequent, rest } = splitByFrequency(DEPTS, { kind: 'department', scope: 1 })
    expect(frequent).toEqual([])
    expect(rest.map((d) => d.name)).toEqual(['Автоматизация', 'Бухгалтерия', 'Снабжение'])
  })

  it('часто выбираемое поднимается наверх и не дублируется в остальных', () => {
    rememberPick('department', 1, 3)
    rememberPick('department', 1, 3)
    rememberPick('department', 1, 2)

    const { frequent, rest } = splitByFrequency(DEPTS, { kind: 'department', scope: 1 })
    expect(frequent.map((d) => d.id)).toEqual([3, 2])
    expect(rest.map((d) => d.id)).toEqual([1])
  })

  it('счётчик скоупится компанией', () => {
    rememberPick('department', 1, 3)
    expect(pickScore('department', 1, 3)).toBeGreaterThan(0)
    expect(pickScore('department', 2, 3)).toBe(0)
  })

  it('лимит подсказок не превышается', () => {
    const many = Array.from({ length: 12 }, (_, i) => ({ id: i + 1, name: `Отдел ${i + 1}` }))
    many.forEach((d) => rememberPick('department', 1, d.id))
    const { frequent } = splitByFrequency(many, { kind: 'department', scope: 1, limit: 5 })
    expect(frequent).toHaveLength(5)
  })
})
