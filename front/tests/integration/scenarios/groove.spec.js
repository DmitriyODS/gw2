// Сценарий «Мой Groove» через useGrooveStore и api-модуль против живого
// groovesvc: питомец, кормление, кудосы (категория+текст обязательны, 422),
// рейтинг, рейд. Плюс контроль: фронт больше НЕ зовёт zap/stroke.
import { it, expect } from 'vitest'
import { describeIntegration, dbQuery } from '../setup/harness.js'
import { newCompanyAdmin, newMember } from '../setup/factory.js'
import { useGrooveStore } from '@/stores/groove.js'
import * as grooveApi from '@/api/groove.js'

describeIntegration('groove api/store', () => {
  it('getPet создаёт питомца, feedPet кормит', async () => {
    const admin = await newCompanyAdmin('grooveadmin')
    admin.session.use()
    const store = useGrooveStore()
    await store.fetchPet()
    expect(store.pet).toBeTruthy()
    expect(store.pet.user_id).toBe(admin.auth.userId)

    // Голодному питомцу без грувов кормление отвечает 422 (проверяем разбор
    // ошибки клиентом), затем выдаём грувы напрямую и кормим успешно.
    await expect(store.feedPet()).rejects.toMatchObject({ status: 422 })
    dbQuery(`UPDATE pets SET beans = 50 WHERE user_id = ${admin.auth.userId}`)
    const res = await store.feedPet()
    expect(res).toBeTruthy()
    expect(store.pet).toBeTruthy()
  })

  it('кудос: валидный проходит, себе — 422 SELF_KUDOS, чужая категория — 422', async () => {
    const admin = await newCompanyAdmin('kudosadmin')
    const member = await newMember(admin, admin.companyId, 1, 'kudosmember')

    admin.session.use()
    const store = useGrooveStore()

    // Валидный кудос admin→member (категория + текст).
    await expect(store.sendKudos(member.auth.userId, 'quality', 'отличная работа')).resolves.toBeTruthy()

    // Кудос себе запрещён.
    await expect(store.sendKudos(admin.auth.userId, 'helped', 'сам себе'))
      .rejects.toMatchObject({ status: 422, error: 'SELF_KUDOS' })

    // Неизвестная категория.
    await expect(store.sendKudos(member.auth.userId, 'wow', 'спасибо'))
      .rejects.toMatchObject({ status: 422, error: 'BAD_CATEGORY' })
  })

  it('rating отдаёт items/me/total; raid — boss/week_start', async () => {
    const admin = await newCompanyAdmin('ratingadmin')
    admin.session.use()
    const store = useGrooveStore()
    await store.fetchPet() // питомец нужен, чтобы попасть в рейтинг

    await store.fetchRating()
    expect(store.rating).toBeTruthy()
    expect(Array.isArray(store.rating.items)).toBe(true)
    expect(store.rating.me !== undefined).toBe(true)
    expect(store.rating.total !== undefined).toBe(true)

    await store.fetchRaid()
    expect(store.raid).toBeTruthy()
    expect(store.raid.boss).toBeTruthy()
    expect(store.raid.week_start).toBeTruthy()
  })

  it('фронтовый api/groove больше НЕ содержит zap/stroke', () => {
    const names = Object.keys(grooveApi)
    expect(names.some((n) => /zap/i.test(n))).toBe(false)
    expect(names.some((n) => /stroke/i.test(n))).toBe(false)
  })
})
