/* «Диск» против живого drivesvc: файлы и папки, корзина, шаринг.

   Проверяем не столько успешные пути, сколько инварианты сервиса: чужой файл
   не читается и не удаляется, папка не может стать своим потомком, корзина
   возвращает содержимое целиком, а доступ, выданный на папку, действует на
   лежащие в ней файлы. */
import { it, expect } from 'vitest'
import { describeIntegration, Session, uniq } from '../setup/harness.js'
import { registerVerified } from '../setup/factory.js'
import * as api from '@/api/drive.js'
import { apiRequest } from '@/api/client.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

// Файл для загрузки: harness сам собирает multipart из File.
function makeFile(name = 'note.txt', text = 'привет') {
  return new File([text], name, { type: 'text/plain' })
}

describeIntegration('drive API: файлы и папки', () => {
  it('жизненный цикл файла: загрузка, переименование, корзина, восстановление', async () => {
    const u = await registerVerified()
    u.session.use()

    const file = await api.uploadFile(makeFile(uniq('файл') + '.txt', 'содержимое'))
    expect(file.id).toBeGreaterThan(0)
    expect(file.size).toBeGreaterThan(0)
    // Наружу отдаётся адрес, а не ключ хранилища.
    expect(file.url).toMatch(/^\/uploads\//)

    await api.renameFile(file.id, 'переименован.txt')
    expect((await api.getFile(file.id)).name).toBe('переименован.txt')

    // В корзине файл жив: место занято, пока его можно вернуть.
    await api.trashFile(file.id)
    const listing = await api.browse({})
    expect(listing.files.some((f) => f.id === file.id)).toBe(false)
    const trash = await api.browse({ view: 'trash' })
    expect(trash.files.some((f) => f.id === file.id)).toBe(true)

    await api.restoreFile(file.id)
    expect((await api.browse({})).files.some((f) => f.id === file.id)).toBe(true)

    await api.purgeFile(file.id)
    await expectStatus(api.getFile(file.id), 404)
  })

  it('папки: вложенность, хлебные крошки и запрет цикла', async () => {
    const u = await registerVerified()
    u.session.use()

    const parent = await api.createFolder('Документы')
    const child = await api.createFolder('Договоры', parent.id)

    const inside = await api.browse({ folder_id: parent.id })
    expect(inside.folders.map((f) => f.id)).toContain(child.id)
    expect(inside.path.map((f) => f.id)).toEqual([parent.id])

    // Папку нельзя положить в собственного потомка — поддерево оторвалось бы.
    await expectStatus(api.moveFolder(parent.id, child.id), 409)
    // И в саму себя тоже.
    await expectStatus(api.moveFolder(parent.id, parent.id), 409)
  })

  it('корзина уносит папку вместе с содержимым и возвращает его обратно', async () => {
    const u = await registerVerified()
    u.session.use()

    const folder = await api.createFolder('Проект')
    const file = await api.uploadFile(makeFile('внутри.txt'), folder.id)

    await api.trashFolder(folder.id)
    const trash = await api.browse({ view: 'trash' })
    expect(trash.folders.some((f) => f.id === folder.id)).toBe(true)
    expect(trash.files.some((f) => f.id === file.id)).toBe(true)

    await api.restoreFolder(folder.id)
    const back = await api.browse({ folder_id: folder.id })
    expect(back.files.some((f) => f.id === file.id)).toBe(true)
  })

  it('избранное и поиск: выборки сквозные, папка их не ограничивает', async () => {
    const u = await registerVerified()
    u.session.use()

    const folder = await api.createFolder('Вложенная')
    const name = `${uniq('отчёт')}.txt`
    const file = await api.uploadFile(makeFile(name), folder.id)

    // Поиск идёт по всему диску, хотя файл лежит в папке.
    const found = await api.browse({ search: name.slice(0, 8) })
    expect(found.files.some((f) => f.id === file.id)).toBe(true)
    // Регистр не важен.
    const upper = await api.browse({ search: name.slice(0, 8).toUpperCase() })
    expect(upper.files.some((f) => f.id === file.id)).toBe(true)

    await api.starFile(file.id, true)
    const starred = await api.browse({ view: 'starred' })
    expect(starred.files.some((f) => f.id === file.id)).toBe(true)
  })

  it('чужой файл недоступен: ни прочитать, ни удалить', async () => {
    const owner = await registerVerified()
    owner.session.use()
    const file = await api.uploadFile(makeFile('личное.txt'))

    const stranger = await registerVerified()
    stranger.session.use()
    await expectStatus(api.getFile(file.id), 403)
    await expectStatus(api.trashFile(file.id), 403)
    await expectStatus(api.renameFile(file.id, 'взлом.txt'), 403)
  })
})

describeIntegration('drive API: доступ', () => {
  it('доступ к папке каскадит на её файлы', async () => {
    const owner = await registerVerified()
    owner.session.use()
    const folder = await api.createFolder('Общая')
    const file = await api.uploadFile(makeFile('внутри-общей.txt'), folder.id)

    // registerVerified переключает активную сессию на нового пользователя —
    // возвращаемся к владельцу, иначе доступ выдаёт сам адресат.
    const mate = await registerVerified()
    owner.session.use()
    await api.shareTo('folder', folder.id, { user_id: mate.auth.user.id, can_edit: false })

    // Адресат видит и папку, и лежащий в ней файл — отдельной выдачи не нужно.
    mate.session.use()
    const shared = await api.sharedWithMe()
    expect(shared.folders.some((f) => f.id === folder.id)).toBe(true)

    const one = await api.getFile(file.id)
    expect(one.my_access).toBe('view')
    // Просмотр — не правка: переименовать чужой файл нельзя.
    await expectStatus(api.renameFile(file.id, 'моё.txt'), 403)
  })

  it('публичная ссылка открывает файл без авторизации, отозванная — нет', async () => {
    const owner = await registerVerified()
    owner.session.use()
    const file = await api.uploadFile(makeFile('публичный.txt', 'всем видно'))
    const link = await api.createLink('file', file.id)
    expect(link.code).toBeTruthy()

    // Код — capability: сессия для чтения не нужна. Гость — чистая сессия
    // без входа: ни токена, ни cookie.
    new Session('гость').use()
    const shared = await api.getShared(link.code)
    expect(shared.file?.id).toBe(file.id)

    owner.session.use()
    await api.deleteLink(link.id)
    await expectStatus(api.getShared(link.code), 404)
  })

  it('ссылку на чужой файл не создать', async () => {
    const owner = await registerVerified()
    owner.session.use()
    const file = await api.uploadFile(makeFile('чужой.txt'))

    const stranger = await registerVerified()
    stranger.session.use()
    await expectStatus(api.createLink('file', file.id), 403)
  })

  it('большой файл уезжает частями и собирается целиком', async () => {
    const u = await registerVerified()
    u.session.use()

    // Больше CHUNK_THRESHOLD — клиент сам переключается на загрузку частями.
    const size = api.CHUNK_THRESHOLD + api.CHUNK_SIZE + 1024
    const blob = new Blob([new Uint8Array(size).fill(7)], { type: 'application/octet-stream' })
    const big = new File([blob], uniq('большой') + '.bin', { type: 'application/octet-stream' })

    const steps = []
    const created = await api.uploadFile(big, null, { onProgress: (p) => steps.push(p) })

    expect(created.size).toBe(size)
    // Прогресс шёл кусками и дошёл до конца — иначе полоса не двигалась бы.
    expect(steps.length).toBeGreaterThan(1)
    expect(steps.at(-1)).toBe(1)

    // Файл читается целиком: собранное из кусков совпадает по объёму.
    // Идём ручкой сервиса (она несёт токен) — статику /uploads в стенде
    // раздавать некому.
    const stored = await apiRequest(`/drive/files/${created.id}/download`, { blob: true })
    expect((await stored.arrayBuffer()).byteLength).toBe(size)
  })

  it('брошенная загрузка частями отменяется, а её кусок не становится файлом', async () => {
    const u = await registerVerified()
    u.session.use()

    const before = (await api.browse({})).files.length
    // Загрузка частями — общий контракт платформы: init → chunk → finish.
    const started = await apiRequest('/drive/uploads/init', {
      method: 'POST',
      body: { file_name: 'брошенный.bin', size: 1024, mime: 'application/octet-stream' },
    })
    expect(started.code).toBeTruthy()

    await apiRequest(`/drive/uploads/${started.code}`, { method: 'DELETE' })
    // Отменённую загрузку не собрать: заготовки больше нет.
    await expectStatus(
      apiRequest(`/drive/uploads/${started.code}/finish`, { method: 'POST' }),
      404,
    )
    expect((await api.browse({})).files.length).toBe(before)
  })
})
