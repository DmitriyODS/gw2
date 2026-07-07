import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import PetDetailModal from './PetDetailModal.vue'
import FeedMiniGame from './FeedMiniGame.vue'

const pushMock = vi.fn()
vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal()
  return { ...actual, useRouter: () => ({ push: pushMock }) }
})

const mkPet = (over = {}) => ({
  user_id: 1, name: 'Грувик', species: 'fox', stage: 3, xp: 50,
  next_stage_xp: 280, kudos: 10, sick: false, feed_streak: 2,
  recovery: 0, recovery_target: 3, accessories: [], quest: null, ...over,
})

function factory({ pet = mkPet(), initialAction = null } = {}) {
  const pinia = createTestingPinia({
    createSpy: vi.fn,
    initialState: { pets: { pet, activityLog: [], activityLoaded: false } },
    stubActions: false,
  })
  return mount(PetDetailModal, {
    props: { initialAction },
    global: {
      plugins: [pinia],
      stubs: { teleport: true, FeedMiniGame: true, WalkMiniGame: true, HealMiniGame: true },
    },
  })
}

describe('PetDetailModal', () => {
  beforeEach(() => {
    pushMock.mockClear()
  })

  it('рендерит имя и стадию питомца', () => {
    const w = factory()
    expect(w.find('.pdm-name').text()).toBe('Грувик')
    expect(w.text()).toContain('Подросток')
  })

  it('клик по крестику эмитит close', async () => {
    const w = factory()
    await w.find('.pdm-close').trigger('click')
    expect(w.emitted('close')).toBeTruthy()
  })

  it('initialAction сразу открывает нужную мини-игру', async () => {
    const w = factory({ initialAction: 'feed' })
    await flushPromises()
    expect(w.findComponent(FeedMiniGame).exists()).toBe(true)
  })

  it('вкладка «История» подгружает журнал активности один раз', async () => {
    const w = factory()
    const store = w.vm.$pinia.state.value.pets
    expect(store.activityLoaded).toBe(false)
    await w.findAll('.pdm-tab')[1].trigger('click')
    await flushPromises()
    expect(w.find('.pdm-history').exists()).toBe(true)
  })

  it('кнопка «Перейти к грувикам» закрывает модалку и ведёт на /pets', async () => {
    const w = factory()
    await w.find('.pdm-link-btn').trigger('click')
    expect(w.emitted('close')).toBeTruthy()
    expect(pushMock).toHaveBeenCalledWith('/pets')
  })

  it('больной питомец показывает блок болезни вместо XP-бара', () => {
    const w = factory({ pet: mkPet({ sick: true }) })
    expect(w.find('.pdm-sick-block').exists()).toBe(true)
    expect(w.find('.pdm-xp').exists()).toBe(false)
  })

  it('кнопка «Полечить» видна только больному питомцу', () => {
    const healthy = factory({ pet: mkPet({ sick: false }) })
    expect(healthy.text()).not.toContain('Полечить')

    const sick = factory({ pet: mkPet({ sick: true }) })
    expect(sick.text()).toContain('Полечить')
  })
})
