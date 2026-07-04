import { describe, it, expect } from 'vitest'
import { SLIDES, SLIDE_COMPONENTS, visibleSlides } from './tvSlides.js'

describe('tvSlides каталог', () => {
  it('id слайдов уникальны', () => {
    const ids = SLIDES.map((s) => s.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('для каждого kind есть зарегистрированный компонент', () => {
    for (const s of SLIDES) {
      expect(SLIDE_COMPONENTS[s.kind], s.id).toBeTruthy()
    }
  })
})

describe('visibleSlides', () => {
  it('без выключенных и с долгом > 0 — все слайды', () => {
    const list = visibleSlides([], { debtValue: 5 })
    expect(list.length).toBe(SLIDES.length)
  })

  it('скрывает слайд «долг», когда долга нет', () => {
    const list = visibleSlides([], { debtValue: 0 })
    expect(list.some((s) => s.kind === 'debt')).toBe(false)
    expect(list.length).toBe(SLIDES.length - 1)
  })

  it('исключает выключенные в настройках', () => {
    const list = visibleSlides(['today-podium'], { debtValue: 5 })
    expect(list.some((s) => s.id === 'today-podium')).toBe(false)
  })

  it('всё выключено → не гаснет, показывает брендовый слайд', () => {
    const allIds = SLIDES.map((s) => s.id)
    const list = visibleSlides(allIds, { debtValue: 5 })
    expect(list).toHaveLength(1)
    expect(list[0].kind).toBe('brand')
  })

  it('дефолтные аргументы: долг скрыт при debtValue=0 по умолчанию', () => {
    const list = visibleSlides()
    expect(list.some((s) => s.kind === 'debt')).toBe(false)
  })
})
