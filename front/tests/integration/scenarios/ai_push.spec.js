/* ИИ-возможности (aisvc) и реестр пуш-токенов (pushsvc).

   Настоящего LLM в стенде нет — и это ровно тот случай, который обязан
   отрабатывать корректно в проде: ключ не заведён, сеть недоступна, модель
   отвечает ошибкой. Здесь проверяется, что каждая ИИ-ручка в таком состоянии
   даёт понятный отказ и не роняет раздел, что личные настройки сохраняются, а
   сырой ключ наружу не отдаётся, и что платформенные настройки закрыты от
   всех, кроме супер-админа. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq, dbQuery, Session } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin, newMember } from '../setup/factory.js'
import { useAuthStore } from '@/stores/auth.js'
import * as ai from '@/api/ai.js'
import * as assistant from '@/api/assistant.js'
import * as push from '@/api/push.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

// Отказ ИИ обязан быть «человеческим»: 4xx с сообщением, а не 500.
async function expectClientError(promise) {
  const err = await promise.then(() => null, (e) => e)
  expect(err).toBeTruthy()
  expect(err.status).toBeGreaterThanOrEqual(400)
  expect(err.status).toBeLessThan(500)
  expect(String(err.message || '')).not.toBe('')
}

async function superAdmin(label = 'root') {
  const u = await registerVerified(label)
  dbQuery(`UPDATE users SET is_super_admin = TRUE WHERE id = ${u.auth.userId}`)
  const s = new Session(label + '-super')
  s.use()
  const a = useAuthStore()
  await a.login(u.login, u.password)
  return { session: s, auth: a, userId: u.auth.userId }
}

describeIntegration('ai API: состояние и компанийные настройки', () => {
  it('статус ИИ отвечает всегда — даже когда ничего не настроено', async () => {
    const u = await newCompanyAdmin()
    u.session.use()
    const status = await ai.getAiStatus()
    // Раздел не должен гадать: флаг обязан быть булевым и без ключа — false.
    expect(typeof (status.enabled ?? status.ai_enabled)).toBe('boolean')
    expect(status.enabled ?? status.ai_enabled).toBe(false)
  })

  it('настройки ИИ компании видит администратор, сотрудник — нет', async () => {
    const admin = await newCompanyAdmin('admin')
    const worker = await newMember(admin, admin.companyId, 1, 'worker')

    admin.session.use()
    const settings = await ai.getAiSettings(admin.companyId)
    expect(settings).toBeTruthy()

    worker.session.use()
    await expect(ai.getAiSettings(admin.companyId)).rejects.toBeTruthy()
  })

  it('настройки чужой компании недоступны', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newCompanyAdmin('b')
    b.session.use()
    await expect(ai.getAiSettings(a.companyId)).rejects.toBeTruthy()
    await expect(ai.updateAiSettings(a.companyId, { enabled: true })).rejects.toBeTruthy()
  })

  it('проверка связи и переиндексация без ключа отвечают понятной ошибкой', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    await expectClientError(ai.testAiSettings(admin.companyId))
    // Статус индексации — чтение, оно обязано работать всегда.
    expect(await ai.getAiIndexingStatus(admin.companyId)).toBeTruthy()
  })

  it('ТВ-факт без ИИ не роняет табло', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    // Факт — украшение: без ИИ сервер отдаёт пустоту, а не ошибку, иначе
    // табло мигало бы сообщением о сбое на каждом обороте слайдов.
    const res = await ai.getTvFact()
    expect(res == null || typeof res === 'object').toBe(true)
  })
})

describeIntegration('ai API: личный ключ', () => {
  it('личные настройки заводятся, ключ наружу не отдаётся', async () => {
    const u = await registerVerified()
    u.session.use()

    const before = await ai.getMyAiSettings()
    expect(before.has_key ?? false).toBe(false)

    await ai.updateMyAiSettings({ api_key: 'sk-очень-секретный-ключ-1234567890', enabled: true })
    const after = await ai.getMyAiSettings()
    expect(after.has_key ?? true).toBe(true)
    // Сырой ключ не должен возвращаться никогда — только маска.
    expect(JSON.stringify(after)).not.toContain('sk-очень-секретный-ключ-1234567890')
  })

  it('ключ у каждого свой и чужой не виден', async () => {
    const a = await registerVerified('a')
    a.session.use()
    await ai.updateMyAiSettings({ api_key: 'sk-ключ-пользователя-а-0000000000', enabled: true })

    const b = await registerVerified('b')
    b.session.use()
    const mine = await ai.getMyAiSettings()
    expect(mine.has_key ?? false).toBe(false)
  })

  it('проверка связи отдаёт диагностику, а не падение', async () => {
    const u = await registerVerified()
    u.session.use()
    await ai.updateMyAiSettings({ api_key: 'sk-неработающий-ключ-000000000000', enabled: true })

    // Кнопка «Проверить» обязана показать ПРИЧИНУ: ответ приходит с разбором
    // неудачи, а не ошибкой HTTP — иначе пользователь не поймёт, что не так.
    const res = await ai.testMyAiSettings()
    expect(res.chat).toBe(false)
    expect(String(res.error || '')).not.toBe('')
    expect(res.latency_ms ?? res.latencyMs ?? 0).toBeGreaterThanOrEqual(0)
  })

  it('без авторизации личные настройки закрыты', async () => {
    const guest = new Session('guest')
    guest.use()
    await expectStatus(ai.getMyAiSettings(), 401)
  })
})

describeIntegration('ai API: ассистент и инструменты текста', () => {
  it('история ассистента пуста у новичка и не требует компании', async () => {
    const u = await registerVerified()
    u.session.use()
    const history = await assistant.getAssistantHistory({ limit: 20 })
    expect((history.messages ?? history.items ?? []).length).toBe(0)
  })

  it('без рабочего ключа ассистент отвечает отказом, а не молчанием', async () => {
    const u = await registerVerified()
    u.session.use()
    await expectClientError(assistant.sendAssistantMessage('Сколько часов я отработал за неделю?'))
  })

  it('пустая реплика ассистенту не принимается', async () => {
    const u = await registerVerified()
    u.session.use()
    await expect(assistant.sendAssistantMessage('   ')).rejects.toBeTruthy()
  })

  it('голос за несуществующий ответ не принимается', async () => {
    const u = await registerVerified()
    u.session.use()
    await expect(assistant.sendAssistantFeedback({ messageId: 999999, verdict: 'up' })).rejects.toBeTruthy()
  })

  it('инструменты текста без ИИ дают понятный отказ', async () => {
    const u = await registerVerified()
    u.session.use()
    await expectClientError(ai.transformText({ action: 'improve', text: 'Текст для правки' }))
  })

  it('неизвестное действие над текстом отвергается', async () => {
    const u = await registerVerified()
    u.session.use()
    await expect(ai.transformText({ action: 'выдумка', text: 'Текст' })).rejects.toBeTruthy()
    await expect(ai.transformText({ action: 'improve', text: '' })).rejects.toBeTruthy()
  })

  it('корректура пустого набора сегментов не идёт к модели', async () => {
    const u = await registerVerified()
    u.session.use()
    const res = await ai.proofread([]).catch((e) => e)
    // Либо пустой ответ, либо 4xx — но не обращение к недоступной модели с 500.
    expect(res.status === undefined || res.status < 500).toBe(true)
  })
})

describeIntegration('ai API: платформенные настройки', () => {
  it('каталог моделей и ключ платформы — только супер-админу', async () => {
    const u = await newCompanyAdmin()
    u.session.use()
    await expectStatus(ai.getPlatformAi(), 403)
    await expectStatus(ai.updatePlatformAi({ api_key: 'sk-чужими-руками' }), 403)
    await expectStatus(ai.testPlatformAi(), 403)
  })

  it('супер-админ читает и правит платформенные настройки', async () => {
    const root = await superAdmin()
    root.session.use()
    const before = await ai.getPlatformAi()
    expect(before).toBeTruthy()

    await ai.updatePlatformAi({ base_url: 'https://api.example.invalid/v1' })
    const after = await ai.getPlatformAi()
    expect(after.base_url ?? after.settings?.base_url).toBe('https://api.example.invalid/v1')

    // Проверка связи с выдуманным адресом обязана вернуть ошибку, а не 500.
    await expectClientError(ai.testPlatformAi())
  })
})

describeIntegration('push API: реестр токенов устройств', () => {
  it('токен регистрируется, повтор не плодит записей, снятие убирает', async () => {
    const u = await registerVerified()
    u.session.use()
    const token = uniq('fcm-token-')

    await push.registerPushToken(token, 'android')
    await push.registerPushToken(token, 'android')
    const rows = dbQuery(`SELECT count(*) FROM device_tokens WHERE token = '${token}'`)
    expect(rows[0][0]).toBe('1')

    await push.unregisterPushToken(token)
    expect(dbQuery(`SELECT count(*) FROM device_tokens WHERE token = '${token}'`)[0][0]).toBe('0')
  })

  it('токен переезжает за пользователем, а не остаётся у прежнего', async () => {
    const a = await registerVerified('a')
    const token = uniq('fcm-shared-')
    a.session.use()
    await push.registerPushToken(token, 'android')

    // Одно устройство — сменился человек: пуши обязаны идти новому владельцу,
    // иначе прежний продолжит получать чужие уведомления.
    const b = await registerVerified('b')
    b.session.use()
    await push.registerPushToken(token, 'android')

    const owner = dbQuery(`SELECT user_id FROM device_tokens WHERE token = '${token}'`)
    expect(owner.length).toBe(1)
    expect(Number(owner[0][0])).toBe(b.auth.userId)
  })

  it('пустой токен не принимается', async () => {
    const u = await registerVerified()
    u.session.use()
    await expect(push.registerPushToken('', 'android')).rejects.toBeTruthy()
  })

  it('без авторизации токен не зарегистрировать', async () => {
    const guest = new Session('guest')
    guest.use()
    await expectStatus(push.registerPushToken('чужой-токен', 'android'), 401)
  })
})
