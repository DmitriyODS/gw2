// Фабрики тестовых пользователей/компаний поверх фронтовых сторов/api.
import { expect } from 'vitest'
import { Session, verificationCode, uniq } from './harness.js'
import { useAuthStore } from '@/stores/auth.js'
import * as companiesApi from '@/api/companies.js'

// registerVerified — новый Session с подтверждённым пользователем и активной
// сессией (без компании). Возвращает { session, auth, login, email, password }.
export async function registerVerified(label = '') {
  const session = new Session(label)
  session.use()
  const auth = useAuthStore()
  const login = uniq('user_')
  const email = `${login}@apitest.local`
  const password = 'secret-pass-123'
  const reg = await auth.register({ fio: 'Тестов Пользователь Апиевич', email, login, password })
  expect(reg.verificationRequired).toBe(true)
  const code = verificationCode(email)
  await auth.verifyEmail({ email, code })
  expect(auth.isAuth).toBe(true)
  await auth.loadMe()
  return { session, auth, login, email, password }
}

// newCompanyAdmin — пользователь + собственная компания, сессия переключена
// на неё (роль администратор). Возвращает { session, auth, companyId, ... }.
export async function newCompanyAdmin(label = '') {
  const u = await registerVerified(label)
  const created = await companiesApi.createCompany({ name: uniq('ООО ') })
  await u.auth.switchCompany(created.id)
  return { ...u, companyId: created.id }
}

// addMemberToCompany — включить существующего пользователя в компанию с ролью
// (действует создатель) и переключить сессию адресата на неё.
export async function addMemberToCompany(creator, companyId, member, roleId = 1) {
  creator.session.use()
  await companiesApi.addCompanyMember(companyId, member.auth.userId, roleId)
  member.session.use()
  await member.auth.switchCompany(companyId)
}

// newMember — новый пользователь, включённый в компанию создателя с ролью.
export async function newMember(creator, companyId, roleId = 1, label = '') {
  const m = await registerVerified(label)
  await addMemberToCompany(creator, companyId, m, roleId)
  return m
}
