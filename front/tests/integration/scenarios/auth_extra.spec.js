/* Авторизация, часть вторая: реестр сессий, сброс пароля, обязательная смена
   дефолтного пароля, брутфорс-щит, подсказка логина и email-приглашения.

   Здесь проверяются правила, ценой ошибки в которых становится чужой доступ:
   отозванная сессия обязана перестать обновляться, «забыл пароль» не должен
   раскрывать существование аккаунта, а человек с дефолтным паролем не должен
   ходить по API до его смены. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq, dbQuery, verificationCode, Session } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin } from '../setup/factory.js'
import { useAuthStore } from '@/stores/auth.js'
import * as auth from '@/api/auth.js'
import * as companies from '@/api/companies.js'
import * as users from '@/api/users.js'
import { LEGAL_DOC_KEYS, LEGAL_VERSION } from '@/utils/legal.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

function resetToken(email) {
  const rows = dbQuery(`SELECT p.token FROM password_resets p
    JOIN users u ON u.id = p.user_id WHERE lower(u.email) = lower('${email}')
    ORDER BY p.id DESC LIMIT 1`)
  return rows[0]?.[0] || ''
}

describeIntegration('auth API: реестр сессий и устройств', () => {
  it('вход заводит карточку сессии, текущая помечена', async () => {
    const u = await registerVerified()
    u.session.use()
    const items = await auth.listSessions()
    expect(items.length).toBeGreaterThan(0)
    expect(items.some((s) => s.current)).toBe(true)
  })

  it('отозванная сессия перестаёт обновляться', async () => {
    const u = await registerVerified()

    // Второй вход того же человека — отдельная карточка со своей cookie.
    const other = new Session('second')
    other.use()
    const otherAuth = useAuthStore()
    await otherAuth.login(u.login, u.password)
    expect(otherAuth.isAuth).toBe(true)

    u.session.use()
    const items = await auth.listSessions()
    const foreign = items.find((s) => !s.current)
    expect(foreign).toBeTruthy()
    await auth.revokeSession(foreign.id)

    // Отозванный вход не поднимается по своей cookie — это и есть «выкинуть
    // устройство»: иначе отзыв был бы косметикой.
    other.use()
    await expect(auth.refreshToken()).rejects.toMatchObject({ status: 401 })
  })

  it('чужую сессию отозвать нельзя', async () => {
    const a = await registerVerified('a')
    a.session.use()
    const mine = await auth.listSessions()
    const myId = mine[0].id

    const b = await registerVerified('b')
    b.session.use()
    // Чужой идентификатор просто «не находится»: существование чужих сессий не
    // раскрываем.
    await expect(auth.revokeSession(myId)).rejects.toBeTruthy()

    a.session.use()
    expect((await auth.listSessions()).some((s) => s.id === myId)).toBe(true)
  })

  it('выход гасит только свой сеанс', async () => {
    const u = await registerVerified()
    u.session.use()
    const before = (await auth.listSessions()).length

    const second = new Session('second')
    second.use()
    const secondAuth = useAuthStore()
    await secondAuth.login(u.login, u.password)
    await secondAuth.logout()

    u.session.use()
    const after = (await auth.listSessions()).length
    expect(after).toBe(before)
    // Своя сессия жива — по ней всё ещё ходим.
    expect((await users.getMe()).id).toBe(u.auth.userId)
  })
})

describeIntegration('auth API: восстановление доступа', () => {
  it('«забыл пароль» не раскрывает, есть ли такой аккаунт', async () => {
    const u = await registerVerified()
    const known = await auth.forgotPassword(u.email)
    const unknown = await auth.forgotPassword('no-such-user@apitest.local')
    // Ответы неотличимы — иначе форма превращается в перебор адресов.
    expect(known.status ?? known.ok ?? true).toEqual(unknown.status ?? unknown.ok ?? true)
  })

  it('сброс по токену меняет пароль, повторный токен не работает', async () => {
    const u = await registerVerified()
    await auth.forgotPassword(u.email)
    const token = resetToken(u.email)
    expect(token).toBeTruthy()

    const done = await auth.resetPassword(token, 'new-secret-456')
    expect(done.login).toBe(u.login)

    // Одноразовость: тем же письмом пароль второй раз не меняют.
    await expect(auth.resetPassword(token, 'again-789')).rejects.toBeTruthy()

    // Старый пароль больше не пускает, новый — пускает.
    const s = new Session('after-reset')
    s.use()
    const a = useAuthStore()
    await expect(a.login(u.login, u.password)).rejects.toBeTruthy()
    await a.login(u.login, 'new-secret-456')
    expect(a.isAuth).toBe(true)
  })

  it('выдуманный токен сброса отвергается', async () => {
    await expect(auth.resetPassword('не-существующий-токен', 'какой-то-пароль')).rejects.toBeTruthy()
  })

  it('короткий пароль не принимается', async () => {
    const u = await registerVerified()
    await auth.forgotPassword(u.email)
    const token = resetToken(u.email)
    await expect(auth.resetPassword(token, '123')).rejects.toBeTruthy()
  })
})

describeIntegration('auth API: дефолтный пароль сотрудника', () => {
  it('заведённый администратором ходит по API только после смены пароля', async () => {
    const admin = await newCompanyAdmin('admin')
    admin.session.use()
    const login = uniq('emp_')
    await companies.createCompanyUser(admin.companyId, {
      fio: 'Сотрудников Сотрудник Сотрудникович',
      login,
      email: `${login}@apitest.local`,
      role_id: 1,
    })

    const s = new Session('employee')
    s.use()
    const a = useAuthStore()
    const session = await a.login(login, `${login}123`)
    expect(session.forceChange ?? a.claims?.force_change).toBeTruthy()

    // До смены пароля API закрыт целиком — кроме самой смены и выхода.
    await expectStatus(users.getMe(), 403)

    await a.changeDefaultCredentials({ password: 'own-secret-123', confirmPassword: 'own-secret-123' })
    /* Гейты идут по очереди: сменил пароль — упёрся в правовые документы.
       Сотрудник, заведённый администратором, принимает их сам: согласие даёт
       субъект персональных данных, а не работодатель за него. Гейт придержан
       до отдельного выпуска (domain.LegalGateEnabled) — при снятом флаге шага
       просто нет. */
    if (a.legalRequired) {
      await expectStatus(users.getMe(), 403)
      await a.acceptLegal(LEGAL_VERSION, LEGAL_DOC_KEYS)
    }
    expect((await users.getMe()).login).toBe(login)
  })

  it('сотрудник, заведённый администратором, считается подтверждённым', async () => {
    const admin = await newCompanyAdmin('admin')
    admin.session.use()
    const login = uniq('emp_')
    await companies.createCompanyUser(admin.companyId, {
      fio: 'Проверкин Проверка Проверкович',
      login,
      email: `${login}@apitest.local`,
      role_id: 1,
    })
    const rows = dbQuery(`SELECT email_verified FROM users WHERE login = '${login}'`)
    expect(rows[0]?.[0]).toBe('t')
  })
})

describeIntegration('auth API: щит от подбора и мелочи входа', () => {
  it('серия неудач блокирует вход на время', async () => {
    const u = await registerVerified()
    const s = new Session('bruteforce')
    s.use()
    const a = useAuthStore()

    let blocked = null
    for (let i = 0; i < 7 && !blocked; i++) {
      try { await a.login(u.login, 'заведомо-неверный') } catch (e) { if (e.status === 429) blocked = e }
    }
    expect(blocked).toBeTruthy()
    expect(blocked.data?.retry_after_sec ?? blocked.retry_after_sec ?? 1).toBeGreaterThan(0)
  })

  it('подсказка логина транслитерирует ФИО и не предлагает занятое', async () => {
    const first = await auth.suggestLogin('Иванов Иван Иванович')
    expect(first.login).toMatch(/^[a-z0-9._-]+$/i)

    const u = await registerVerified()
    const busy = await auth.suggestLogin('Иванов Иван Иванович')
    expect(busy.login).not.toBe(u.login)
  })

  it('логин неподтверждённого отвергается с понятным кодом', async () => {
    const s = new Session('unverified')
    s.use()
    const a = useAuthStore()
    const login = uniq('unv_')
    const email = `${login}@apitest.local`
    await a.register({ fio: 'Неподтверждённый Пользователь Тестович', email, login, password: 'secret-pass-123' })

    await expect(a.login(login, 'secret-pass-123')).rejects.toMatchObject({ status: 403 })
  })

  it('повторная регистрация занятого логина/почты не проходит', async () => {
    const u = await registerVerified()
    const s = new Session('dup')
    s.use()
    const a = useAuthStore()
    await expect(a.register({
      fio: 'Дубликат Дубликатов Дубликатович',
      email: `other-${u.login}@apitest.local`,
      login: u.login,
      password: 'secret-pass-123',
    })).rejects.toBeTruthy()
    await expect(a.register({
      fio: 'Дубликат Дубликатов Дубликатович',
      email: u.email,
      login: uniq('other_'),
      password: 'secret-pass-123',
    })).rejects.toBeTruthy()
  })

  it('подтверждение неверным кодом не выдаёт сессию', async () => {
    const s = new Session('badcode')
    s.use()
    const a = useAuthStore()
    const login = uniq('bad_')
    const email = `${login}@apitest.local`
    await a.register({ fio: 'Кодов Код Кодович', email, login, password: 'secret-pass-123' })

    await expect(a.verifyEmail({ email, code: '000000' })).rejects.toBeTruthy()
    expect(a.isAuth).toBe(false)

    // Настоящий код по-прежнему работает — неудача не сожгла попытку.
    await a.verifyEmail({ email, code: verificationCode(email) })
    expect(a.isAuth).toBe(true)
  })
})

describeIntegration('auth API: email-приглашения в компанию', () => {
  it('приглашение приводит человека с заданной ролью', async () => {
    const owner = await newCompanyAdmin('owner')
    owner.session.use()
    const invitee = `inv-${uniq('')}@apitest.local`
    await companies.createCompanyInvite(owner.companyId, invitee, 2)

    const rows = dbQuery(`SELECT token FROM company_invites
      WHERE lower(email) = lower('${invitee}') ORDER BY id DESC LIMIT 1`)
    const token = rows[0]?.[0]
    expect(token).toBeTruthy()

    const invited = await registerVerified('invited')
    invited.session.use()
    await invited.auth.acceptInvite(token)

    owner.session.use()
    const members = await companies.listCompanyMembers(owner.companyId)
    const added = members.find((m) => m.id === invited.auth.userId)
    expect(added?.role?.level).toBe(2)
  })

  it('приглашение одноразовое и выдуманный токен не принимается', async () => {
    const owner = await newCompanyAdmin('owner')
    owner.session.use()
    const invitee = `inv-${uniq('')}@apitest.local`
    await companies.createCompanyInvite(owner.companyId, invitee, 1)
    const token = dbQuery(`SELECT token FROM company_invites
      WHERE lower(email) = lower('${invitee}') ORDER BY id DESC LIMIT 1`)[0][0]

    const first = await registerVerified('inv1')
    first.session.use()
    await first.auth.acceptInvite(token)

    const second = await registerVerified('inv2')
    second.session.use()
    await expect(second.auth.acceptInvite(token)).rejects.toBeTruthy()
    await expect(second.auth.acceptInvite('выдуманный-токен')).rejects.toBeTruthy()
  })

  it('приглашать в чужую компанию нельзя', async () => {
    const owner = await newCompanyAdmin('owner')
    const stranger = await newCompanyAdmin('stranger')
    stranger.session.use()
    await expect(companies.createCompanyInvite(
      owner.companyId, `sneak-${uniq('')}@apitest.local`, 3,
    )).rejects.toBeTruthy()
  })
})
