/* Мессенджер (msgsvc): диалоги, сообщения, реакции, прочтения, группы, папки.

   Здесь особенно много правил, которые UI не проверит: watermark прочтений в
   группах, лимит реакций на человека, «удалить у себя» против «у всех»,
   передача владения группой и то, что задача уходит только в свою компанию. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin, newMember } from '../setup/factory.js'
import * as api from '@/api/messenger.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

const listOf = (res) => res.conversations ?? res.items ?? res
const messagesOf = (res) => res.messages ?? res.items ?? res

describeIntegration('messenger API: диалоги и сообщения', () => {
  it('диалог открывается один раз на пару, а не плодится', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, 1, 'b')

    a.session.use()
    const first = await api.openConversation(b.auth.userId)
    const second = await api.openConversation(b.auth.userId)
    expect(second.id).toBe(first.id)
  })

  it('сообщение доходит собеседнику и видно обоим', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, 1, 'b')

    a.session.use()
    const conv = await api.openConversation(b.auth.userId)
    const sent = await api.sendMessage(conv.id, { text: 'Договорились на десять' })
    expect(sent.id).toBeGreaterThan(0)

    b.session.use()
    const seen = messagesOf(await api.listMessages(conv.id))
    expect(seen.some((m) => m.id === sent.id)).toBe(true)
  })

  it('правка своего сообщения проходит, чужого — нет', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, 1, 'b')

    a.session.use()
    const conv = await api.openConversation(b.auth.userId)
    const msg = await api.sendMessage(conv.id, { text: 'Первый вариант' })

    await api.updateMessage(msg.id, 'Исправленный вариант')
    const mine = messagesOf(await api.listMessages(conv.id)).find((m) => m.id === msg.id)
    expect(mine.text).toBe('Исправленный вариант')

    b.session.use()
    await expect(api.updateMessage(msg.id, 'Чужая правка')).rejects.toBeTruthy()
  })

  it('«удалить у себя» прячет только у меня, «у всех» — у обоих', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, 1, 'b')

    a.session.use()
    const conv = await api.openConversation(b.auth.userId)
    const mine = await api.sendMessage(conv.id, { text: 'Скрою у себя' })
    const forAll = await api.sendMessage(conv.id, { text: 'Уберу у всех' })

    await api.deleteMessage(mine.id, 'me')
    await api.deleteMessage(forAll.id, 'all')

    const authorSees = messagesOf(await api.listMessages(conv.id)).map((m) => m.id)
    expect(authorSees).not.toContain(mine.id)
    expect(authorSees).not.toContain(forAll.id)

    b.session.use()
    const mateSees = messagesOf(await api.listMessages(conv.id)).map((m) => m.id)
    // «У себя» собеседника не касается, «у всех» — исчезло и у него.
    expect(mateSees).toContain(mine.id)
    expect(mateSees).not.toContain(forAll.id)
  })

  it('непрочитанные считаются у получателя и гаснут после markRead', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, 1, 'b')

    a.session.use()
    const conv = await api.openConversation(b.auth.userId)
    await api.sendMessage(conv.id, { text: 'Непрочитанное' })

    b.session.use()
    const before = await api.getUnreadCount()
    expect(before.total).toBeGreaterThan(0)

    await api.markRead(conv.id)
    const after = await api.getUnreadCount()
    expect(after.total).toBe(0)
  })

  it('реакций от одного человека не больше двух разных', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, 1, 'b')

    a.session.use()
    const conv = await api.openConversation(b.auth.userId)
    const msg = await api.sendMessage(conv.id, { text: 'Реакции' })

    await api.toggleReaction(msg.id, '👍')
    await api.toggleReaction(msg.id, '🔥')
    // Третья — отказ: правило «не больше двух эмодзи от человека».
    await expectStatus(api.toggleReaction(msg.id, '🎉'), 422)

    // Снятие освобождает место под новую.
    await api.toggleReaction(msg.id, '👍')
    await api.toggleReaction(msg.id, '🎉')
  })

  it('закрепление сообщения общее для чата', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, 1, 'b')

    a.session.use()
    const conv = await api.openConversation(b.auth.userId)
    const msg = await api.sendMessage(conv.id, { text: 'Важное' })
    await api.togglePinMessage(msg.id)

    b.session.use()
    const pinned = await api.listPinnedMessages(conv.id)
    expect((pinned.messages ?? pinned).some((m) => m.id === msg.id)).toBe(true)
  })

  it('закрепление ЧАТА — личное дело каждого', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, 1, 'b')

    a.session.use()
    const conv = await api.openConversation(b.auth.userId)
    await api.togglePin(conv.id)
    const mine = listOf(await api.listConversations()).find((c) => c.id === conv.id)
    expect(mine.pinned_at ?? mine.pinned).toBeTruthy()

    b.session.use()
    const mate = listOf(await api.listConversations()).find((c) => c.id === conv.id)
    expect(mate?.pinned_at ?? mate?.pinned ?? null).toBeFalsy()
  })

  it('в чужой диалог не написать и не прочитать', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, 1, 'b')
    a.session.use()
    const conv = await api.openConversation(b.auth.userId)

    const stranger = await registerVerified('stranger')
    stranger.session.use()
    await expect(api.listMessages(conv.id)).rejects.toBeTruthy()
    await expect(api.sendMessage(conv.id, { text: 'Влезаю' })).rejects.toBeTruthy()
  })

  it('пустое сообщение без вложений не отправляется', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, 1, 'b')
    a.session.use()
    const conv = await api.openConversation(b.auth.userId)

    await expect(api.sendMessage(conv.id, { text: '' })).rejects.toBeTruthy()
    await expect(api.sendMessage(conv.id, { text: '   ' })).rejects.toBeTruthy()
  })
})

describeIntegration('messenger API: группы', () => {
  async function groupOfThree() {
    const owner = await newCompanyAdmin('owner')
    const m1 = await newMember(owner, owner.companyId, 1, 'm1')
    const m2 = await newMember(owner, owner.companyId, 1, 'm2')
    owner.session.use()
    const group = await api.createGroup({
      title: uniq('Проект '), memberIds: [m1.auth.userId, m2.auth.userId],
    })
    return { owner, m1, m2, group }
  }

  it('создание группы: владелец и участники на месте', async () => {
    const { owner, m1, group } = await groupOfThree()
    owner.session.use()
    const info = await api.getGroup(group.id)
    const members = info.members ?? []
    expect(members.length).toBe(3)
    // Создатель группы — её владелец, остальные приглашённые — участники.
    expect(members.find((m) => m.user?.id === owner.auth.userId)?.role).toBe('owner')
    expect(members.some((m) => m.user?.id === m1.auth.userId)).toBe(true)
  })

  it('состав и роли меняет тот, кому это позволено', async () => {
    const { owner, m1, m2, group } = await groupOfThree()

    // Рядовой участник состав не трогает.
    m1.session.use()
    await expect(api.removeGroupMember(group.id, m2.auth.userId)).rejects.toBeTruthy()

    // Владелец назначает администратора, тот уже вправе убирать людей.
    owner.session.use()
    await api.setMemberRole(group.id, m1.auth.userId, 'admin')
    m1.session.use()
    await api.removeGroupMember(group.id, m2.auth.userId)

    owner.session.use()
    const info = await api.getGroup(group.id)
    expect((info.members ?? []).some((m) => m.user?.id === m2.auth.userId)).toBe(false)
  })

  it('передача владения меняет владельца, прежний остаётся участником', async () => {
    const { owner, m1, group } = await groupOfThree()
    owner.session.use()
    await api.transferOwnership(group.id, m1.auth.userId)

    const info = await api.getGroup(group.id)
    const roles = Object.fromEntries((info.members ?? []).map((m) => [m.user?.id, m.role]))
    expect(roles[m1.auth.userId]).toBe('owner')
    expect(roles[owner.auth.userId]).not.toBe('owner')
  })

  it('переименование — только тому, кто вправе править карточку', async () => {
    const { owner, m1, group } = await groupOfThree()
    m1.session.use()
    await expect(api.renameGroup(group.id, 'Самовольно')).rejects.toBeTruthy()

    owner.session.use()
    await api.renameGroup(group.id, 'Проект Альфа')
    expect((await api.getGroup(group.id)).title).toBe('Проект Альфа')
  })

  it('выход из группы: участник уходит и перестаёт её видеть', async () => {
    const { m1, group } = await groupOfThree()
    m1.session.use()
    await api.leaveGroup(group.id)
    const mine = listOf(await api.listConversations())
    expect(mine.some((c) => c.id === group.id)).toBe(false)
  })

  it('ссылка-приглашение вводит нового участника, отзыв — закрывает', async () => {
    const { owner, group } = await groupOfThree()
    owner.session.use()
    const link = await api.groupInviteLink(group.id)
    expect(link.code).toBeTruthy()

    const guest = await newMember(owner, owner.companyId, 1, 'guest')
    guest.session.use()
    const preview = await api.groupInvitePreview(link.code)
    expect(preview.title ?? preview.group?.title).toBeTruthy()
    await api.joinGroup(link.code)

    owner.session.use()
    expect((await api.getGroup(group.id)).members.some((m) => m.user?.id === guest.auth.userId)).toBe(true)

    await api.revokeGroupInviteLink(group.id)
    const late = await newMember(owner, owner.companyId, 1, 'late')
    late.session.use()
    await expect(api.joinGroup(link.code)).rejects.toBeTruthy()
  })

  it('mute не мешает читать, но помечает чат приглушённым', async () => {
    const { m1, group } = await groupOfThree()
    m1.session.use()
    await api.muteGroup(group.id, true)
    const mine = listOf(await api.listConversations()).find((c) => c.id === group.id)
    expect(mine.muted).toBe(true)

    await api.muteGroup(group.id, false)
    const after = listOf(await api.listConversations()).find((c) => c.id === group.id)
    expect(after.muted).toBe(false)
  })
})

describeIntegration('messenger API: папки чатов и фон', () => {
  it('папка: создание, наполнение, порядок, удаление', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, 1, 'b')
    a.session.use()
    const conv = await api.openConversation(b.auth.userId)

    const folder = await api.createFolder({ title: uniq('Работа ') })
    expect(folder.id).toBeGreaterThan(0)

    await api.addFolderItem(folder.id, conv.id)
    let folders = await api.listFolders()
    let mine = (folders.folders ?? folders).find((f) => f.id === folder.id)
    expect((mine.conversation_ids ?? mine.items ?? []).length).toBe(1)

    await api.removeFolderItem(folder.id, conv.id)
    folders = await api.listFolders()
    mine = (folders.folders ?? folders).find((f) => f.id === folder.id)
    expect((mine.conversation_ids ?? mine.items ?? []).length).toBe(0)

    await api.updateFolder(folder.id, { title: 'Переименована' })
    await api.deleteFolder(folder.id)
    folders = await api.listFolders()
    expect((folders.folders ?? folders).some((f) => f.id === folder.id)).toBe(false)
  })

  it('чужая папка: правка и удаление отвечают отказом, а не «успешно»', async () => {
    const a = await registerVerified('a')
    a.session.use()
    const folder = await api.createFolder({ title: uniq('Личная ') })

    const b = await registerVerified('b')
    b.session.use()
    // Данные и так защищены (owner_id в WHERE), но клиент обязан узнать, что
    // операция НЕ прошла: молчаливый успех выглядел бы как переименование.
    await expectStatus(api.updateFolder(folder.id, { title: 'Взлом' }), 404)
    await expectStatus(api.deleteFolder(folder.id), 404)

    a.session.use()
    const mine = (await api.listFolders()).folders ?? await api.listFolders()
    expect((mine.folders ?? mine).find((f) => f.id === folder.id)?.title)
      .not.toBe('Взлом')
  })

  it('фон переписки: общий и по конкретному чату', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newMember(a, a.companyId, 1, 'b')
    a.session.use()
    const conv = await api.openConversation(b.auth.userId)

    await api.setChatBackground(null, { preset: 'waves' })
    await api.setChatBackground(conv.id, { preset: 'dots' })

    const all = await api.getChatBackgrounds()
    const map = all.backgrounds ?? all
    expect(JSON.stringify(map)).toContain('dots')

    await api.deleteChatBackground(conv.id)
    await api.deleteChatBackground()
  })
})
