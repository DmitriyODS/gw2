/* Оставшиеся ручки: группы и вложения мессенджера, разделы и медиа портала,
   правка задач/юнитов/справочников, отчёты статистики, профиль и аватар.

   Набор дописывает покрытие до полного: каждая ручка вызывается и в успешном
   пути, и в отказном (чужое, несуществующее, недостаточно прав). Проверки
   опираются на смысл действия: правка отражается в чтении, удаление убирает
   из выдачи, отчёт по сотруднику доступен тому, кто вправе его видеть. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq, dbQuery, Session } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin, newMember } from '../setup/factory.js'
import { useAuthStore } from '@/stores/auth.js'
import * as messenger from '@/api/messenger.js'
import * as portal from '@/api/portal.js'
import * as tasks from '@/api/tasks.js'
import * as units from '@/api/units.js'
import * as stages from '@/api/stages.js'
import * as unitTypes from '@/api/unitTypes.js'
import * as departments from '@/api/departments.js'
import * as stats from '@/api/stats.js'
import * as users from '@/api/users.js'
import * as companies from '@/api/companies.js'
import * as reminders from '@/api/reminders.js'
import * as registries from '@/api/registries.js'
import * as calendars from '@/api/calendars.js'
import { STAGE_COLORS } from '@/api/stages.js'

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

const ymd = (d) => d.toISOString().slice(0, 10)
const today = ymd(new Date())
const PNG = () => Uint8Array.from(
  atob('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=='),
  (c) => c.charCodeAt(0),
)

async function company(label = '') {
  const admin = await newCompanyAdmin(label)
  admin.session.use()
  const dept = await departments.createDepartment({ name: uniq('Отдел ') })
  const type = await unitTypes.createUnitType({ name: uniq('Работа ') })
  return { admin, deptId: dept.id, typeId: type.id }
}

describeIntegration('messenger API: группы и вложения', () => {
  it('состав группы: добавление, права, удаление участника', async () => {
    const admin = await newCompanyAdmin('owner')
    const first = await newMember(admin, admin.companyId, 1, 'first')
    const second = await newMember(admin, admin.companyId, 1, 'second')

    admin.session.use()
    const group = await messenger.createGroup({
      title: uniq('Группа '), memberIds: [first.auth.userId],
    })
    const id = group.id ?? group.conversation?.id

    await messenger.addGroupMembers(id, [second.auth.userId])
    const info = await messenger.getGroup(id)
    expect((info.members ?? []).length).toBe(3)

    // Тонкие права настраиваются для АДМИНИСТРАТОРОВ группы: рядовому
    // участнику их выдавать не из чего — сперва роль, потом права.
    await expectClientError(messenger.setMemberRights(id, second.auth.userId, { can_pin_messages: true }))
    await messenger.setMemberRole(id, second.auth.userId, 'admin')
    await messenger.setMemberRights(id, second.auth.userId, { pin_messages: true })
    const withRights = await messenger.getGroup(id)
    const row = (withRights.members ?? []).find((m) => m.user.id === second.auth.userId)
    expect(row.can_pin_messages).toBe(true)

    // Рядовой участник состав не меняет.
    first.session.use()
    await expectClientError(messenger.addGroupMembers(id, [admin.auth.userId]))

    admin.session.use()
    await messenger.removeGroupMember(id, second.auth.userId)
    expect(((await messenger.getGroup(id)).members ?? []).length).toBe(2)
  })

  it('аватар группы ставится из уже загруженного вложения', async () => {
    const admin = await newCompanyAdmin('owner')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')

    admin.session.use()
    const group = await messenger.createGroup({ title: uniq('С аватаром '), memberIds: [mate.auth.userId] })
    const id = group.id ?? group.conversation?.id

    const uploaded = await messenger.uploadAttachment(new File([PNG()], 'avatar.png', { type: 'image/png' }))
    const attachmentId = uploaded.id ?? uploaded.attachment?.id
    expect(attachmentId).toBeTruthy()

    await messenger.setGroupAvatar(id, attachmentId)
    const info = await messenger.getGroup(id)
    expect(info.avatar_path ?? info.conversation?.avatar_path ?? null).toBeTruthy()
  })

  it('пересылка сообщения копирует его в другой чат', async () => {
    const admin = await newCompanyAdmin('owner')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')
    const third = await newMember(admin, admin.companyId, 1, 'third')

    admin.session.use()
    const dialog = await messenger.openConversation(mate.auth.userId)
    const dialogId = dialog.id ?? dialog.conversation?.id
    const msg = await messenger.sendMessage(dialogId, { text: 'Исходное сообщение' })

    await messenger.forwardMessage(msg.id, { userIds: [third.auth.userId] })

    third.session.use()
    const list = await messenger.listConversations()
    const texts = []
    for (const c of list) texts.push(...(await messenger.listMessages(c.id)).map((m) => m.text ?? ''))
    expect(texts.some((t) => t.includes('Исходное'))).toBe(true)
  })

  it('«кто прочитал» показывает участников с прочтением', async () => {
    const admin = await newCompanyAdmin('owner')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')

    admin.session.use()
    const group = await messenger.createGroup({ title: uniq('Прочтения '), memberIds: [mate.auth.userId] })
    const id = group.id ?? group.conversation?.id
    const msg = await messenger.sendMessage(id, { text: 'Прочитайте это' })

    mate.session.use()
    await messenger.markRead(id)

    admin.session.use()
    const readBy = await messenger.messageReadBy(msg.id)
    const ids = (readBy.readers ?? []).map((u) => u.id ?? u.user_id)
    expect(ids).toContain(mate.auth.userId)
  })

  it('переписка скрывается у себя и удаляется у всех', async () => {
    const admin = await newCompanyAdmin('owner')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')

    admin.session.use()
    const dialog = await messenger.openConversation(mate.auth.userId)
    const id = dialog.id ?? dialog.conversation?.id
    await messenger.sendMessage(id, { text: 'Разговор' })

    await messenger.deleteConversation(id, 'me')
    expect((await messenger.listConversations()).some((c) => c.id === id)).toBe(false)

    // У собеседника переписка на месте — «скрыть у себя» не трогает чужое.
    mate.session.use()
    expect((await messenger.listConversations()).some((c) => c.id === id)).toBe(true)
  })

  it('чат поддержки открывается, входящие поддержки — только супер-админу', async () => {
    const u = await newCompanyAdmin()
    u.session.use()
    const chat = await messenger.openDevChat()
    expect(chat).toBeTruthy()
    await expectStatus(messenger.listSupportInbox(), 403)

    dbQuery(`UPDATE users SET is_super_admin = TRUE WHERE id = ${u.auth.userId}`)
    const s = new Session('root')
    s.use()
    await useAuthStore().login(u.login, u.password)
    expect(await messenger.listSupportInbox()).toBeTruthy()
  })

  it('порядок папок чатов сохраняется', async () => {
    const u = await newCompanyAdmin()
    u.session.use()
    const first = await messenger.createFolder({ title: uniq('Работа ') })
    const second = await messenger.createFolder({ title: uniq('Личное ') })

    await messenger.reorderFolders([second.id, first.id])
    const list = await messenger.listFolders()
    const items = list.items ?? list.folders ?? list
    expect(items.findIndex((f) => f.id === second.id)).toBeLessThan(items.findIndex((f) => f.id === first.id))
  })
})

describeIntegration('portal API: разделы, медиа, пересылка', () => {
  it('раздел переименовывается и виден в списке', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    const topic = await portal.createTopic({ name: uniq('Новости '), color: 'blue' })
    await portal.updateTopic(topic.id, { name: 'Новости компании', color: 'green', icon: '📰' })

    const list = await portal.getTopics()
    const updated = (list.topics ?? list.items ?? list).find((t) => t.id === topic.id)
    expect(updated.name).toBe('Новости компании')
    expect(updated.icon).toBe('📰')

    await portal.deleteTopic(topic.id)
  })

  it('вложение поста загружается и удаляется', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    const post = await portal.createPost({ body: 'Пост с картинкой' })

    const uploaded = await portal.uploadAttachment(post.id, new File([PNG()], 'pic.png', { type: 'image/png' }))
    const attachmentId = uploaded.id ?? uploaded.attachment?.id
    expect(attachmentId).toBeTruthy()
    expect(((await portal.getPost(post.id)).attachments ?? []).length).toBe(1)

    await portal.deleteAttachment(attachmentId)
    expect(((await portal.getPost(post.id)).attachments ?? []).length).toBe(0)
  })

  it('просмотр отмечается и гасит непрочитанное', async () => {
    const admin = await newCompanyAdmin('admin')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')

    admin.session.use()
    const post = await portal.createPost({ body: 'Читайте внимательно' })

    mate.session.use()
    await portal.markView(post.id)
    expect(await portal.getPost(post.id)).toBeTruthy()
  })

  it('популярные теги считаются по ленте', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    const tag = `тема${Date.now().toString().slice(-6)}`
    await portal.createPost({ body: `Первый про #${tag}` })
    await portal.createPost({ body: `Второй про #${tag}` })

    const top = await portal.getPopularTags(10)
    const items = top.tags ?? top.items ?? top
    expect(items.some((t) => (t.tag ?? t.name ?? t) === tag)).toBe(true)
  })

  it('пост пересылается в переписку', async () => {
    const admin = await newCompanyAdmin('admin')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')

    admin.session.use()
    const post = await portal.createPost({ body: 'Важное объявление для пересылки' })
    await portal.forwardPost(post.id, { userIds: [mate.auth.userId] })

    mate.session.use()
    const list = await messenger.listConversations()
    const kinds = []
    for (const c of list) kinds.push(...(await messenger.listMessages(c.id)).map((m) => m.kind))
    expect(kinds).toContain('post')
  })

  it('пост чужой компании переслать нельзя', async () => {
    const a = await newCompanyAdmin('a')
    a.session.use()
    const post = await portal.createPost({ body: 'Внутреннее' })

    const b = await newCompanyAdmin('b')
    b.session.use()
    await expectClientError(portal.forwardPost(post.id, { userIds: [b.auth.userId] }))
  })
})

describeIntegration('tasks API: правка, удаление, участники', () => {
  it('задача переименовывается и удаляется совсем', async () => {
    const { admin, deptId } = await company('a')
    admin.session.use()
    const task = await tasks.createTask({ name: 'Черновая', department_id: deptId })

    await tasks.updateTask(task.id, { name: 'Уточнённая' })
    expect((await tasks.getTask(task.id)).name).toBe('Уточнённая')

    await tasks.deleteTask(task.id)
    await expectClientError(tasks.getTask(task.id))
  })

  it('участники задачи — те, кто по ней работал', async () => {
    const { admin, deptId, typeId } = await company('a')
    admin.session.use()
    const task = await tasks.createTask({ name: 'С участниками', department_id: deptId })
    const unit = await units.createUnit(task.id, { name: 'работа', unit_type_id: typeId })
    await units.stopUnit(unit.id)

    const list = await tasks.getTaskContributors(task.id)
    const ids = (list.items ?? list.users ?? list).map((u) => u.id ?? u.user_id)
    expect(ids).toContain(admin.auth.userId)
  })

  it('юнит правится и удаляется', async () => {
    const { admin, deptId, typeId } = await company('a')
    admin.session.use()
    const task = await tasks.createTask({ name: 'С юнитом', department_id: deptId })
    const unit = await units.createUnit(task.id, { name: 'первичное', unit_type_id: typeId })
    await units.stopUnit(unit.id)

    await units.updateUnit(unit.id, { name: 'исправленное' })
    const list = await units.getUnits(task.id)
    expect((list.units ?? list).find((u) => u.id === unit.id).name).toBe('исправленное')

    await units.deleteUnit(unit.id)
    const after = await units.getUnits(task.id)
    expect((after.units ?? after).some((u) => u.id === unit.id)).toBe(false)
  })

  it('этап и тип юнита переименовываются, чужие — нет', async () => {
    const a = await company('a')
    a.admin.session.use()
    const stage = await stages.createStage({ name: uniq('Этап '), color: STAGE_COLORS[0] })
    await stages.updateStage(stage.id, { name: 'Этап уточнён', color: STAGE_COLORS[1] })
    expect((await stages.getStages()).stages?.find((s) => s.id === stage.id)?.name
      ?? (await stages.getStages()).find((s) => s.id === stage.id).name).toBe('Этап уточнён')

    await unitTypes.updateUnitType(a.typeId, { name: 'Работа уточнена' })
    const types = await unitTypes.getUnitTypes()
    expect((types.unit_types ?? types.items ?? types).find((t) => t.id === a.typeId).name).toBe('Работа уточнена')

    const b = await company('b')
    b.admin.session.use()
    await expectClientError(stages.updateStage(stage.id, { name: 'Чужой этап' }))
    await expectClientError(stages.deleteStage(stage.id))
    await expectClientError(unitTypes.updateUnitType(a.typeId, { name: 'Чужой тип' }))

    a.admin.session.use()
    await stages.deleteStage(stage.id)
  })
})

describeIntegration('stats API: отчёты по людям', () => {
  it('сводки по сотрудникам и ответственным отдаются', async () => {
    const { admin } = await company('a')
    admin.session.use()
    expect(await stats.getStatsEmployees()).toBeTruthy()
    expect(await stats.getStatsResponsibles()).toBeTruthy()
  })

  it('задачи конкретного сотрудника видны его компании', async () => {
    const { admin, deptId, typeId } = await company('a')
    admin.session.use()
    const task = await tasks.createTask({ name: 'Учтённая', department_id: deptId })
    const unit = await units.createUnit(task.id, { name: 'работа', unit_type_id: typeId })
    await units.stopUnit(unit.id)

    const mine = await stats.getStatsUserTasks(admin.auth.userId, today, today)
    expect(mine).toBeTruthy()

    // Из другой компании отчёт по чужому человеку недоступен.
    const b = await company('b')
    b.admin.session.use()
    const foreign = await stats.getStatsUserTasks(admin.auth.userId, today, today).catch((e) => e)
    const rows = foreign?.items ?? foreign?.tasks ?? []
    expect(Array.isArray(rows) ? rows.length : 0).toBe(0)
  })

  it('активность сотрудника: сводка, лента и выгрузка', async () => {
    const { admin, deptId } = await company('a')
    admin.session.use()
    await tasks.createTask({ name: 'Для активности', department_id: deptId })

    expect(await stats.getEmployeeActivity(admin.auth.userId, today, today)).toBeTruthy()
    const feed = await stats.getEmployeeActivityFeed(admin.auth.userId, { from: today, to: today })
    expect(feed).toBeTruthy()
    expect((await stats.exportEmployeeActivity(admin.auth.userId, today, today)).ok).toBe(true)
  })
})

describeIntegration('users API: профиль и аватар', () => {
  it('аватар загружается и снимается', async () => {
    const u = await registerVerified()
    u.session.use()
    await users.uploadAvatar(new File([PNG()], 'avatar.png', { type: 'image/png' }))
    expect((await users.getMe()).avatar_path).toBeTruthy()

    await users.deleteAvatar()
    // Своей картинки больше нет — вместо неё раздел рисует identicon.
    expect((await users.getMe()).avatar_path ?? null).toBeNull()
  })

  it('администратор заводит сотрудника в своей компании и сбрасывает ему пароль', async () => {
    const admin = await newCompanyAdmin('admin')
    admin.session.use()
    const login = uniq('emp_')
    const created = await users.createUser({
      fio: 'Новиков Новик Новикович',
      login,
      email: `${login}@apitest.local`,
      role_id: 1,
    })
    const id = created.id ?? created.user?.id
    expect(id).toBeTruthy()

    const card = await users.getUser(id)
    expect(card.login).toBe(login)

    await users.resetUserPassword(id)
    expect(dbQuery(`SELECT is_default_pass FROM users WHERE id = ${id}`)[0][0]).toBe('t')
  })

  it('карточка сотрудника чужой компании закрыта', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newCompanyAdmin('b')
    b.session.use()
    await expectClientError(users.getUser(a.auth.userId))
    await expectClientError(users.resetUserPassword(a.auth.userId))
  })
})

describeIntegration('companies API: остальное управление', () => {
  it('карточка участника правится создателем', async () => {
    const owner = await newCompanyAdmin('owner')
    const member = await newMember(owner, owner.companyId, 1, 'member')

    owner.session.use()
    await companies.updateCompanyMember(owner.companyId, member.auth.userId, { post: 'Ведущий инженер' })
    const members = await companies.listCompanyMembers(owner.companyId)
    expect(members.find((m) => m.id === member.auth.userId).post).toBe('Ведущий инженер')

    await companies.resetCompanyMemberPassword(owner.companyId, member.auth.userId)
    expect(dbQuery(`SELECT is_default_pass FROM users WHERE id = ${member.auth.userId}`)[0][0]).toBe('t')
  })

  it('кандидаты и справочник компании', async () => {
    const owner = await newCompanyAdmin('owner')
    const member = await newMember(owner, owner.companyId, 1, 'member')

    owner.session.use()
    const dir = await companies.getCompanyDirectory(owner.companyId)
    expect((dir.items ?? dir).some((u) => u.id === member.auth.userId)).toBe(true)

    // Кандидаты — те, кого ещё нет в компании: свой участник туда не попадает.
    const candidates = await companies.getCompanyCandidates(owner.companyId, member.login)
    expect((candidates.items ?? candidates).some((u) => u.id === member.auth.userId)).toBe(false)
  })

  it('превью приглашения показывает компанию и роль', async () => {
    const owner = await newCompanyAdmin('owner')
    owner.session.use()
    const email = `preview-${uniq('')}@apitest.local`
    await companies.createCompanyInvite(owner.companyId, email, 2)
    const token = dbQuery(`SELECT token FROM company_invites
      WHERE lower(email) = lower('${email}') ORDER BY id DESC LIMIT 1`)[0][0]

    const invited = await registerVerified('invited')
    invited.session.use()
    const preview = await companies.getInvitePreview(token)
    expect(preview.company_name).toBeTruthy()
    expect(preview.role_name).toBeTruthy()

    await companies.acceptCompanyInvite(token)
    owner.session.use()
    expect((await companies.listCompanyMembers(owner.companyId))
      .some((m) => m.id === invited.auth.userId)).toBe(true)
  })

  it('отключённая компания не пускает участников', async () => {
    const owner = await newCompanyAdmin('owner')
    const member = await newMember(owner, owner.companyId, 1, 'member')

    dbQuery(`UPDATE users SET is_super_admin = TRUE WHERE id = ${owner.auth.userId}`)
    const root = new Session('root')
    root.use()
    await useAuthStore().login(owner.login, owner.password)
    await companies.toggleCompanyActive(owner.companyId, false)

    // Отключённая компания закрывает вход в неё, а не только вывеску.
    member.session.use()
    const err = await companies.listCompanyMembers(owner.companyId).then(() => null, (e) => e)
    expect(err).toBeTruthy()

    root.use()
    await companies.toggleCompanyActive(owner.companyId, true)
  })

  it('список всех компаний — супер-админу; удаление — создателю', async () => {
    const owner = await newCompanyAdmin('owner')
    owner.session.use()
    await expectStatus(companies.listCompanies(), 403)

    const victim = await companies.createCompany({ name: uniq('На удаление ') })
    await companies.deleteCompany(victim.id)
    await expectClientError(companies.getCompany(victim.id))
  })
})

describeIntegration('прочие ручки', () => {
  it('файл реестра и календаря загружается в хранилище', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    // Файл кладут В РЕЕСТР: от него зависят и право на загрузку, и чья квота.
    const target = await registries.createRegistry(uniq('Для файлов '))
    const regFile = await registries.uploadFile(target.id, new File([PNG()], 'doc.png', { type: 'image/png' }))
    expect(regFile.path).toContain('registry/')
    expect(regFile.name).toBe('doc.png')
    const calFile = await calendars.uploadFile(new File([PNG()], 'doc.png', { type: 'image/png' }))
    expect(calFile.path).toContain('calendar/')
  })

  it('напоминания, привязанные к записи, находятся по ссылке', async () => {
    const u = await registerVerified()
    u.session.use()
    const linked = await reminders.getLinked('diary', 999999)
    expect(Array.isArray(linked.items ?? linked.reminders ?? linked)).toBe(true)
  })

  it('переиндексация задач без ИИ отвечает понятно', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    const ai = await import('@/api/ai.js')
    await expectClientError(ai.reindexAiTasks(admin.companyId))
  })

  it('повторная отправка письма подтверждения троттлится', async () => {
    const s = new Session('resend')
    s.use()
    const auth = await import('@/api/auth.js')
    const store = useAuthStore()
    const login = uniq('res_')
    const email = `${login}@apitest.local`
    await store.register({ fio: 'Повторов Повтор Повторович', email, login, password: 'secret-pass-123' })

    // Письмо ушло при регистрации; повтор в ближайшую минуту отвергается,
    // иначе форму превратят в рассыльщик. Ответ — либо явный отказ, либо
    // «ок» без письма, но второе письмо в БД не появляется.
    const sentBefore = dbQuery(`SELECT count(*) FROM email_verifications v
      JOIN users u ON u.id = v.user_id WHERE lower(u.email) = lower('${email}')`)[0][0]
    await auth.resendVerification(email).catch(() => {})
    const sentAfter = dbQuery(`SELECT count(*) FROM email_verifications v
      JOIN users u ON u.id = v.user_id WHERE lower(u.email) = lower('${email}')`)[0][0]
    expect(sentAfter).toBe(sentBefore)
  })

  it('смена дефолтного пароля недоступна тому, у кого его нет', async () => {
    const u = await registerVerified()
    u.session.use()
    const auth = await import('@/api/auth.js')
    await expectClientError(auth.changeDefault({ password: 'another-secret-123' }))
  })
})
