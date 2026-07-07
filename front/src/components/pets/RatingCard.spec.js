import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import RatingCard from './RatingCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'

function factory(rating, myUserId = 7) {
  const pinia = createTestingPinia({
    createSpy: vi.fn,
    initialState: {
      pets: { rating },
      auth: { user: { id: myUserId } },
    },
  })
  return mount(RatingCard, { global: { plugins: [pinia] } })
}

const mkRow = (pos, id, over = {}) => ({
  position: pos, xp: 100 - pos, kudos_week: 50 - pos, pet_name: `Pet${id}`, stage: 3, species: 'fox',
  user: { id, fio: `User${id}` }, ...over,
})

describe('RatingCard', () => {
  it('пустой рейтинг — карточка с EmptyState вместо списка', () => {
    const w = factory(null)
    expect(w.find('.rating-card').exists()).toBe(true)
    expect(w.findComponent(EmptyState).exists()).toBe(true)
    expect(w.findAll('.rating-row')).toHaveLength(0)
  })

  it('показывает топ-10 строк', () => {
    const items = Array.from({ length: 12 }, (_, i) => mkRow(i + 1, i + 1))
    const w = factory({ items, total: 12, me: null })
    expect(w.findAll('.rating-row')).toHaveLength(10)
  })

  it('подсвечивает собственную строку (mine)', () => {
    const items = [mkRow(1, 7), mkRow(2, 8)] // id=7 — это я
    const w = factory({ items, total: 2, me: null }, 7)
    const mine = w.findAll('.rating-row.mine')
    expect(mine).toHaveLength(1)
    expect(mine[0].text()).toContain('User7')
  })

  it('добавляет мою строку с разрывом, если я вне топ-10', () => {
    const items = Array.from({ length: 10 }, (_, i) => mkRow(i + 1, i + 1))
    const me = mkRow(15, 77) // position 15 → gapBefore (15 > 11)
    const w = factory({ items, total: 20, me }, 77)
    const rows = w.findAll('.rating-row')
    expect(rows).toHaveLength(11)
    expect(w.find('.rating-row.gap').exists()).toBe(true)
    expect(w.find('.rating-row.mine').exists()).toBe(true)
  })

  it('число и бар — по кудосам недели', () => {
    const items = [mkRow(1, 1, { kudos_week: 8 }), mkRow(2, 2, { kudos_week: 4 })]
    const w = factory({ items, total: 2, me: null })
    const kudos = w.findAll('.rating-kudos')
    expect(kudos[0].text()).toContain('8')
    expect(kudos[1].text()).toContain('4')
    const fills = w.findAll('.rating-fill')
    expect(fills[0].attributes('style')).toContain('width: 100%')
    expect(fills[1].attributes('style')).toContain('width: 50%')
  })

  it('нулевые кудосы недели показываются как 0', () => {
    const w = factory({ items: [mkRow(1, 1, { kudos_week: 0 })], total: 1, me: null })
    expect(w.find('.rating-kudos').text()).toContain('0')
  })
})
