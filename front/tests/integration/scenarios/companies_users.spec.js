/* Компании, участники и права (authsvc).

   Здесь проверяются самые дорогие правила платформы: кто кем управляет.
   По CLAUDE.md участниками и ролями распоряжается СОЗДАТЕЛЬ компании (или
   супер-админ), роль нельзя выдать выше своей, а последнего администратора
   компании нельзя ни разжаловать, ни удалить. Ошибка в любом из этих правил —
   это либо потеря доступа к компании, либо чужие руки в чужих данных. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin, newMember } from '../setup/factory.js'
import * as companies from '@/api/companies.js'
import * as users from '@/api/users.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

const ROLE_EMPLOYEE = 1
const ROLE_MANAGER = 2
const ROLE_ADMIN = 3

describeIntegration('companies API: компания и её настройки', () => {
  it('создание компании делает создателя администратором', async () => {
    const u = await registerVerified()
    u.session.use()
    const created = await companies.createCompany({ name: uniq('ООО ') })
    expect(created.id).toBeGreaterThan(0)

    const mine = await companies.listMyCompanies()
    expect((mine.items ?? []).some((c) => c.id === created.id)).toBe(true)

    await u.auth.switchCompany(created.id)
    const members = await companies.listCompanyMembers(created.id)
    const me = members.find((m) => m.id === u.auth.userId)
    expect(me.role?.level).toBe(ROLE_ADMIN)
  })

  it('чужая компания не читается и не правится', async () => {
    const a = await newCompanyAdmin('a')
    const b = await registerVerified('b')

    b.session.use()
    await expect(companies.getCompany(a.companyId)).rejects.toBeTruthy()
    await expect(companies.updateCompany(a.companyId, { name: 'Захвачено' })).rejects.toBeTruthy()
    await expect(companies.listCompanyMembers(a.companyId)).rejects.toBeTruthy()
  })

  it('настройки выходных и питомцев меняет администратор компании', async () => {
    const admin = await newCompanyAdmin('admin')
    const worker = await newMember(admin, admin.companyId, ROLE_EMPLOYEE, 'worker')

    admin.session.use()
    await companies.updateWeekendSettings(admin.companyId, [6, 0])
    const w = await companies.getWeekendSettings(admin.companyId)
    expect(w.weekend_days ?? w.days).toEqual(expect.arrayContaining([6, 0]))

    await companies.updateGrooveSettings(admin.companyId, false)
    const g = await companies.getGrooveSettings(admin.companyId)
    expect(g.enabled ?? g.uses_groove).toBe(false)

    // Рядовому сотруднику настройки компании недоступны.
    worker.session.use()
    await expect(companies.updateWeekendSettings(admin.companyId, [0])).rejects.toBeTruthy()
  })
})

describeIntegration('companies API: участники и роли', () => {
  it('создатель добавляет участника и меняет ему роль', async () => {
    const owner = await newCompanyAdmin('owner')
    const member = await newMember(owner, owner.companyId, ROLE_EMPLOYEE, 'member')

    owner.session.use()
    await companies.setMemberRole(owner.companyId, member.auth.userId, ROLE_MANAGER)
    const members = await companies.listCompanyMembers(owner.companyId)
    expect(members.find((m) => m.id === member.auth.userId)?.role?.level).toBe(ROLE_MANAGER)
  })

  it('сотрудник не управляет составом компании', async () => {
    const owner = await newCompanyAdmin('owner')
    const worker = await newMember(owner, owner.companyId, ROLE_EMPLOYEE, 'worker')
    const other = await newMember(owner, owner.companyId, ROLE_EMPLOYEE, 'other')

    worker.session.use()
    await expect(companies.setMemberRole(owner.companyId, other.auth.userId, ROLE_ADMIN)).rejects.toBeTruthy()
    await expect(companies.removeCompanyMember(owner.companyId, other.auth.userId)).rejects.toBeTruthy()
  })

  it('администратор-не-создатель составом не управляет', async () => {
    const owner = await newCompanyAdmin('owner')
    const admin2 = await newMember(owner, owner.companyId, ROLE_ADMIN, 'admin2')
    const worker = await newMember(owner, owner.companyId, ROLE_EMPLOYEE, 'worker')

    // Правило creatorAuthority: участниками распоряжается создатель компании.
    admin2.session.use()
    await expect(companies.removeCompanyMember(owner.companyId, worker.auth.userId)).rejects.toBeTruthy()
  })

  it('последнего администратора нельзя разжаловать или убрать', async () => {
    const owner = await newCompanyAdmin('owner')
    owner.session.use()

    // Создатель — единственный администратор: понижение отняло бы у компании
    // управление насовсем.
    await expect(companies.setMemberRole(owner.companyId, owner.auth.userId, ROLE_EMPLOYEE)).rejects.toBeTruthy()
    await expect(companies.removeCompanyMember(owner.companyId, owner.auth.userId)).rejects.toBeTruthy()
  })

  it('участник видит справочник своей компании и не видит чужой', async () => {
    const a = await newCompanyAdmin('a')
    const worker = await newMember(a, a.companyId, ROLE_EMPLOYEE, 'worker')
    const b = await newCompanyAdmin('b')

    worker.session.use()
    const dir = await users.getDirectory()
    expect(dir.some((u) => u.id === a.auth.userId)).toBe(true)
    expect(dir.some((u) => u.id === b.auth.userId)).toBe(false)
  })

  it('ссылка-приглашение вводит человека сотрудником, перевыпуск гасит старую', async () => {
    const owner = await newCompanyAdmin('owner')
    owner.session.use()
    // Ссылки может не быть вовсе: код выпускается по требованию.
    const invite = await companies.regenerateCompanyInvite(owner.companyId)
    expect(invite.code).toBeTruthy()
    expect((await companies.getCompanyInvite(owner.companyId)).code).toBe(invite.code)

    const guest = await registerVerified('guest')
    guest.session.use()
    await companies.joinCompanyByCode(invite.code)
    // Вошедший — рядовой сотрудник, поэтому «мои компании» (там, где он
    // администратор) его новую компанию не покажут: проверяем по составу.
    owner.session.use()
    const members = await companies.listCompanyMembers(owner.companyId)
    expect(members.some((m) => m.id === guest.auth.userId)).toBe(true)

    owner.session.use()
    const next = await companies.regenerateCompanyInvite(owner.companyId)
    expect(next.code).not.toBe(invite.code)

    const late = await registerVerified('late')
    late.session.use()
    await expect(companies.joinCompanyByCode(invite.code)).rejects.toBeTruthy()
  })

  it('повторный вход по коду не задваивает членство', async () => {
    const owner = await newCompanyAdmin('owner')
    owner.session.use()
    const invite = await companies.regenerateCompanyInvite(owner.companyId)

    const guest = await registerVerified('guest')
    guest.session.use()
    await companies.joinCompanyByCode(invite.code)
    // Повторный вход — не ошибка сервера: членство уже есть, состав не двоится.
    await companies.joinCompanyByCode(invite.code)

    owner.session.use()
    const members = await companies.listCompanyMembers(owner.companyId)
    expect(members.filter((m) => m.id === guest.auth.userId).length).toBe(1)
  })
})

describeIntegration('users API: профиль и доступ', () => {
  it('свой профиль читается и правится', async () => {
    const u = await registerVerified()
    u.session.use()
    const me = await users.getMe()
    expect(me.id).toBe(u.auth.userId)

    await users.updateMe({ fio: 'Изменённый Пользователь Тестович' })
    expect((await users.getMe()).fio).toBe('Изменённый Пользователь Тестович')
  })

  it('настройки рабочего стола переживают перезапись и непрозрачны для сервера', async () => {
    const u = await registerVerified()
    u.session.use()
    await users.saveDesktopPrefs({ pinned: ['tasks', 'notes'], liveTiles: false })
    const saved = await users.getDesktopPrefs()
    expect(saved.pinned ?? saved.prefs?.pinned).toEqual(['tasks', 'notes'])

    await users.saveDesktopPrefs({ pinned: [], liveTiles: true })
    const after = await users.getDesktopPrefs()
    expect(after.pinned ?? after.prefs?.pinned).toEqual([])
  })

  it('список всех пользователей платформы — только супер-админу', async () => {
    const u = await newCompanyAdmin()
    u.session.use()
    // Администратор компании — не супер-админ: платформенный список ему закрыт.
    await expectStatus(users.getUsers(), 403)
  })

  it('чужой профиль напрямую не правится', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, ROLE_EMPLOYEE, 'b')

    b.session.use()
    await expect(users.updateUser(a.auth.userId, { fio: 'Переименован силой' })).rejects.toBeTruthy()
    await expect(users.deleteUser(a.auth.userId)).rejects.toBeTruthy()
  })

  it('карточка постороннего отдаётся без контактов, карточка коллеги — с ними', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, ROLE_EMPLOYEE, 'b')
    const outsider = await newCompanyAdmin('outsider')

    b.session.use()
    // Коллега по компании — контакты видны (по ним пишут и звонят).
    const mate = await users.getDirectoryUser(a.auth.userId)
    expect(mate.id).toBe(a.auth.userId)
    expect(mate.email).toBeTruthy()

    // Посторонний виден по имени (его показывают у общих сущностей), но его
    // телефон и почта — PII: по числовому id их выдавать нельзя.
    const stranger = await users.getDirectoryUser(outsider.auth.userId)
    expect(stranger.id).toBe(outsider.auth.userId)
    expect(stranger.email ?? null).toBeNull()
    expect(stranger.phone ?? null).toBeNull()
  })

  it('поиск по справочнику фильтрует, а пустой запрос отдаёт всех своих', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, ROLE_EMPLOYEE, 'b')

    a.session.use()
    const all = await users.getDirectory()
    expect(all.length).toBeGreaterThanOrEqual(2)

    const found = await users.getDirectory(b.login, false, { byLogin: true })
    expect(found.some((u) => u.id === b.auth.userId)).toBe(true)

    const none = await users.getDirectory('заведомо-небывалый-логин')
    expect(none.length).toBe(0)
  })
})
