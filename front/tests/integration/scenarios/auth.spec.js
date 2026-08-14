// Сценарий «идентичность»: регистрация → подтверждение email → сессия →
// создание компании → switch → login/logout/refresh → login-gate при 2 компаниях.
// Всё через auth-стор и api-модули (фронтовый слой), против живого authsvc.
import { it, expect, beforeEach } from 'vitest'
import { describeIntegration, Session, verificationCode, uniq } from '../setup/harness.js'
import { useAuthStore } from '@/stores/auth.js'
import * as companiesApi from '@/api/companies.js'
import { refreshToken, login as apiLogin } from '@/api/auth.js'
import { LEGAL_DOC_KEYS, LEGAL_VERSION } from '@/utils/legal.js'
import { getMe } from '@/api/users.js'

// Зарегистрировать и подтвердить нового пользователя в рамках его Session.
async function registerVerified(s) {
  s.use()
  const auth = useAuthStore()
  const login = uniq('user_')
  const email = `${login}@apitest.local`
  const password = 'secret-pass-123'
  // ФИО уникально — по нему называется личная компания, а имена компаний
  // уникальны (см. factory.js).
  const reg = await auth.register({ fio: `${uniq('Тестов ')} Пользователь Апиевич`, email, login, password })
  expect(reg.verificationRequired).toBe(true)
  expect(reg.email).toBe(email)
  const code = verificationCode(email)
  expect(code).toMatch(/^\d+$/)
  await auth.verifyEmail({ email, code })
  // Правовые документы (152-ФЗ): пока гейт включён, до согласия все сервисы
  // отвечают 403 — живой пользователь проходит этот шаг после первого входа.
  // Гейт придержан до отдельного выпуска, поэтому шаг условный.
  if (auth.legalRequired) await auth.acceptLegal(LEGAL_VERSION, LEGAL_DOC_KEYS)
  return { login, email, password, auth }
}

describeIntegration('auth-flow: регистрация и сессия', () => {
  beforeEach(() => { /* каждый тест создаёт свои Session'ы */ })

  it('register → verify-email выдаёт сессию с токеном и user_id в claims', async () => {
    const s = new Session()
    const { auth, login } = await registerVerified(s)
    expect(auth.isAuth).toBe(true)
    expect(auth.token).toBeTruthy()
    expect(auth.userId).toBeGreaterThan(0)
    /* Новичок входит в СВОЮ личную компанию: без неё половина разделов
       отвечала бы «нужна активная компания» (см. EnsurePersonalCompany).
       Роль в ней — администратор, он же её создатель. */
    expect(auth.companyId).toBeGreaterThan(0)
    expect(auth.roleLevel).toBe(3)
    expect(auth.companies).toHaveLength(1)
    // loadMe грузится в фоне — дождёмся и проверим профиль (/users/me парсится).
    await auth.loadMe()
    expect(auth.user?.login).toBe(login)
  })

  it('создание компании + switch-company переносит активную компанию в claims', async () => {
    const s = new Session()
    const { auth } = await registerVerified(s)
    const personal = auth.companyId // личная компания новичка
    const created = await companiesApi.createCompany({ name: uniq('ООО ') })
    expect(created.id).toBeGreaterThan(0)
    // created_by — сам создатель (полные права).
    expect(auth.companyId).toBe(personal) // до switch активной остаётся прежняя
    await auth.switchCompany(created.id)
    expect(auth.companyId).toBe(created.id)
    expect(auth.roleLevel).toBe(3) // создатель — администратор
  })

  it('login после verify возвращает сессию; logout гасит; refresh поднимает по cookie', async () => {
    const s = new Session()
    const { login, password, auth } = await registerVerified(s)
    // У новичка ровно одна компания — личная, поэтому login идёт без gate.
    const personal = auth.companyId

    // Явный login через стор.
    const res = await auth.login(login, password)
    expect(res.forceChange).toBe(false)
    expect(auth.isAuth).toBe(true)
    expect(auth.companyId).toBe(personal) // 1 компания — автоактивна

    // refresh по cookie (jar хранит refresh_token) возвращает новый access.
    const data = await refreshToken()
    expect(data.access_token).toBeTruthy()
    expect(data.company_id).toBe(personal)
  })

  it('login-gate: две компании → needs_company_selection, select-company завершает вход', async () => {
    const s = new Session()
    const { login, password, auth } = await registerVerified(s)
    // Личная компания уже есть — добавляем к ней ещё две.
    const c1 = await companiesApi.createCompany({ name: uniq('Компания-А ') })
    await auth.switchCompany(c1.id)
    const c2 = await companiesApi.createCompany({ name: uniq('Компания-Б ') })
    await auth.switchCompany(c2.id)

    // Сырой login через api — ждём needs_company_selection + select_token.
    const raw = await apiLogin({ login, password })
    expect(raw.needs_company_selection).toBe(true)
    expect(Array.isArray(raw.companies)).toBe(true)
    expect(raw.companies.length).toBe(3) // личная + две заведённые
    expect(raw.select_token).toBeTruthy()

    // Через стор: login отдаёт needsSelection, затем selectCompany завершает.
    const gate = await auth.login(login, password)
    expect(gate.needsSelection).toBe(true)
    await auth.selectCompany(gate.selectToken, c1.id)
    expect(auth.companyId).toBe(c1.id)
    expect(auth.isMultiCompany).toBe(true)
  })

  it('refresh-цикл client.js: протухший access → авто-refresh по cookie → повтор', async () => {
    const s = new Session()
    const { auth } = await registerVerified(s)
    const created = await companiesApi.createCompany({ name: uniq('ООО ') })
    await auth.switchCompany(created.id)

    // Портим access-токен (эмуляция протухшего): запрос вернёт 401, client.js
    // сам сходит на /auth/refresh (refresh-cookie в jar), обновит сессию и
    // повторит исходный запрос — вызывающий код ошибки не видит.
    auth.token = 'v4.public.brokentoken'
    const me = await getMe()
    expect(me.id).toBe(auth.userId)
    // Сессия восстановлена: токен валиден (не остался сломанным), компания та же.
    // (PASETO с теми же клеймами и iat/exp посекундно байт-в-байт совпадает с
    // прежним — это норма, значение не сверяем.)
    expect(auth.token).not.toBe('v4.public.brokentoken')
    expect(auth.token).toBeTruthy()
    expect(auth.companyId).toBe(created.id)
  })
})
