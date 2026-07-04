import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import RatingCard from './RatingCard.vue'

function factory(rating, myUserId = 7) {
  const pinia = createTestingPinia({
    createSpy: vi.fn,
    initialState: {
      groove: { rating },
      auth: { user: { id: myUserId } },
    },
  })
  return mount(RatingCard, { global: { plugins: [pinia] } })
}

const mkRow = (pos, id, over = {}) => ({
  position: pos, xp: 100 - pos, pet_name: `Pet${id}`, stage: 3, species: 'fox',
  user: { id, fio: `User${id}` }, ...over,
})

describe('RatingCard', () => {
  it('пустой рейтинг — секция не рендерится', () => {
    const w = factory(null)
    expect(w.find('.rating-card').exists()).toBe(false)
  })

  it('показывает топ-5 строк', () => {
    const items = [1, 2, 3, 4, 5, 6].map((p) => mkRow(p, p))
    const w = factory({ items, total: 6, me: null })
    expect(w.findAll('.rating-row')).toHaveLength(5)
  })

  it('подсвечивает собственную строку (mine)', () => {
    const items = [mkRow(1, 7), mkRow(2, 8)] // id=7 — это я
    const w = factory({ items, total: 2, me: null }, 7)
    const mine = w.findAll('.rating-row.mine')
    expect(mine).toHaveLength(1)
    expect(mine[0].text()).toContain('User7')
  })

  it('добавляет мою строку с разрывом, если я вне топ-5', () => {
    const items = [1, 2, 3, 4, 5].map((p) => mkRow(p, p))
    const me = mkRow(9, 7) // position 9 → gapBefore (9 > 6)
    const w = factory({ items, total: 20, me }, 7)
    const rows = w.findAll('.rating-row')
    expect(rows).toHaveLength(6)
    expect(w.find('.rating-row.gap').exists()).toBe(true)
    expect(w.find('.rating-row.mine').exists()).toBe(true)
  })

  it('kudos_week показывается только при значении > 0', () => {
    const withKudos = factory({ items: [mkRow(1, 1, { kudos_week: 3 })], total: 1, me: null })
    expect(withKudos.find('.rating-kudos').exists()).toBe(true)

    const zeroKudos = factory({ items: [mkRow(1, 1, { kudos_week: 0 })], total: 1, me: null })
    expect(zeroKudos.find('.rating-kudos').exists()).toBe(false)
  })
})
