import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import FloatingPet from './FloatingPet.vue'
import PetDetailModal from './PetDetailModal.vue'

vi.mock('@/api/pets.js', () => ({
  getMyPet: vi.fn(() => Promise.resolve(null)),
}))

function factory(pet) {
  const pinia = createTestingPinia({
    createSpy: vi.fn,
    stubActions: false,
    initialState: { pets: { pet }, auth: { user: { id: 1 } } },
  })
  return mount(FloatingPet, {
    global: {
      plugins: [pinia],
      stubs: { teleport: true, PetDetailModal: true },
    },
  })
}

const mkPet = (over = {}) => ({
  user_id: 1, name: 'Грувик', species: 'fox', stage: 3, xp: 50,
  next_stage_xp: 280, kudos: 10, sick: false, last_fed_date: null, ...over,
})

describe('FloatingPet', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('ничего не рендерит без питомца', () => {
    const w = factory(null)
    expect(w.find('.fp-avatar').exists()).toBe(false)
  })

  it('рендерит аватар питомца', () => {
    const w = factory(mkPet())
    expect(w.find('.fp-avatar').exists()).toBe(true)
    expect(w.find('.fp-sick-badge').exists()).toBe(false)
  })

  it('болеющий питомец — бейдж и класс sick', () => {
    const w = factory(mkPet({ sick: true }))
    expect(w.find('.fp-sick-badge').exists()).toBe(true)
    expect(w.find('.fp-emoji').classes()).toContain('sick')
  })

  it('клик по аватару открывает модалку питомца', async () => {
    const w = factory(mkPet())
    expect(w.findComponent(PetDetailModal).exists()).toBe(false)
    await w.find('.fp-avatar').trigger('click')
    expect(w.findComponent(PetDetailModal).exists()).toBe(true)
  })

  it('переход в болезнь показывает пузырь с подсказкой полечить', async () => {
    const w = factory(mkPet({ sick: false }))
    expect(w.find('.fp-bubble').exists()).toBe(false)
    const store = w.vm.$pinia.state.value.pets
    store.pet = mkPet({ sick: true })
    await w.vm.$nextTick()
    await w.vm.$nextTick()
    expect(w.find('.fp-bubble').exists()).toBe(true)
    expect(w.find('.fp-bubble').text()).toContain('заболел')
  })

  it('тап по пузырю открывает модалку и прячет пузырь', async () => {
    const w = factory(mkPet({ sick: false }))
    const store = w.vm.$pinia.state.value.pets
    store.pet = mkPet({ sick: true })
    await w.vm.$nextTick()
    await w.vm.$nextTick()
    await w.find('.fp-bubble').trigger('click')
    expect(w.find('.fp-bubble').exists()).toBe(false)
    expect(w.findComponent(PetDetailModal).exists()).toBe(true)
  })
})
