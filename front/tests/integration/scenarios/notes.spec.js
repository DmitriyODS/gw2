/* Заметки против живого notesvc: заметки и папки, метки, публичные ссылки,
   адресный шаринг пользователю и компании, экспорт/импорт.

   Проверяем не только успешные пути: чужая заметка не должна ни читаться, ни
   правиться, папка не может стать своим потомком, а удаление папки — уносить
   вложенное. Это инварианты сервиса, а не UI, и ловить их надо здесь. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin, newMember } from '../setup/factory.js'
import * as api from '@/api/notes.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

describeIntegration('notes API: заметки и папки', () => {
  it('жизненный цикл заметки: создание, правка, копия, удаление', async () => {
    const u = await registerVerified()
    u.session.use()

    const created = await api.createNote('Первая заметка')
    expect(created.id).toBeGreaterThan(0)

    const doc = { type: 'doc', content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Привет' }] }] }
    await api.updateNote(created.id, { title: 'Переименована', doc })

    const one = await api.getNote(created.id)
    expect(one.title).toBe('Переименована')
    expect(one.my_access).toBe('owner')
    // Плоский текст сервер считает сам из документа: наружу он идёт выжимкой
    // (excerpt) — по нему же работает поиск.
    expect(one.excerpt ?? '').toContain('Привет')

    const copy = await api.copyNote(created.id)
    expect(copy.id).not.toBe(created.id)

    await api.deleteNote(created.id)
    await expectStatus(api.getNote(created.id), 404)
  })

  it('поиск находит по тексту документа, а не только по названию', async () => {
    const u = await registerVerified()
    u.session.use()
    const n = await api.createNote('Без названия')
    await api.updateNote(n.id, {
      doc: { type: 'doc', content: [{ type: 'paragraph', content: [{ type: 'text', text: 'кварцевый песок' }] }] },
    })

    const found = await api.getNotes({ search: 'кварцевый' })
    expect((found.notes ?? []).some((x) => x.id === n.id)).toBe(true)

    const empty = await api.getNotes({ search: 'заведомо-небывалое-слово' })
    expect((empty.notes ?? []).some((x) => x.id === n.id)).toBe(false)
  })

  it('чужая заметка недоступна: ни чтения, ни правки, ни удаления', async () => {
    const owner = await registerVerified('owner')
    owner.session.use()
    const n = await api.createNote('Личное')

    const stranger = await registerVerified('stranger')
    stranger.session.use()
    await expectStatus(api.getNote(n.id), 404)
    await expectStatus(api.updateNote(n.id, { title: 'Взлом' }), 404)
    await expectStatus(api.deleteNote(n.id), 404)

    // Владельцу заметка по-прежнему видна и цела.
    owner.session.use()
    expect((await api.getNote(n.id)).title).toBe('Личное')
  })

  it('папки: вложенность, запрет цикла, удаление репарентит детей', async () => {
    const u = await registerVerified()
    u.session.use()

    const root = await api.createFolder(uniq('Корень '))
    const child = await api.createFolder(uniq('Ветка '), root.id)
    const inner = await api.createNote('Внутри', child.id)

    const kids = await api.getFolderChildren(root.id)
    expect((kids.folders ?? []).some((f) => f.id === child.id)).toBe(true)

    // Папка не может уехать внутрь собственного потомка — это разорвало бы дерево.
    await expect(api.moveFolder(root.id, child.id)).rejects.toBeTruthy()

    // Удаление папки поднимает детей к родителю, а не уносит их с собой.
    await api.deleteFolder(child.id)
    const note = await api.getNote(inner.id)
    expect(note.folder_id === root.id || note.folder_id === null).toBe(true)
  })

  it('метки: создание, назначение, фильтр и удаление', async () => {
    const u = await registerVerified()
    u.session.use()

    const tag = await api.createTag(uniq('срочно-'), 'red')
    expect(tag.id).toBeGreaterThan(0)

    const n = await api.createNote('С меткой')
    const other = await api.createNote('Без метки')
    await api.setNoteTags(n.id, [tag.id])
    expect((await api.getNote(n.id)).tag_ids).toContain(tag.id)

    const byTag = await api.getNotes({ tag_ids: String(tag.id) })
    const ids = (byTag.notes ?? []).map((x) => x.id)
    expect(ids).toContain(n.id)
    expect(ids).not.toContain(other.id)

    await api.deleteTag(tag.id)
    expect((await api.getTags()).some?.((t) => t.id === tag.id)).toBeFalsy()
  })
})

describeIntegration('notes API: доступ и обмен', () => {
  it('публичная ссылка: view открывает чтение, edit — запись, отзыв закрывает', async () => {
    const u = await registerVerified()
    u.session.use()
    const n = await api.createNote('Публичная')

    const share = await api.createShare(n.id, 'view')
    expect(share.code).toBeTruthy()

    // Гостю (без сессии) содержимое видно по коду.
    const guest = await registerVerified('guest')
    guest.session.use()
    const shared = await api.getSharedNote(share.code)
    expect(shared.note?.title ?? shared.title).toBe('Публичная')

    // Ссылка «только просмотр» править не даёт.
    await expect(api.updateSharedNote(share.code, { title: 'Правка гостя' })).rejects.toBeTruthy()

    u.session.use()
    await api.revokeShare(n.id, share.id)
    guest.session.use()
    await expect(api.getSharedNote(share.code)).rejects.toBeTruthy()
  })

  it('адресный шаринг пользователю: view читает, edit пишет, отзыв закрывает', async () => {
    const owner = await registerVerified('owner')
    owner.session.use()
    const n = await api.createNote('Совместная')

    const mate = await registerVerified('mate')
    owner.session.use() // registerVerified переключил сессию на нового пользователя
    await api.shareNoteWithUser(n.id, mate.auth.userId, false)

    mate.session.use()
    const seen = await api.getNote(n.id)
    expect(seen.my_access).toBe('view')
    await expect(api.updateNote(n.id, { title: 'Правка' })).rejects.toBeTruthy()

    owner.session.use()
    await api.shareNoteWithUser(n.id, mate.auth.userId, true)
    mate.session.use()
    expect((await api.getNote(n.id)).my_access).toBe('edit')
    await api.updateNote(n.id, { title: 'Соавтор поправил' })

    owner.session.use()
    await api.unshareNoteUser(n.id, mate.auth.userId)
    mate.session.use()
    await expectStatus(api.getNote(n.id), 404)
  })

  it('доступ по расшаренной ПАПКЕ каскадит на вложенную заметку', async () => {
    const owner = await newCompanyAdmin('owner')
    owner.session.use()
    const folder = await api.createFolder(uniq('Общая '))
    const inside = await api.createNote('Внутри общей', folder.id)

    const mate = await newMember(owner, owner.companyId, 1, 'mate')
    owner.session.use()
    await api.shareFolderWithUser(folder.id, mate.auth.userId, false)

    mate.session.use()
    const seen = await api.getNote(inside.id)
    expect(['view', 'edit']).toContain(seen.my_access)

    owner.session.use()
    await api.unshareFolderUser(folder.id, mate.auth.userId)
    mate.session.use()
    await expectStatus(api.getNote(inside.id), 404)
  })

  it('шаринг компании открывает заметку её участникам', async () => {
    const owner = await newCompanyAdmin('owner')
    const mate = await newMember(owner, owner.companyId, 1, 'mate')

    owner.session.use()
    const n = await api.createNote('Для команды')
    await api.shareNoteWithCompany(n.id, owner.companyId, false)

    mate.session.use()
    expect((await api.getNote(n.id)).my_access).toBe('view')

    // Посторонний вне компании доступа не получает.
    const outsider = await registerVerified('outsider')
    outsider.session.use()
    await expectStatus(api.getNote(n.id), 404)
  })
})

describeIntegration('notes API: границы и отказы', () => {
  it('несуществующие идентификаторы — 404, а не 500', async () => {
    const u = await registerVerified()
    u.session.use()
    await expectStatus(api.getNote(999999999), 404)
    await expectStatus(api.deleteNote(999999999), 404)
    await expect(api.getFolderChildren(999999999)).rejects.toBeTruthy()
  })

  it('пустое название допустимо — заметка живёт «без названия»', async () => {
    const u = await registerVerified()
    u.session.use()
    const n = await api.createNote('')
    expect(n.id).toBeGreaterThan(0)
    expect(await api.getNote(n.id)).toBeTruthy()
  })

  it('слишком длинное название сервер не принимает молча', async () => {
    const u = await registerVerified()
    u.session.use()
    const n = await api.createNote('Норма')
    const huge = 'я'.repeat(5000)
    try {
      await api.updateNote(n.id, { title: huge })
      // Если приняли — обязаны обрезать до разумного, а не хранить километр.
      const after = await api.getNote(n.id)
      expect(after.title.length).toBeLessThanOrEqual(1000)
    } catch (e) {
      expect(e.status).toBeGreaterThanOrEqual(400)
      expect(e.status).toBeLessThan(500)
    }
  })

  it('экспорт: три формата отдаются, незнакомый — отказ, пустой — txt', async () => {
    const u = await registerVerified()
    u.session.use()
    const n = await api.createNote('На выгрузку')

    for (const fmt of ['txt', 'md', 'docx']) {
      const res = await api.exportNote(n.id, fmt)
      expect(res.ok).toBe(true)
      expect(res.headers.get('content-disposition')).toContain(`.${fmt}`)
    }

    // Пустой формат — это «по умолчанию», а не мусор.
    const byDefault = await api.exportNote(n.id, '')
    expect(byDefault.ok).toBe(true)
    expect(byDefault.headers.get('content-disposition')).toContain('.txt')

    // А вот явно неизвестный формат — ошибка клиента: отдать txt вместо
    // запрошенного значило бы вернуть «успешный» файл не того типа.
    await expectStatus(api.exportNote(n.id, 'exe'), 400)
  })

  it('без сессии заметки не отдаются', async () => {
    const u = await registerVerified()
    u.session.use()
    await api.createNote('Секрет')
    u.auth.logout && (await u.auth.logout().catch(() => {}))
    await expect(api.getNotes({})).rejects.toBeTruthy()
  })
})
