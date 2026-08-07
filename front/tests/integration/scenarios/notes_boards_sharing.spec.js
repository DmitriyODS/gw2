/* Заметки и доски: папки, доступы, выгрузки и совместная работа.

   Оба раздела устроены одинаково (личная сущность + иерархия папок + шаринг),
   поэтому и правила здесь общие, и цена ошибки одна: доступ, выданный на папку,
   ОБЯЗАН каскадировать на её содержимое, а снятый — обязан пропадать. Второе
   важное правило — «эффективный доступ считает сервер»: клиент не должен
   получать право на правку из того, что он что-то видит.

   Файловые ручки (импорт, картинки, превью) проверяются настоящей загрузкой —
   стенд собирает multipart сам. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq, Session } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin, newMember } from '../setup/factory.js'
import * as notes from '@/api/notes.js'
import * as boards from '@/api/boards.js'

async function expectClientError(promise) {
  const err = await promise.then(() => null, (e) => e)
  expect(err).toBeTruthy()
  expect(err.status).toBeGreaterThanOrEqual(400)
  expect(err.status).toBeLessThan(500)
  return err
}

const docOf = (text) => ({
  type: 'doc',
  content: [{ type: 'paragraph', content: [{ type: 'text', text }] }],
})

const sceneOf = (text) => ({
  version: 2,
  layers: [{ id: 'l1', name: 'Слой 1', visible: true, locked: false }],
  objects: [{ id: 'o1', layer: 'l1', kind: 'text', x: 10, y: 10, text }],
})

describeIntegration('notes API: папки и перемещение', () => {
  it('дерево папок строится, ребёнок читается, цикл не допускается', async () => {
    const u = await registerVerified()
    u.session.use()
    const root = await notes.createFolder(uniq('Проекты '))
    const child = await notes.createFolder(uniq('Договоры '), root.id)

    const tree = await notes.getFolders()
    const items = tree.folders ?? tree.items ?? tree
    expect(items.some((f) => f.id === root.id)).toBe(true)

    const children = await notes.getFolderChildren(root.id)
    expect((children.folders ?? children.items ?? []).some((f) => f.id === child.id)).toBe(true)

    // Папку нельзя убрать внутрь собственного потомка — дерево перестало бы
    // быть деревом, а поддерево пропало бы из навигации.
    await expectClientError(notes.moveFolder(root.id, child.id))
  })

  it('удаление папки поднимает содержимое к родителю, а не стирает его', async () => {
    const u = await registerVerified()
    u.session.use()
    const parent = await notes.createFolder(uniq('Родитель '))
    const middle = await notes.createFolder(uniq('Середина '), parent.id)
    const note = await notes.createNote('Заметка внутри', middle.id)

    await notes.deleteFolder(middle.id)
    const survived = await notes.getNote(note.id)
    expect(survived.folder_id).toBe(parent.id)
  })

  it('копия папки уносит всё поддерево', async () => {
    const u = await registerVerified()
    u.session.use()
    const src = await notes.createFolder(uniq('Исходная '))
    await notes.createNote('Первая', src.id)
    await notes.createNote('Вторая', src.id)

    const copy = await notes.copyFolder(src.id)
    const copyId = copy.id ?? copy.folder?.id
    const inside = await notes.getNotes({ folder_id: copyId })
    expect((inside.items ?? inside.notes ?? []).length).toBe(2)
  })

  it('заметка переезжает между папками и в корень', async () => {
    const u = await registerVerified()
    u.session.use()
    const folder = await notes.createFolder(uniq('Папка '))
    const note = await notes.createNote('Переезжающая')

    await notes.moveNote(note.id, folder.id)
    expect((await notes.getNote(note.id)).folder_id).toBe(folder.id)

    await notes.moveNote(note.id, null)
    expect((await notes.getNote(note.id)).folder_id ?? null).toBeNull()
  })

  it('метка правится и снимается', async () => {
    const u = await registerVerified()
    u.session.use()
    const tag = await notes.createTag(uniq('важное-'), 'red')
    await notes.updateTag(tag.id, { name: 'очень важное', color: 'blue' })
    const list = await notes.getTags()
    const updated = (list.tags ?? list.items ?? list).find((t) => t.id === tag.id)
    expect(updated.name).toBe('очень важное')
    await notes.deleteTag(tag.id)
  })
})

describeIntegration('notes API: доступы', () => {
  it('доступ по папке каскадирует на её заметки', async () => {
    const owner = await registerVerified('owner')
    const mate = await registerVerified('mate')

    owner.session.use()
    const folder = await notes.createFolder(uniq('Общая '))
    const note = await notes.createNote('Внутри общей папки', folder.id)
    await notes.updateNote(note.id, { doc: docOf('Содержимое') })

    // До выдачи доступа заметка не видна вовсе.
    mate.session.use()
    await expectClientError(notes.getNote(note.id))

    owner.session.use()
    await notes.shareFolderWithUser(folder.id, mate.auth.userId, false)

    // Доступ на папку распространяется на её содержимое — иначе шаринг папки
    // не имел бы смысла.
    mate.session.use()
    const seen = await notes.getNote(note.id)
    expect(seen.id).toBe(note.id)
    expect(seen.my_access).toBe('view')
    // Право на чтение не даёт править.
    await expectClientError(notes.updateNote(note.id, { doc: docOf('Правка чужой рукой') }))

    owner.session.use()
    await notes.unshareFolderUser(folder.id, mate.auth.userId)
    mate.session.use()
    await expectClientError(notes.getNote(note.id))
  })

  it('доступ «на правку» позволяет менять текст', async () => {
    const owner = await registerVerified('owner')
    const mate = await registerVerified('mate')

    owner.session.use()
    const note = await notes.createNote('Совместная')
    await notes.shareNoteWithUser(note.id, mate.auth.userId, true)

    mate.session.use()
    expect((await notes.getNote(note.id)).my_access).toBe('edit')
    await notes.updateNote(note.id, { doc: docOf('Дополнено соавтором') })
    expect((await notes.getNote(note.id)).excerpt ?? '').toContain('соавтором')

    // Список тех, кому открыт доступ, ведёт владелец.
    owner.session.use()
    const members = await notes.getNoteMembers(note.id)
    expect(members.members.some((m) => (m.user_id ?? m.id) === mate.auth.userId)).toBe(true)
  })

  it('доступ компании открывает заметку её участникам', async () => {
    const admin = await newCompanyAdmin('admin')
    const worker = await newMember(admin, admin.companyId, 1, 'worker')
    const outsider = await registerVerified('outsider')

    admin.session.use()
    const note = await notes.createNote('Для всей компании')
    const companies = await notes.getMyCompanies()
    expect((companies.companies ?? companies.items ?? []).some((c) => c.id === admin.companyId)).toBe(true)
    await notes.shareNoteWithCompany(note.id, admin.companyId, false)

    worker.session.use()
    expect((await notes.getNote(note.id)).id).toBe(note.id)

    // Посторонний в компании не состоит — ему заметка закрыта.
    outsider.session.use()
    await expectClientError(notes.getNote(note.id))

    admin.session.use()
    await notes.unshareNoteCompany(note.id, admin.companyId)
    worker.session.use()
    await expectClientError(notes.getNote(note.id))
  })

  it('доступ компании на папку открывает поддерево', async () => {
    const admin = await newCompanyAdmin('admin')
    const worker = await newMember(admin, admin.companyId, 1, 'worker')

    admin.session.use()
    const folder = await notes.createFolder(uniq('Компанийная '))
    const note = await notes.createNote('В компанийной папке', folder.id)
    await notes.shareFolderWithCompany(folder.id, admin.companyId, true)

    const members = await notes.getFolderMembers(folder.id)
    expect(members).toBeTruthy()

    worker.session.use()
    expect((await notes.getNote(note.id)).my_access).toBe('edit')

    admin.session.use()
    await notes.unshareFolderCompany(folder.id, admin.companyId)
    worker.session.use()
    await expectClientError(notes.getNote(note.id))
  })

  it('публичная ссылка на правку принимает изменения, на чтение — нет', async () => {
    const owner = await registerVerified('owner')
    owner.session.use()
    const note = await notes.createNote('Публичная')
    const view = await notes.createShare(note.id, 'view')
    const edit = await notes.createShare(note.id, 'edit')

    const guest = new Session('guest')
    guest.use()
    expect((await notes.getSharedNote(view.code)).note?.id ?? (await notes.getSharedNote(view.code)).id).toBe(note.id)
    await expectClientError(notes.updateSharedNote(view.code, { doc: docOf('Гость правит') }))
    await notes.updateSharedNote(edit.code, { doc: docOf('Гость с правом правки') })

    owner.session.use()
    expect((await notes.getNote(note.id)).excerpt).toContain('правом правки')
    const shares = await notes.getShares(note.id)
    await notes.revokeShare(note.id, (shares.shares ?? shares.items ?? shares)[0].id)
  })

  it('совместное редактирование рассылается аудитории', async () => {
    const owner = await registerVerified('owner')
    const mate = await registerVerified('mate')
    owner.session.use()
    const note = await notes.createNote('Вместе')
    await notes.shareNoteWithUser(note.id, mate.auth.userId, true)

    mate.session.use()
    await notes.sendCollab(note.id, { kind: 'join' })
    await notes.sendCollab(note.id, { kind: 'cursor', anchor: 1, head: 3 })
    await notes.sendCollab(note.id, { kind: 'leave' })

    // Посторонний в комнату не попадает.
    const stranger = await registerVerified('stranger')
    stranger.session.use()
    await expectClientError(notes.sendCollab(note.id, { kind: 'join' }))
  })
})

describeIntegration('notes API: файлы и выгрузка', () => {
  it('импорт .txt заводит заметку с текстом', async () => {
    const u = await registerVerified()
    u.session.use()
    // Первая строка файла становится названием, остальное — текстом.
    const file = new File(['Название из файла\nТекст из файла для импорта'], 'заметка.txt', { type: 'text/plain' })
    const imported = await notes.importNote(file)
    const id = imported.id ?? imported.note?.id
    expect(id).toBeTruthy()
    const got = await notes.getNote(id)
    expect(got.title).toBe('Название из файла')
    expect(got.excerpt).toContain('из файла для импорта')
  })

  it('картинка загружается и получает адрес в хранилище', async () => {
    const u = await registerVerified()
    u.session.use()
    const note = await notes.createNote('С картинкой')
    // 1×1 PNG.
    const png = Uint8Array.from(atob('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=='), (c) => c.charCodeAt(0))
    const res = await notes.uploadImage(note.id, new File([png], 'точка.png', { type: 'image/png' }))
    expect(String(res.url ?? res.path ?? '')).toContain('/uploads/')
  })

  it('выгрузка заметки, папки и всего раздела', async () => {
    const u = await registerVerified()
    u.session.use()
    const folder = await notes.createFolder(uniq('Выгрузка '))
    const note = await notes.createNote('Выгружаемая', folder.id)
    await notes.updateNote(note.id, { doc: docOf('Текст для файла') })

    expect((await notes.exportNote(note.id, 'txt')).ok).toBe(true)
    expect((await notes.exportNote(note.id, 'docx')).ok).toBe(true)
    expect((await notes.exportFolder(folder.id, 'txt')).ok).toBe(true)
    expect((await notes.exportScope('all', 'txt')).ok).toBe(true)
  })

  it('неизвестный формат выгрузки отвергается', async () => {
    const u = await registerVerified()
    u.session.use()
    const note = await notes.createNote('Форматы')
    await expectClientError(notes.exportNote(note.id, 'pdf-которого-нет'))
  })
})

describeIntegration('boards API: папки, доступы, выгрузка', () => {
  it('доска переезжает по папкам, копия повторяет сцену', async () => {
    const u = await registerVerified()
    u.session.use()
    const folder = await boards.createFolder(uniq('Схемы '))
    const board = await boards.createBoard('Первая доска')
    await boards.updateBoard(board.id, { scene: sceneOf('Надпись на доске') })

    await boards.moveBoard(board.id, folder.id)
    expect((await boards.getBoard(board.id)).folder_id).toBe(folder.id)

    const copy = await boards.copyBoard(board.id)
    const copyId = copy.id ?? copy.board?.id
    expect((await boards.getBoard(copyId)).excerpt ?? '').toContain('Надпись')
  })

  it('дерево папок досок: дети, копия, защита от цикла', async () => {
    const u = await registerVerified()
    u.session.use()
    const root = await boards.createFolder(uniq('Корень '))
    const child = await boards.createFolder(uniq('Ветка '), root.id)
    await boards.createBoard('Внутри ветки', child.id)

    const children = await boards.getFolderChildren(root.id)
    expect((children.folders ?? children.items ?? []).some((f) => f.id === child.id)).toBe(true)

    const copy = await boards.copyFolder(child.id)
    expect(copy).toBeTruthy()
    await expectClientError(boards.moveFolder(root.id, child.id))

    const all = await boards.getFolders()
    expect((all.folders ?? all.items ?? all).length).toBeGreaterThanOrEqual(2)
  })

  it('доступ на доску и на папку каскадирует так же, как в заметках', async () => {
    const owner = await registerVerified('owner')
    const mate = await registerVerified('mate')

    owner.session.use()
    const folder = await boards.createFolder(uniq('Общие схемы '))
    const board = await boards.createBoard('Схема', folder.id)

    mate.session.use()
    await expectClientError(boards.getBoard(board.id))

    owner.session.use()
    await boards.shareFolderWithUser(folder.id, mate.auth.userId, true)
    expect(await boards.getFolderMembers(folder.id)).toBeTruthy()

    mate.session.use()
    expect((await boards.getBoard(board.id)).my_access).toBe('edit')

    owner.session.use()
    await boards.unshareFolderUser(folder.id, mate.auth.userId)
    mate.session.use()
    await expectClientError(boards.getBoard(board.id))
  })

  it('доступ компании к доске', async () => {
    const admin = await newCompanyAdmin('admin')
    const worker = await newMember(admin, admin.companyId, 1, 'worker')

    admin.session.use()
    const board = await boards.createBoard('Компанийная доска')
    const companies = await boards.getMyCompanies()
    expect((companies.companies ?? companies.items ?? []).length).toBeGreaterThan(0)
    await boards.shareBoardWithCompany(board.id, admin.companyId, false)

    worker.session.use()
    expect((await boards.getBoard(board.id)).id).toBe(board.id)

    admin.session.use()
    expect(await boards.getBoardMembers(board.id)).toBeTruthy()
    await boards.unshareBoardCompany(board.id, admin.companyId)
    worker.session.use()
    await expectClientError(boards.getBoard(board.id))
  })

  it('публичная ссылка: чтение и правка по праву доступа', async () => {
    const owner = await registerVerified('owner')
    owner.session.use()
    const board = await boards.createBoard('Публичная доска')
    const view = await boards.createShare(board.id, 'view')
    const edit = await boards.createShare(board.id, 'edit')

    const guest = new Session('guest')
    guest.use()
    expect(await boards.getSharedBoard(view.code)).toBeTruthy()
    await expectClientError(boards.updateSharedBoard(view.code, { scene: sceneOf('Гость рисует') }))
    await boards.updateSharedBoard(edit.code, { scene: sceneOf('Гость с правом') })

    owner.session.use()
    expect((await boards.getBoard(board.id)).excerpt ?? '').toContain('правом')
    const shares = await boards.getShares(board.id)
    await boards.revokeShare(board.id, (shares.shares ?? shares.items ?? shares)[0].id)
  })

  it('совместная работа: пообъектные правки принимаются соавтором', async () => {
    const owner = await registerVerified('owner')
    const mate = await registerVerified('mate')
    owner.session.use()
    const board = await boards.createBoard('Вместе рисуем')
    await boards.shareBoardWithUser(board.id, mate.auth.userId, true)

    mate.session.use()
    await boards.sendCollab(board.id, { kind: 'join' })
    await boards.sendCollab(board.id, { kind: 'ops', ops: [{ op: 'update', id: 'o1', patch: { x: 20 } }] })
    await boards.sendCollab(board.id, { kind: 'leave' })

    const stranger = await registerVerified('stranger')
    stranger.session.use()
    await expectClientError(boards.sendCollab(board.id, { kind: 'join' }))
  })

  it('выгрузка доски, папки и раздела; импорт возвращает сцену', async () => {
    const u = await registerVerified()
    u.session.use()
    const folder = await boards.createFolder(uniq('Экспорт '))
    const board = await boards.createBoard('Выгружаемая', folder.id)
    await boards.updateBoard(board.id, { scene: sceneOf('Экспортируемая надпись') })

    expect((await boards.exportBoard(board.id, 'svg')).ok).toBe(true)
    expect((await boards.exportBoard(board.id, 'json')).ok).toBe(true)
    expect((await boards.exportFolder(folder.id, 'json')).ok).toBe(true)
    expect((await boards.exportScope('all', 'json')).ok).toBe(true)
    await expectClientError(boards.exportBoard(board.id, 'нет-такого-формата'))

    // Файл .json — это сама сцена (то, что отдаёт выгрузка), название берётся
    // из имени файла.
    const json = JSON.stringify(sceneOf('Из файла'))
    const imported = await boards.importBoard(new File([json], 'Импортированная.json', { type: 'application/json' }))
    const id = imported.id ?? imported.board?.id
    const restored = await boards.getBoard(id)
    expect(restored.title).toBe('Импортированная')
    expect(restored.excerpt ?? '').toContain('Из файла')
  })

  it('картинка и миниатюра доски сохраняются', async () => {
    const u = await registerVerified()
    u.session.use()
    const board = await boards.createBoard('С картинкой')
    const png = Uint8Array.from(atob('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=='), (c) => c.charCodeAt(0))

    const img = await boards.uploadImage(board.id, new File([png], 'точка.png', { type: 'image/png' }))
    expect(String(img.url ?? img.path ?? '')).toContain('/uploads/')

    await boards.uploadPreview(board.id, new Blob([png], { type: 'image/png' }))
    expect((await boards.getBoard(board.id)).preview_url ?? '').toContain('/uploads/')
  })
})
