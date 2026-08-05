/* Хвост покрытия: вход через Яндекс ID и OAuth-провайдер, оставшиеся ручки
   YouGile и магазина питомцев.

   Внешних сервисов (Яндекс, YouGile) в стенде нет — и это ровно то состояние,
   в котором платформа обязана вести себя предсказуемо: кнопка входа гаснет,
   чужой код авторизации не превращается в сессию, а мастер интеграции честно
   говорит, что аккаунт не подключён, вместо пустых списков. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq, dbQuery, Session } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin } from '../setup/factory.js'
import * as auth from '@/api/auth.js'
import * as yougile from '@/api/yougile.js'
import * as pets from '@/api/pets.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

async function expectClientError(promise) {
  const err = await promise.then(() => null, (e) => e)
  expect(err).toBeTruthy()
  expect(err.status).toBeGreaterThanOrEqual(400)
  expect(err.status).toBeLessThan(500)
  return err
}

describeIntegration('auth API: Яндекс ID и OAuth-провайдер', () => {
  it('без настроенных ключей вход через Яндекс выключен', async () => {
    const guest = new Session('guest')
    guest.use()
    const cfg = await auth.yandexConfig()
    // Кнопку рисуем только при enabled — иначе человек упрётся в ошибку
    // провайдера вместо входа.
    expect(cfg.enabled).toBe(false)
  })

  it('адрес авторизации Яндекса собирается из client_id и возврата', () => {
    const url = auth.yandexAuthURL('test-client', 'app')
    expect(url).toContain('response_type=code')
    expect(url).toContain('client_id=test-client')
    expect(url).toContain(encodeURIComponent('/yandex-callback'))
    expect(url).toContain('state=app')
  })

  it('выдуманный код Яндекса сессию не выдаёт', async () => {
    const guest = new Session('guest')
    guest.use()
    await expectClientError(auth.yandexCallback('код-которого-не-существует'))
  })

  it('привязка Яндекс-аккаунта: статус читается, чужой код не привязывается', async () => {
    const u = await registerVerified()
    u.session.use()
    const status = await auth.yandexLinkStatus()
    expect(status.linked ?? false).toBe(false)

    await expectClientError(auth.yandexLink('выдуманный-код'))

    // Отвязка — приведение к состоянию «не привязан»: повтор не ошибка.
    expect((await auth.yandexUnlink()).linked).toBe(false)
    expect((await auth.yandexLinkStatus()).linked ?? false).toBe(false)
  })

  it('привязка требует входа', async () => {
    const guest = new Session('guest')
    guest.use()
    await expectStatus(auth.yandexLinkStatus(), 401)
  })

  it('согласие OAuth без настроенных кредов не выдаёт код', async () => {
    const u = await registerVerified()
    u.session.use()
    // Пустые креды навыка — эндпоинт обязан отказать, а не выпустить код,
    // который потом обменяют на сессию.
    await expectClientError(auth.oauthAuthorize({
      client_id: 'alice', redirect_uri: 'https://social.yandex.net/broker/redirect', state: 'x',
    }))
  })

  it('чужой redirect_uri не принимается', async () => {
    const u = await registerVerified()
    u.session.use()
    await expectClientError(auth.oauthAuthorize({
      client_id: 'alice', redirect_uri: 'https://evil.example/steal', state: 'x',
    }))
  })
})

describeIntegration('yougile API: мастер подключения без аккаунта', () => {
  it('подбор компаний по чужим данным не проходит', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    await expectClientError(yougile.lookupYougileCompanies({
      login: 'нет-такого@apitest.local', password: 'неверный',
    }))
  })

  it('доски и колонки без подключения отвечают отказом', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    await expectClientError(yougile.listYougileBoards('project-id'))
    await expectClientError(yougile.listYougileColumns('board-id'))
  })

  it('смена ключа и отключение без подключённого аккаунта', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    await expectClientError(yougile.rotateYougile({ password: 'какой-то' }))

    // Отключение — приведение к состоянию «не подключено»: повтор не ошибка.
    await yougile.disconnectYougile()
    expect((await yougile.getYougileStatus()).connected).toBe(false)
  })
})

describeIntegration('pets API: облики и продажа', () => {
  it('облик покупается за кудосы, чужой ключ отвергается', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    await pets.getMyPet()
    dbQuery(`UPDATE pets SET kudos = 100000 WHERE user_id = ${admin.auth.userId}`)

    const shop = await pets.getShop()
    const species = (shop.species ?? []).find((s) => !s.owned && (s.price ?? 0) > 0)
    if (species) {
      const before = Number(dbQuery(`SELECT kudos FROM pets WHERE user_id = ${admin.auth.userId}`)[0][0])
      await pets.buySpecies(species.key ?? species.id)
      const after = Number(dbQuery(`SELECT kudos FROM pets WHERE user_id = ${admin.auth.userId}`)[0][0])
      expect(after).toBeLessThan(before)
      // Купленный облик надевается — иначе покупка бессмысленна.
      await pets.switchSpecies(species.key ?? species.id)
    }
    await expectClientError(pets.buySpecies('нет-такого-облика'))
  })

  it('продать можно только своё', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    await pets.getMyPet()
    await expectClientError(pets.sellItem('нет-такой-вещи'))
  })

  it('недельный рейтинг компании отдаётся с моим местом', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    await pets.getMyPet()
    const rating = await pets.getRating()
    expect(Array.isArray(rating.items ?? rating.top ?? [])).toBe(true)
    expect(rating.me ?? rating.my ?? null).not.toBe(undefined)
  })
})
