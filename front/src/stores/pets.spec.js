import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/pets.js', () => ({
  getMyPet: vi.fn(),
  feedPet: vi.fn(),
  renamePet: vi.fn(),
  equipItem: vi.fn(),
  switchSpecies: vi.fn(),
  claimQuest: vi.fn(),
  startAdventure: vi.fn(),
  getShop: vi.fn(),
  getMysteryItem: vi.fn(),
  buyItem: vi.fn(),
  buySpecies: vi.fn(),
  walkPet: vi.fn(),
  healPet: vi.fn(),
  strokePet: vi.fn(),
  getZoo: vi.fn(),
  getRating: vi.fn(),
  getLive: vi.fn(),
  getActivityLog: vi.fn(),
}))

import * as api from '@/api/pets.js'
import { usePetsStore } from './pets.js'
import { useAuthStore } from './auth.js'
import { useNotificationsStore } from './notifications.js'

describe('pets store', () => {
  let pets
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    pets = usePetsStore()
  })

  it('fetchPet кладёт снапшот питомца в стор', async () => {
    api.getMyPet.mockResolvedValue({ user_id: 1, name: 'Грувик', kudos: 5 })
    await pets.fetchPet()
    expect(pets.pet).toEqual({ user_id: 1, name: 'Грувик', kudos: 5 })
  })

  it('feedPet мёржит ответ поверх текущего питомца (контекстные поля)', async () => {
    pets.pet = { user_id: 1, kudos: 10, xp: 5 }
    api.feedPet.mockResolvedValue({ kudos: 7, xp: 17, phrase: 'Вкусно!' })
    const res = await pets.feedPet()
    expect(pets.pet).toEqual({ user_id: 1, kudos: 7, xp: 17, phrase: 'Вкусно!' })
    expect(res.phrase).toBe('Вкусно!')
  })

  it('startAdventure мёржит ответ поверх текущего питомца', async () => {
    pets.pet = { user_id: 1, kudos: 10, adventure_until: null, adventure_place: null }
    api.startAdventure.mockResolvedValue({
      adventure_until: '2026-07-07T18:00:00.000000+00:00',
      adventure_place: 'на речку',
    })
    const res = await pets.startAdventure()
    expect(pets.pet.kudos).toBe(10) // бесплатно
    expect(pets.pet.adventure_until).toBe('2026-07-07T18:00:00.000000+00:00')
    expect(pets.pet.adventure_place).toBe('на речку')
    expect(res.adventure_place).toBe('на речку')
  })

  it('fetchPet с adventure_reward показывает уведомление о возврате', async () => {
    const notify = useNotificationsStore()
    const spy = vi.spyOn(notify, 'success').mockImplementation(() => {})
    api.getMyPet.mockResolvedValue({
      user_id: 1, kudos: 15, xp: 20,
      adventure_until: null, adventure_place: null,
      adventure_reward: { kudos: 7, xp: 9, place: 'в горы' },
    })
    await pets.fetchPet()
    expect(pets.pet.kudos).toBe(15)
    expect(spy).toHaveBeenCalledWith('Вернулся из приключения: +7 кудосов, +9 XP')
  })

  it('fetchPet без adventure_reward уведомление не показывает', async () => {
    const notify = useNotificationsStore()
    const spy = vi.spyOn(notify, 'success').mockImplementation(() => {})
    api.getMyPet.mockResolvedValue({ user_id: 1, kudos: 5, adventure_until: null })
    await pets.fetchPet()
    expect(spy).not.toHaveBeenCalled()
  })

  it('walkPet/healPet обновляют питомца из ответа', async () => {
    pets.pet = { user_id: 1, kudos: 20 }
    api.walkPet.mockResolvedValue({ kudos: 15, recovered: false })
    await pets.walkPet()
    expect(pets.pet.kudos).toBe(15)

    api.healPet.mockResolvedValue({ kudos: 7, recovered: true })
    const res = await pets.healPet()
    expect(pets.pet.kudos).toBe(7)
    expect(res.recovered).toBe(true)
  })

  it('strokePet обновляет запись в зоопарке по user_id', async () => {
    pets.zoo = [{ user_id: 2, name: 'Кекс', xp: 10 }]
    api.strokePet.mockResolvedValue({ user_id: 2, name: 'Кекс', xp: 12 })
    await pets.strokePet(2)
    expect(pets.zoo[0].xp).toBe(12)
  })

  it('fetchShop разбирает {items}', async () => {
    api.getShop.mockResolvedValue({ items: [{ key: 'crown', kind: 'accessory' }] })
    await pets.fetchShop()
    expect(pets.shop).toEqual([{ key: 'crown', kind: 'accessory' }])
    expect(pets.shopLoaded).toBe(true)
  })

  it('buyItem обновляет питомца и перечитывает магазин', async () => {
    pets.pet = { user_id: 1, kudos: 20, accessories: [] }
    api.buyItem.mockResolvedValue({ kudos: 17, accessories: ['crown'] })
    api.getShop.mockResolvedValue({ items: [] })
    await pets.buyItem('crown')
    expect(pets.pet.kudos).toBe(17)
    expect(api.getShop).toHaveBeenCalledOnce()
  })

  it('fetchZoo/fetchRating/fetchLive/fetchActivityLog заполняют состояние', async () => {
    api.getZoo.mockResolvedValue([{ user_id: 1 }])
    await pets.fetchZoo()
    expect(pets.zoo).toEqual([{ user_id: 1 }])

    api.getRating.mockResolvedValue({ items: [], me: null, total: 0 })
    await pets.fetchRating()
    expect(pets.rating).toEqual({ items: [], me: null, total: 0 })

    api.getLive.mockResolvedValue({ items: [{ unit_id: 9 }] })
    await pets.fetchLive()
    expect(pets.live).toEqual([{ unit_id: 9 }])
    expect(pets.liveLoaded).toBe(true)

    api.getActivityLog.mockResolvedValue({ items: [{ kind: 'fed' }] })
    await pets.fetchActivityLog()
    expect(pets.activityLog).toEqual([{ kind: 'fed' }])
    expect(pets.activityLoaded).toBe(true)
  })

  it('applyPetUpdate игнорирует чужой user_id, отражает свой в зоопарке', () => {
    pets.pet = { user_id: 1, kudos: 5 }
    pets.zoo = [{ user_id: 1, kudos: 5 }, { user_id: 2, kudos: 9 }]

    pets.applyPetUpdate({ user_id: 2, kudos: 100 })
    expect(pets.pet.kudos).toBe(5) // не моё обновление — не трогаем pet

    pets.applyPetUpdate({ user_id: 1, kudos: 8 })
    expect(pets.pet.kudos).toBe(8)
    expect(pets.zoo[0].kudos).toBe(8)
  })

  it('isMine: без активной компании данные всех считаются своими', () => {
    expect(pets.isMine(null)).toBe(true)
    expect(pets.isMine(42)).toBe(true) // myCompanyId не задан — fail-open
  })

  it('isMine: с активной компанией фильтрует чужие события', () => {
    const auth = useAuthStore()
    auth.user = { id: 1, company_id: 7 }
    expect(pets.isMine(7)).toBe(true)
    expect(pets.isMine(8)).toBe(false)
  })
})
