/* Загрузка файлов во ВСЕХ разделах, где она есть, — через api-модули фронта
   против живых сервисов.

   Проверяем оба пути сразу: мелкий файл уходит одним запросом, крупный —
   ЧАСТЯМИ (pkg/chunkupload). Механика частей общая, но подключена к шести
   разделам поодиночке, и разойтись они могут незаметно: раздел собирается,
   тесты сервиса зелены, а файл на сборке теряется. Отсюда один сценарий на
   всех — он же сторожит, что порог и контракт ручек не разъехались. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { apiRequest } from '@/api/client.js'
import { newCompanyAdmin } from '../setup/factory.js'
import { CHUNK_THRESHOLD } from '@/utils/chunkUpload.js'
import * as reg from '@/api/registries.js'
import * as cal from '@/api/calendars.js'
import * as notes from '@/api/notes.js'
import * as boards from '@/api/boards.js'
import * as portal from '@/api/portal.js'
import * as drive from '@/api/drive.js'
import * as messenger from '@/api/messenger.js'

/* Файл заведомо больше порога — значит уедет частями. Байты кладём узором, а
   не нулями: так сборка «не в том порядке» видна по содержимому, а не только
   по длине. */
function bigFile(name, mime = 'application/octet-stream') {
  const size = CHUNK_THRESHOLD + 1024 * 1024 // порог + мегабайт
  const bytes = new Uint8Array(size)
  for (let i = 0; i < size; i++) bytes[i] = i % 251
  return new File([bytes], name, { type: mime })
}

function smallFile(name, mime = 'text/plain') {
  return new File([new Uint8Array(1024).fill(7)], name, { type: mime })
}

/* Скачать собранный файл и сверить его С УЗОРОМ. Это главная проверка чанкового
   пути: длина сойдётся и при перепутанном порядке частей, а содержимое — нет.
   Ручка скачивания есть у диска, поэтому побайтно сверяем на нём; остальным
   разделам достаточно того, что запись потока в хранилище сама проверяет
   записанный объём (storage.PutStream) и падает при расхождении. */
async function expectSameBytes(fileId, size) {
  const resp = await apiRequest(`/drive/files/${fileId}/download`, { blob: true, timeout: 120000 })
  expect(resp.ok).toBe(true)
  const body = Buffer.from(await resp.arrayBuffer())
  expect(body.length).toBe(size)
  for (const i of [0, 1, 4095, 5 * 1024 * 1024, 5 * 1024 * 1024 + 1, size - 1]) {
    expect(body[i]).toBe(i % 251)
  }
}

describeIntegration('загрузка файлов: одним запросом и частями', () => {
  it('реестры: обычный файл и файл частями', async () => {
    const admin = await newCompanyAdmin('upreg')
    admin.session.use()
    const r = await reg.createRegistry(uniq('Файлы '))

    const small = await reg.uploadFile(r.id, smallFile('note.txt'))
    expect(small.path).toContain('registry/')
    expect(small.size).toBe(1024)

    const big = bigFile('big.bin')
    const seen = []
    const out = await reg.uploadFile(r.id, big, { onProgress: (p) => seen.push(p) })
    expect(out.path).toContain('registry/')
    expect(out.size).toBe(big.size)
    // Прогресс идёт по частям и доходит до конца — иначе индикатор врёт.
    expect(seen.length).toBeGreaterThan(1)
    expect(seen.at(-1)).toBe(1)
  })

  it('календари: обычный файл и файл частями', async () => {
    const admin = await newCompanyAdmin('upcal')
    admin.session.use()
    await cal.createCalendar(uniq('Файлы '))

    expect((await cal.uploadFile(smallFile('note.txt'))).path).toContain('calendar/')
    const big = bigFile('big.bin')
    const out = await cal.uploadFile(big)
    expect(out.path).toContain('calendar/')
    expect(out.size).toBe(big.size)
  })

  it('заметки: картинка обычная и частями', async () => {
    const admin = await newCompanyAdmin('upnote')
    admin.session.use()
    const note = await notes.createNote(uniq('Заметка '))

    const small = await notes.uploadImage(note.id, smallFile('pic.png', 'image/png'))
    expect(small.path).toContain('/uploads/notes/')

    const big = bigFile('big.png', 'image/png')
    const out = await notes.uploadImage(note.id, big)
    // Форма ответа не зависит от размера файла — иначе редактор ломался бы
    // ровно на больших картинках.
    expect(out.path).toContain('/uploads/notes/')
  })

  it('доски: картинка обычная и частями', async () => {
    const admin = await newCompanyAdmin('upboard')
    admin.session.use()
    const board = await boards.createBoard(uniq('Доска '))

    expect((await boards.uploadImage(board.id, smallFile('pic.png', 'image/png'))).path)
      .toContain('/uploads/boards/')

    const big = bigFile('big.png', 'image/png')
    const out = await boards.uploadImage(board.id, big)
    expect(out.path).toContain('/uploads/boards/')
  })

  it('портал: вложение поста обычное и частями', async () => {
    const admin = await newCompanyAdmin('upportal')
    admin.session.use()
    const post = await portal.createPost({ body: uniq('Пост ') })

    const small = await portal.uploadAttachment(post.id, smallFile('doc.txt'))
    expect(small.file_path).toContain('portal/')

    const big = bigFile('big.bin')
    const out = await portal.uploadAttachment(post.id, big)
    expect(out.size).toBe(big.size)
  })

  it('диск: файл обычный и частями', async () => {
    const admin = await newCompanyAdmin('updrive')
    admin.session.use()

    const small = await drive.uploadFile(smallFile('note.txt'))
    expect(small.id).toBeGreaterThan(0)
    expect(small.size).toBe(1024)

    const big = bigFile('big.bin')
    const out = await drive.uploadFile(big)
    expect(out.size).toBe(big.size)
    // Побайтная сверка собранного файла: части встали в правильном порядке.
    await expectSameBytes(out.id, big.size)

    // Файл виден в разделе — запись о нём завелась, а не только объект.
    const list = await drive.browse({})
    expect(list.files.some((f) => f.id === out.id)).toBe(true)
  })

  it('мессенджер: вложение обычное и частями', async () => {
    const admin = await newCompanyAdmin('upmsg')
    admin.session.use()

    const small = await messenger.uploadAttachment(smallFile('note.txt'))
    expect(small.id).toBeGreaterThan(0)

    const big = bigFile('big.bin')
    const out = await messenger.uploadAttachment(big)
    expect(out.id).toBeGreaterThan(0)
    expect(out.size_bytes ?? out.size).toBe(big.size)
  })

  it('брошенная загрузка отменяется и не мешает следующей', async () => {
    const admin = await newCompanyAdmin('upcancel')
    admin.session.use()
    const r = await reg.createRegistry(uniq('Отмена '))

    // Отмена на середине: сервер должен принять её и не оставить сессию.
    const ctrl = new AbortController()
    const big = bigFile('big.bin')
    const failed = reg.uploadFile(r.id, big, {
      signal: ctrl.signal,
      onProgress: (p) => { if (p > 0) ctrl.abort() },
    })
    await expect(failed).rejects.toBeTruthy()

    // Следующая загрузка проходит как ни в чём не бывало.
    const out = await reg.uploadFile(r.id, bigFile('again.bin'))
    expect(out.size).toBe(big.size)
  })
})
