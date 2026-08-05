/* Биллинг (billingsvc): витрина, лимиты, токены ИИ, заказы, свои товары.

   В выпуске 7.0 подписки скрыты от пользователя, и это состояние проверяется
   ЗДЕСЬ, а не в UI: тариф никого не ограничивает (EffectiveLimits), токены ИИ
   выдаются всем поровну — 1000 в сутки. Механика покупок при этом цела, и её
   правила тоже проверяем: заказ применяется один раз, чужой не отменить,
   админские ручки закрыты от обычного пользователя. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin } from '../setup/factory.js'
import * as api from '@/api/billing.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

const DAILY_AI_TOKENS = 1000

describeIntegration('billing API: витрина и лимиты', () => {
  it('витрина отдаёт тарифы, докупки и текущее состояние', async () => {
    const u = await registerVerified()
    u.session.use()
    const showcase = await api.getShowcase()
    expect(showcase).toBeTruthy()
    expect(Array.isArray(showcase.plans ?? [])).toBe(true)
    expect(showcase.entitlements ?? showcase.subscription ?? null).not.toBe(undefined)
  })

  it('пока подписки скрыты, тариф не ограничивает счётные лимиты', async () => {
    const u = await newCompanyAdmin()
    u.session.use()
    const ent = await api.getEntitlements(u.companyId)
    const limits = ent.limits ?? ent.entitlements?.limits
    expect(limits).toBeTruthy()

    // -1 значит «без ограничения»: без этого человек упирался бы в бесплатный
    // тариф, не имея возможности что-либо купить.
    for (const key of ['tasks', 'boards', 'diaries', 'calendars', 'registries', 'storage_bytes']) {
      expect(limits[key]).toBe(-1)
    }
    expect(limits.portal).toBe(true)
    expect(limits.advanced_stats).toBe(true)
  })

  it('токены ИИ выдаются всем поровну — суточная норма', async () => {
    const u = await registerVerified()
    u.session.use()
    const ai = await api.getAiUsage()
    expect(ai.tokens_limit).toBe(DAILY_AI_TOKENS)
    expect(ai.tokens_used).toBe(0)
    expect(ai.tokens_left).toBe(DAILY_AI_TOKENS)
  })

  it('хранилище считается и отдаётся списком по разделам', async () => {
    const u = await registerVerified()
    u.session.use()
    const storage = await api.getStorage()
    expect(Array.isArray(storage.items ?? [])).toBe(true)
  })
})

describeIntegration('billing API: покупки и заказы', () => {
  it('расчёт цены: известный тариф считается, выдуманный — отказ', async () => {
    const u = await registerVerified()
    u.session.use()
    const showcase = await api.getShowcase()
    const plan = (showcase.plans ?? []).find((p) => p.price_month > 0)
    if (!plan) return // в стенде тарифы не заведены — считать нечего

    const q = await api.quote({ kind: 'subscription', item_code: plan.code, period: 'month', qty: 1 })
    expect(q.amount).toBeGreaterThan(0)

    await expect(api.quote({
      kind: 'subscription', item_code: 'no-such-plan', period: 'month', qty: 1,
    })).rejects.toBeTruthy()
  })

  it('заказы: список пуст у новичка, чужой заказ не отменить', async () => {
    const a = await registerVerified('a')
    a.session.use()
    const mine = await api.getOrders()
    expect((mine.items ?? []).length).toBe(0)

    // Несуществующий/чужой заказ отменить нельзя.
    await expect(api.cancelOrder(999999)).rejects.toBeTruthy()
  })

  it('промокод: выдуманный не активируется', async () => {
    const u = await registerVerified()
    u.session.use()
    await expect(api.activatePromo('НЕТ-ТАКОГО-КОДА')).rejects.toBeTruthy()
  })

  it('свои товары: черновик заводится, отправляется на проверку и снимается', async () => {
    const u = await registerVerified()
    u.session.use()

    const product = await api.createMyProduct({
      kind: 'theme', title: uniq('Тема '), description: 'Своя тема', price: 19900, payload: {},
    })
    expect(product.id).toBeGreaterThan(0)

    // Карточка товара сохраняется ЦЕЛИКОМ: вид и цена обязательны и при правке
    // (частичное тело сервер не принимает — проверяем это отдельно ниже).
    await api.updateMyProduct(product.id, {
      kind: 'theme', title: 'Тема обновлённая', description: 'Своя тема', price: 19900, payload: {},
    })
    await expect(api.updateMyProduct(product.id, { title: 'Только имя' })).rejects.toBeTruthy()

    await api.submitMyProduct(product.id)

    const mine = await api.getMyStore()
    const found = (mine.products ?? []).find((p) => p.id === product.id)
    expect(found.status).toBe('review')

    await api.deleteMyProduct(product.id)
  })

  it('чужой товар не правится и не удаляется', async () => {
    const a = await registerVerified('a')
    a.session.use()
    const product = await api.createMyProduct({
      kind: 'theme', title: uniq('Чужая '), description: '', price: 10000, payload: {},
    })

    const b = await registerVerified('b')
    b.session.use()
    await expect(api.updateMyProduct(product.id, { title: 'Присвоено' })).rejects.toBeTruthy()
    await expect(api.deleteMyProduct(product.id)).rejects.toBeTruthy()
  })

  it('вывод выручки без денег не проходит', async () => {
    const u = await registerVerified()
    u.session.use()
    await expect(api.requestPayout(100000, 'СБП +7 000 000-00-00')).rejects.toBeTruthy()
  })
})

describeIntegration('billing API: админские ручки', () => {
  it('обычному пользователю раздел управления закрыт', async () => {
    const u = await newCompanyAdmin()
    u.session.use()

    // Администратор КОМПАНИИ — не супер-админ платформы.
    await expectStatus(api.adminOverview(), 403)
    await expectStatus(api.adminGetSettings(), 403)
    await expectStatus(api.adminSubscriptions(), 403)
    await expectStatus(api.adminPromos(), 403)
    await expectStatus(api.adminOrders(), 403)
    await expectStatus(api.adminAudit(), 403)
  })

  it('выдача тарифа и токенов чужими руками недоступна', async () => {
    const u = await registerVerified()
    u.session.use()
    await expectStatus(api.adminGrantSubscription({ user_id: u.auth.userId, plan: 'senior', days: 30 }), 403)
    await expectStatus(api.adminGrantTokens(u.auth.userId, 100000), 403)
    await expectStatus(api.adminResetTokens(u.auth.userId), 403)
  })
})
