/* «Аудит платформы» и управление пользователями глазами супер-админа:
   тарифы и цены, подписки, промокоды, товары, заказы, выплаты, журнал —
   и платформенный реестр аккаунтов (authsvc).

   Тут два рода проверок. Первый — деньги и доступ: цена меняется только
   супер-админом, промокод действует по своим правилам, заказ подтверждается
   ровно один раз. Второй — необратимость: деактивация, восстановление и
   окончательное удаление аккаунта должны отличаться друг от друга, иначе
   «удалил по ошибке» станет концом истории. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq, dbQuery, Session } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin } from '../setup/factory.js'
import { useAuthStore } from '@/stores/auth.js'
import * as billing from '@/api/billing.js'
import * as users from '@/api/users.js'

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

async function superAdmin(label = 'root') {
  const u = await registerVerified(label)
  dbQuery(`UPDATE users SET is_super_admin = TRUE WHERE id = ${u.auth.userId}`)
  const s = new Session(label + '-super')
  s.use()
  const a = useAuthStore()
  await a.login(u.login, u.password)
  return { session: s, auth: a, userId: u.auth.userId, login: u.login, password: u.password }
}

describeIntegration('billing admin: тарифы, цены и настройки', () => {
  it('сводка платформы отдаётся супер-админу', async () => {
    const root = await superAdmin()
    root.session.use()
    expect(await billing.adminOverview()).toBeTruthy()
  })

  it('цена тарифа меняется и попадает в витрину', async () => {
    const root = await superAdmin()
    root.session.use()

    const showcase = await billing.getShowcase()
    const plan = (showcase.plans ?? []).find((p) => p.price_month > 0)
    if (!plan) return // линейка задаётся данными

    // Тариф сохраняется целиком (как в разделе «Тарифы и цены»): частичное
    // тело обнулило бы остальные поля.
    await billing.adminUpdatePlan(plan.code, { ...plan, price_month: 123400 })
    const after = await billing.getShowcase()
    const updated = (after.plans ?? []).find((p) => p.code === plan.code)
    expect(updated.price_month).toBe(123400)

    // Каталог цен один на платформу — возвращаем как было, иначе следующие
    // проверки будут считать по подменённому прайсу.
    await billing.adminUpdatePlan(plan.code, plan)
  })

  it('выдуманный тариф и докупка не правятся', async () => {
    const root = await superAdmin()
    root.session.use()
    await expectClientError(billing.adminUpdatePlan('нет-такого-тарифа', { price_month: 100 }))
    await expectClientError(billing.adminUpdateAddon('нет-такой-докупки', { price: 100 }))
  })

  it('настройки платформы читаются и правятся', async () => {
    const root = await superAdmin()
    root.session.use()
    const before = await billing.adminGetSettings()
    expect(before).toBeTruthy()
    // Настройки тоже сохраняются целиком — иначе правка комиссии выключила бы
    // магазин и приём платежей.
    await billing.adminUpdateSettings({ ...before, commission_pct: 25 })
    const after = await billing.adminGetSettings()
    expect(after.commission_pct).toBe(25)
    expect(after.store_enabled).toBe(before.store_enabled)

    await billing.adminUpdateSettings(before)
  })
})

describeIntegration('billing admin: подписки и токены', () => {
  it('выданная подписка видна в списке, отзыв её снимает', async () => {
    const root = await superAdmin()
    const client = await registerVerified('client')

    root.session.use()
    await billing.adminGrantSubscription({ user_id: client.auth.userId, plan: 'senior', days: 30 })
    const list = await billing.adminSubscriptions({ q: client.login })
    expect((list.items ?? []).some((s) => s.user_id === client.auth.userId)).toBe(true)

    await billing.adminRevokeSubscription(client.auth.userId)
    const after = await billing.adminSubscriptions({ q: client.login })
    const row = (after.items ?? []).find((s) => s.user_id === client.auth.userId)
    // Снятая подписка либо исчезает из списка, либо помечается неактивной —
    // но не остаётся действующей.
    expect(row == null || row.active === false || row.status !== 'active').toBe(true)
  })

  it('выдуманный тариф не выдаётся', async () => {
    const root = await superAdmin()
    const client = await registerVerified('client')
    root.session.use()
    await expectClientError(billing.adminGrantSubscription({
      user_id: client.auth.userId, plan: 'галактический', days: 30,
    }))
  })

  it('подаренные токены увеличивают остаток, сброс возвращает суточную норму', async () => {
    const root = await superAdmin()
    const client = await registerVerified('client')

    client.session.use()
    const before = await billing.getAiUsage()

    root.session.use()
    await billing.adminGrantTokens(client.auth.userId, 5000)

    client.session.use()
    const granted = await billing.getAiUsage()
    expect(granted.tokens_left).toBeGreaterThan(before.tokens_left)

    root.session.use()
    await billing.adminResetTokens(client.auth.userId)
    client.session.use()
    const reset = await billing.getAiUsage()
    expect(reset.tokens_used).toBe(0)
  })

  it('поиск пользователей по подписке находит по логину', async () => {
    const root = await superAdmin()
    const client = await registerVerified('client')
    root.session.use()
    const found = await billing.adminSearchUsers(client.login)
    expect((found.items ?? found.users ?? []).some((u) => u.id === client.auth.userId)).toBe(true)
  })
})

describeIntegration('billing admin: промокоды', () => {
  it('созданный промокод действует, удалённый — нет', async () => {
    const root = await superAdmin()
    root.session.use()
    const code = uniq('GIFT').toUpperCase()
    const promo = await billing.adminCreatePromo({
      code, kind: 'tokens', value: 500, max_uses: 5, per_user_limit: 1, is_active: true,
    })
    const promoId = promo.id ?? promo.promo?.id
    expect(promoId).toBeTruthy()

    // Подарочный промокод активируется сам по себе; скидочный — только вместе
    // с покупкой (это и отличает подарок от скидки).
    const client = await registerVerified('client')
    client.session.use()
    const applied = await billing.activatePromo(code)
    expect(applied).toBeTruthy()

    const percentCode = uniq('SALE').toUpperCase()
    root.session.use()
    await billing.adminCreatePromo({
      code: percentCode, kind: 'percent', value: 20, max_uses: 5, per_user_limit: 1, is_active: true,
    })
    client.session.use()
    await expectClientError(billing.activatePromo(percentCode))

    root.session.use()
    await billing.adminDeletePromo(promoId)

    const other = await registerVerified('other')
    other.session.use()
    await expectClientError(billing.activatePromo(code))
  })

  it('промокод с исчерпанным лимитом не активируется', async () => {
    const root = await superAdmin()
    root.session.use()
    const code = uniq('ONCE').toUpperCase()
    await billing.adminCreatePromo({ code, kind: 'tokens', value: 100, max_uses: 1, per_user_limit: 1, is_active: true })

    const first = await registerVerified('first')
    first.session.use()
    await billing.activatePromo(code)

    const second = await registerVerified('second')
    second.session.use()
    // Лимит применений — не украшение: второй активировать не должен.
    await expectClientError(billing.activatePromo(code))
  })

  it('один и тот же промокод дважды одному человеку не достаётся', async () => {
    const root = await superAdmin()
    root.session.use()
    const code = uniq('TWICE').toUpperCase()
    await billing.adminCreatePromo({ code, kind: 'tokens', value: 100, max_uses: 10, per_user_limit: 1, is_active: true })

    const client = await registerVerified('client')
    client.session.use()
    await billing.activatePromo(code)
    await expectClientError(billing.activatePromo(code))
  })

  it('правка промокода отражается на его действии', async () => {
    const root = await superAdmin()
    root.session.use()
    const code = uniq('EDIT').toUpperCase()
    const promo = await billing.adminCreatePromo({ code, kind: 'tokens', value: 100, max_uses: 10, per_user_limit: 1, is_active: true })
    const promoId = promo.id ?? promo.promo.id

    await billing.adminUpdatePromo(promoId, { ...promo, is_active: false })
    const client = await registerVerified('client')
    client.session.use()
    await expectClientError(billing.activatePromo(code))
  })

  it('список промокодов закрыт от обычного пользователя', async () => {
    const u = await newCompanyAdmin()
    u.session.use()
    await expectStatus(billing.adminPromos(), 403)
    await expectStatus(billing.adminCreatePromo({ code: 'HACK', kind: 'percent', value: 100 }), 403)
  })
})

describeIntegration('billing admin: товары и модерация', () => {
  it('товар платформы заводится, правится и снимается', async () => {
    const root = await superAdmin()
    root.session.use()
    const title = uniq('Тема платформы ')
    const created = await billing.adminCreateProduct({
      kind: 'theme', title, description: 'Официальная тема', price: 9900,
      payload: {}, status: 'published', sort: 0,
    })
    const id = created.id ?? created.product?.id
    expect(id).toBeTruthy()

    await billing.adminUpdateProduct(id, { ...created, title: title + ' (обновлена)' })
    const list = await billing.adminProducts()
    expect((list.items ?? list.products ?? []).some((p) => p.id === id)).toBe(true)

    await billing.adminDeleteProduct(id)
  })

  it('товар автора публикуется после проверки, отклонённый — с причиной', async () => {
    const root = await superAdmin()
    const author = await registerVerified('author')

    author.session.use()
    const good = await billing.createMyProduct({
      kind: 'theme', title: uniq('Хорошая '), description: 'Ок', price: 19900, payload: {},
    })
    await billing.submitMyProduct(good.id)

    const bad = await billing.createMyProduct({
      kind: 'theme', title: uniq('Плохая '), description: 'Нет', price: 19900, payload: {},
    })
    await billing.submitMyProduct(bad.id)

    root.session.use()
    await billing.adminReviewProduct(good.id, true)
    await billing.adminReviewProduct(bad.id, false, 'Не соответствует правилам')

    author.session.use()
    const mine = await billing.getMyStore()
    const goodRow = (mine.products ?? []).find((p) => p.id === good.id)
    const badRow = (mine.products ?? []).find((p) => p.id === bad.id)
    expect(goodRow.status).toBe('published')
    expect(badRow.status).toBe('rejected')
    // Причина обязана дойти до автора — иначе он не поймёт, что исправлять.
    expect(badRow.reject_reason).toContain('правилам')

    // Опубликованный товар появляется в общей витрине.
    const catalog = await billing.getProducts({})
    expect((catalog.items ?? catalog.products ?? []).some((p) => p.id === good.id)).toBe(true)
    const card = await billing.getProduct(good.id)
    expect(card.id).toBe(good.id)
  })

  it('автор снимает свой товар с продажи', async () => {
    const root = await superAdmin()
    const author = await registerVerified('author')

    author.session.use()
    const product = await billing.createMyProduct({
      kind: 'theme', title: uniq('Снимаемая '), description: '', price: 19900, payload: {},
    })
    await billing.submitMyProduct(product.id)

    root.session.use()
    await billing.adminReviewProduct(product.id, true)

    author.session.use()
    await billing.withdrawMyProduct(product.id)
    const catalog = await billing.getProducts({})
    expect((catalog.items ?? catalog.products ?? []).some((p) => p.id === product.id)).toBe(false)
  })

  it('чужой товар модерировать нельзя', async () => {
    const author = await registerVerified('author')
    author.session.use()
    const product = await billing.createMyProduct({
      kind: 'theme', title: uniq('Чужая '), description: '', price: 10000, payload: {},
    })
    await billing.submitMyProduct(product.id)

    const stranger = await newCompanyAdmin('stranger')
    stranger.session.use()
    await expectStatus(billing.adminReviewProduct(product.id, true), 403)
  })
})

describeIntegration('billing admin: заказы и выплаты', () => {
  it('заказ подтверждается ровно один раз', async () => {
    const root = await superAdmin()
    const client = await registerVerified('client')

    client.session.use()
    const showcase = await billing.getShowcase()
    // Бесплатный тариф не оформляют: он и так действует по умолчанию.
    const plan = (showcase.plans ?? []).find((p) => p.price_month > 0 && p.code !== 'junior')
    if (!plan) return

    const order = await billing.purchase({
      kind: 'subscription', item_code: plan.code, period: 'month', qty: 1,
    })
    const orderId = order.id ?? order.order?.id
    expect(orderId).toBeTruthy()

    root.session.use()
    await billing.adminConfirmOrder(orderId)
    // Повторное подтверждение не должно применить заказ во второй раз —
    // иначе оплата одного счёта дважды продлит подписку.
    await expectClientError(billing.adminConfirmOrder(orderId))
  })

  it('список заказов и журнал действий доступны супер-админу', async () => {
    const root = await superAdmin()
    root.session.use()
    expect(await billing.adminOrders({})).toBeTruthy()
    expect(await billing.adminAudit({})).toBeTruthy()
    expect(await billing.adminPayouts()).toBeTruthy()
  })

  it('несуществующая выплата не обрабатывается', async () => {
    const root = await superAdmin()
    root.session.use()
    await expectClientError(billing.adminProcessPayout(999999, 'paid', 'перевод'))
  })

  it('автопродление переключается только при действующей подписке', async () => {
    const root = await superAdmin()
    const client = await registerVerified('client')

    // Без подписки продлевать нечего — ручка обязана это сказать.
    client.session.use()
    await expectClientError(billing.setAutoRenew(false))
    await expectClientError(billing.cancelAddon(999999))

    root.session.use()
    await billing.adminGrantSubscription({ user_id: client.auth.userId, plan: 'senior', days: 30 })

    client.session.use()
    await billing.setAutoRenew(false)
    await billing.setAutoRenew(true)
  })
})

describeIntegration('users API: платформенный реестр аккаунтов', () => {
  it('супер-админ видит всех и заводит аккаунт', async () => {
    const root = await superAdmin()
    root.session.use()
    const list = await users.getUsers()
    expect((list.items ?? list.users ?? list).length).toBeGreaterThan(0)

    const login = uniq('plat_')
    const created = await users.createPlatformUser({
      fio: 'Платформин Платформ Платформович',
      login,
      email: `${login}@apitest.local`,
      password: 'secret-pass-123',
    })
    const id = created.id ?? created.user?.id
    expect(id).toBeTruthy()

    // Карточку супер-админ видит в платформенном списке: /users/:id — ручка
    // администратора КОМПАНИИ, а супер-админ в роли компании не состоит.
    const after = await users.getUsers()
    expect((after.items ?? after.users ?? after).some((u) => u.id === id)).toBe(true)
  })

  it('деактивация и восстановление обратимы, окончательное удаление — нет', async () => {
    const root = await superAdmin()
    const victim = await registerVerified('victim')
    root.session.use()

    await users.deletePlatformUser(victim.auth.userId)
    // Деактивация не стирает данные: аккаунт остаётся в базе выключённым,
    // иначе случайное нажатие уносило бы всю его работу.
    expect(dbQuery(`SELECT is_active FROM users WHERE id = ${victim.auth.userId}`)[0][0]).toBe('f')

    await users.reactivatePlatformUser(victim.auth.userId)
    expect(dbQuery(`SELECT is_active FROM users WHERE id = ${victim.auth.userId}`)[0][0]).toBe('t')

    await users.deletePlatformUser(victim.auth.userId)
    await users.purgePlatformUser(victim.auth.userId)
    expect(dbQuery(`SELECT count(*) FROM users WHERE id = ${victim.auth.userId}`)[0][0]).toBe('0')
  })

  it('деактивированный аккаунт не пускают войти', async () => {
    const root = await superAdmin()
    const victim = await registerVerified('victim')
    root.session.use()
    await users.deletePlatformUser(victim.auth.userId)

    const s = new Session('victim-retry')
    s.use()
    await expect(useAuthStore().login(victim.login, victim.password)).rejects.toBeTruthy()
  })

  it('супер-админа чужими руками не тронуть', async () => {
    const root = await superAdmin('root')
    const other = await superAdmin('other')

    // Один супер-админ не управляет другим — иначе достаточно захватить один
    // аккаунт, чтобы снести остальных.
    other.session.use()
    await expectClientError(users.deletePlatformUser(root.userId))
    await expectClientError(users.updatePlatformUser(root.userId, { fio: 'Переименован' }))
  })

  it('сброс пароля платформенного пользователя возвращает временный', async () => {
    const root = await superAdmin()
    const client = await registerVerified('client')
    root.session.use()
    const res = await users.resetPlatformUserPassword(client.auth.userId)
    expect(res).toBeTruthy()
    // После сброса аккаунт обязан требовать смены пароля при входе.
    expect(dbQuery(`SELECT is_default_pass FROM users WHERE id = ${client.auth.userId}`)[0][0]).toBe('t')
  })

  it('платформенный реестр закрыт от администратора компании', async () => {
    const u = await newCompanyAdmin()
    u.session.use()
    await expectStatus(users.getUsers(), 403)
    await expectStatus(users.createPlatformUser({
      fio: 'Самозванцев Само Званцевич', login: uniq('x_'), email: `${uniq('x_')}@apitest.local`, password: 'secret-pass-123',
    }), 403)
    await expectStatus(users.purgePlatformUser(u.auth.userId), 403)
  })

  it('роль назначается в рамках компании, а не глобально', async () => {
    const admin = await newCompanyAdmin('admin')
    const member = await registerVerified('member')
    admin.session.use()
    // Роли живут в членстве: без указания компании назначать нечего.
    await expectClientError(users.assignRole(member.auth.userId, { role_id: 3 }))
  })
})
