import { describe, it, expect } from 'vitest'
import { TASK_COLORS, TASK_COLOR_IDS, cardColorStyle } from './taskColors.js'

describe('taskColors', () => {
  it('ровно 8 цветов, id уникальны, каждый с подписью', () => {
    expect(TASK_COLORS).toHaveLength(8)
    expect(new Set(TASK_COLOR_IDS).size).toBe(8)
    for (const c of TASK_COLORS) expect(c.label).toBeTruthy()
  })

  it('TASK_COLOR_IDS соответствует набору id (паритет с бэком domain.TaskColors)', () => {
    expect(TASK_COLOR_IDS).toEqual(['red', 'orange', 'amber', 'green', 'teal', 'blue', 'violet', 'pink'])
  })

  it('cardColorStyle подставляет токены выбранного цвета', () => {
    expect(cardColorStyle('teal')).toEqual({
      '--card-tag-surface': 'var(--tag-teal-surface)',
      '--card-tag-border': 'var(--tag-teal-border)',
      '--card-tag-accent': 'var(--tag-teal-accent)',
    })
  })

  it('неизвестный/пустой цвет → пустой стиль (карточка без окраски)', () => {
    expect(cardColorStyle(null)).toEqual({})
    expect(cardColorStyle('')).toEqual({})
    expect(cardColorStyle('rainbow')).toEqual({})
  })
})
