/* Экран блокировки против живого authsvc.

   Проверяем то, ради чего он и заведён: снять экран можно пином или паролем
   (пин забывается чаще), чужой код не подходит, а убрать блокировку без
   секрета нельзя — иначе её отключил бы любой, кто подошёл к запертому
   экрану. Сама сессия при этом остаётся живой: блокировка закрывает вид, а не
   выходит из аккаунта. */
import { it, expect } from 'vitest'
import { describeIntegration } from '../setup/harness.js'
import { registerVerified } from '../setup/factory.js'
import {
  getScreenLock, setScreenLock, disableScreenLock, unlockScreen, listSessions,
} from '@/api/auth.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

describeIntegration('auth API: экран блокировки', () => {
  it('пин включается, снимает экран и убирается вместе с блокировкой', async () => {
    const u = await registerVerified()
    u.session.use()

    expect((await getScreenLock()).enabled).toBe(false)

    const state = await setScreenLock({ pin: '2468', after_min: 5 })
    expect(state.enabled).toBe(true)
    expect(state.after_min).toBe(5)

    await unlockScreen('2468')
    // Пароль аккаунта — запасной путь, если пин забыт.
    await unlockScreen(u.password)
    await expectStatus(unlockScreen('1111'), 403)

    // Сессия жива: блокировка не выходит из аккаунта.
    expect(Array.isArray(await listSessions())).toBe(true)

    await expectStatus(disableScreenLock('1111'), 403)
    await disableScreenLock('2468')
    expect((await getScreenLock()).enabled).toBe(false)
  })

  it('короткий и нецифровой пин не принимается', async () => {
    const u = await registerVerified()
    u.session.use()

    await expectStatus(setScreenLock({ pin: '12' }), 400)
    await expectStatus(setScreenLock({ pin: 'abcd' }), 400)
    // Включить блокировку без пина нельзя: снимать её было бы нечем.
    await expectStatus(setScreenLock({ after_min: 5 }), 400)
  })
})
