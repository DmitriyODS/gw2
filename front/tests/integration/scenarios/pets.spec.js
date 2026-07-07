// Сценарий «Питомцы» через usePetsStore и api-модуль против живого petsvc:
// питомец, кормление, магазин, рейтинг. Лента/кудосы-реакция/рейд/wrapped
// убраны вместе с домены — сценариев на них здесь больше нет.
import { it, expect } from 'vitest'
import { describeIntegration, dbQuery } from '../setup/harness.js'
import { newCompanyAdmin } from '../setup/factory.js'
import { usePetsStore } from '@/stores/pets.js'
import * as petsApi from '@/api/pets.js'

describeIntegration('pets api/store', () => {
  it('getMyPet создаёт питомца, feedPet кормит', async () => {
    const admin = await newCompanyAdmin('petsadmin')
    admin.session.use()
    const store = usePetsStore()
    await store.fetchPet()
    expect(store.pet).toBeTruthy()
    expect(store.pet.user_id).toBe(admin.auth.userId)

    // Голодному питомцу без кудосов кормление отвечает 422, затем выдаём
    // кудосы напрямую и кормим успешно.
    await expect(store.feedPet()).rejects.toMatchObject({ status: 422 })
    dbQuery(`UPDATE pets SET kudos = 50 WHERE user_id = ${admin.auth.userId}`)
    const res = await store.feedPet()
    expect(res).toBeTruthy()
    expect(store.pet).toBeTruthy()
  })

  it('rating отдаёт items/me/total', async () => {
    const admin = await newCompanyAdmin('petsratingadmin')
    admin.session.use()
    const store = usePetsStore()
    await store.fetchPet() // питомец нужен, чтобы попасть в рейтинг

    await store.fetchRating()
    expect(store.rating).toBeTruthy()
    expect(Array.isArray(store.rating.items)).toBe(true)
    expect(store.rating.me !== undefined).toBe(true)
    expect(store.rating.total !== undefined).toBe(true)
  })

  it('shop отдаёт витрину товаров с ценой/редкостью', async () => {
    const admin = await newCompanyAdmin('petsshopadmin')
    admin.session.use()
    const store = usePetsStore()
    await store.fetchShop()
    expect(Array.isArray(store.shop)).toBe(true)
    if (store.shop.length) {
      const item = store.shop[0]
      expect(item).toHaveProperty('rarity')
      expect(item).toHaveProperty('price_kudos')
      expect(item).toHaveProperty('unlock_kind')
    }
  })

  it('фронтовый api/pets больше НЕ содержит feed-ленту/кудос-реакцию/рейд/wrapped', () => {
    const names = Object.keys(petsApi)
    expect(names).not.toContain('getFeed')
    expect(names).not.toContain('sendKudos')
    expect(names).not.toContain('getRaid')
    expect(names).not.toContain('getWrapped')
    expect(names).not.toContain('getLocation')
    expect(names).not.toContain('searchCities')
    expect(names).not.toContain('getGrooveTv')
  })
})
