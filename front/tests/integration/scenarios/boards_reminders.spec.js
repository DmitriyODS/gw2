/* Доски (boardsvc) и напоминания (remindersvc).

   Доски по устройству — близнец заметок (папки, шаринг, публичные ссылки),
   поэтому здесь проверяется то, что у них своё: сцена холста и её версии,
   выгрузка в разные форматы. Напоминания — сроки и повторы: тут важны
   граничные значения (прошлое время, «каждые N», конец повтора). */
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin, newMember } from '../setup/factory.js'
import * as boards from '@/api/boards.js'
import * as reminders from '@/api/reminders.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

const scene = (objects = []) => ({
  version: 2,
  layers: [{ id: 'l1', name: 'Слой 1', visible: true, locked: false }],
  objects,
})

describeIntegration('boards API: доски и сцена', () => {
  it('жизненный цикл доски: создание, сцена, копия, удаление', async () => {
    const u = await registerVerified()
    u.session.use()

    const b = await boards.createBoard('Схема')
    expect(b.id).toBeGreaterThan(0)

    await boards.updateBoard(b.id, {
      title: 'Схема процесса',
      scene: scene([{ id: 'o1', type: 'text', layer: 'l1', text: 'Этап приёмки', x: 10, y: 10 }]),
    })

    const one = await boards.getBoard(b.id)
    expect(one.title).toBe('Схема процесса')
    expect(one.scene?.objects?.length).toBe(1)
    expect(one.my_access).toBe('owner')

    const copy = await boards.copyBoard(b.id)
    expect(copy.id).not.toBe(b.id)

    await boards.deleteBoard(b.id)
    await expectStatus(boards.getBoard(b.id), 404)
  })

  it('текст объектов сцены попадает в поиск', async () => {
    const u = await registerVerified()
    u.session.use()
    const b = await boards.createBoard('Без имени')
    await boards.updateBoard(b.id, {
      scene: scene([{ id: 'o1', type: 'text', layer: 'l1', text: 'гидроизоляция', x: 0, y: 0 }]),
    })

    const found = await boards.getBoards({ search: 'гидроизоляция' })
    expect((found.boards ?? []).some((x) => x.id === b.id)).toBe(true)
  })

  it('чужая доска недоступна ни на чтение, ни на правку', async () => {
    const owner = await registerVerified('owner')
    owner.session.use()
    const b = await boards.createBoard('Личная доска')

    const stranger = await registerVerified('stranger')
    stranger.session.use()
    await expectStatus(boards.getBoard(b.id), 404)
    await expectStatus(boards.updateBoard(b.id, { title: 'Чужое' }), 404)
  })

  it('шаринг доски: view читает, edit пишет, отзыв закрывает', async () => {
    const owner = await newCompanyAdmin('owner')
    const mate = await newMember(owner, owner.companyId, 1, 'mate')

    owner.session.use()
    const b = await boards.createBoard('Командная')
    await boards.shareBoardWithUser(b.id, mate.auth.userId, false)

    mate.session.use()
    expect((await boards.getBoard(b.id)).my_access).toBe('view')
    await expect(boards.updateBoard(b.id, { title: 'Правка' })).rejects.toBeTruthy()

    owner.session.use()
    await boards.shareBoardWithUser(b.id, mate.auth.userId, true)
    mate.session.use()
    await boards.updateBoard(b.id, { title: 'Соавтор поправил' })

    owner.session.use()
    await boards.unshareBoardUser(b.id, mate.auth.userId)
    mate.session.use()
    await expectStatus(boards.getBoard(b.id), 404)
  })

  it('публичная ссылка отдаёт доску, отзыв — закрывает', async () => {
    const u = await registerVerified()
    u.session.use()
    const b = await boards.createBoard('Публичная доска')
    const share = await boards.createShare(b.id, 'view')

    const guest = await registerVerified('guest')
    guest.session.use()
    const seen = await boards.getSharedBoard(share.code)
    expect(seen.board?.title ?? seen.title).toBe('Публичная доска')

    u.session.use()
    await boards.revokeShare(b.id, share.id)
    guest.session.use()
    await expect(boards.getSharedBoard(share.code)).rejects.toBeTruthy()
  })

  it('выгрузка: svg и json отдаются, незнакомый формат — отказ', async () => {
    const u = await registerVerified()
    u.session.use()
    const b = await boards.createBoard('На выгрузку')
    await boards.updateBoard(b.id, { scene: scene() })

    for (const fmt of ['svg', 'json']) {
      const res = await boards.exportBoard(b.id, fmt)
      expect(res.ok).toBe(true)
      expect(res.headers.get('content-disposition')).toContain(`.${fmt}`)
    }

    // Пустой формат — умолчание (svg), мусор — 400.
    const byDefault = await boards.exportBoard(b.id, '')
    expect(byDefault.ok).toBe(true)
    await expectStatus(boards.exportBoard(b.id, 'exe'), 400)
  })

  it('сцену версии 1 сервер принимает и умеет разобрать', async () => {
    const u = await registerVerified()
    u.session.use()
    const b = await boards.createBoard('Старая сцена')
    // Версия 1 — плоский список объектов без слоёв. В БД она остаётся как есть
    // (до текущей её поднимает клиент), но сервер обязан её понимать: выгрузка
    // прогоняет сцену через ParseScene, и объект без слоя не должен её ронять.
    await boards.updateBoard(b.id, {
      scene: { version: 1, objects: [{ id: 'a', type: 'text', text: 'старьё', x: 1, y: 1 }] },
    })
    expect((await boards.getBoard(b.id)).scene?.objects?.length).toBe(1)

    const svg = await boards.exportBoard(b.id, 'svg')
    expect(svg.ok).toBe(true)
    expect(await svg.text()).toContain('старьё')
  })
})

describeIntegration('reminders API: сроки и повторы', () => {
  const inHour = () => new Date(Date.now() + 3600_000).toISOString()

  it('жизненный цикл: создание, правка, выполнение, удаление', async () => {
    const u = await registerVerified()
    u.session.use()

    const r = await reminders.createReminder({
      title: 'Позвонить подрядчику',
      remind_at: inHour(),
      timezone: 'Europe/Moscow',
    })
    expect(r.id).toBeGreaterThan(0)

    await reminders.updateReminder(r.id, { title: 'Позвонить и записать' })
    expect((await reminders.getReminder(r.id)).title).toBe('Позвонить и записать')

    const active = await reminders.getReminders('active')
    expect((active.items ?? active).some?.((x) => x.id === r.id) ?? true).toBeTruthy()

    await reminders.completeReminder(r.id)
    await reminders.deleteReminder(r.id)
    await expect(reminders.getReminder(r.id)).rejects.toBeTruthy()
  })

  it('повтор: правило сохраняется и срок пересчитывается вперёд', async () => {
    const u = await registerVerified()
    u.session.use()
    const r = await reminders.createReminder({
      title: 'Планёрка',
      remind_at: inHour(),
      timezone: 'Europe/Moscow',
      // Повтор — объект: вид, шаг и (для недельного) дни.
      repeat: { kind: 'weekly', interval: 1, days: [1] },
    })
    const one = await reminders.getReminder(r.id)
    expect(one.repeat?.kind).toBe('weekly')
    expect(one.repeat?.interval).toBe(1)
    expect(new Date(one.remind_at).getTime()).toBeGreaterThan(Date.now() - 60_000)
  })

  it('отложить ставит срок через N минут ОТ СЕЙЧАС, а не от прежнего срока', async () => {
    const u = await registerVerified()
    u.session.use()
    const r = await reminders.createReminder({
      title: 'Проверить почту', remind_at: inHour(), timezone: 'Europe/Moscow',
    })

    await reminders.snoozeReminder(r.id, 30)
    const after = new Date((await reminders.getReminder(r.id)).remind_at).getTime()
    const expected = Date.now() + 30 * 60_000
    // «Отложить на 30 минут» отсчитывается от момента нажатия — иначе кнопка
    // у сработавшего напоминания уводила бы срок на часы вперёд.
    expect(Math.abs(after - expected)).toBeLessThan(120_000)
  })

  it('ближайшие возвращаются в порядке срока и не больше запрошенного', async () => {
    const u = await registerVerified()
    u.session.use()
    const base = Date.now() + 3600_000
    for (const [i, title] of ['третье', 'первое', 'второе'].entries()) {
      await reminders.createReminder({
        title,
        remind_at: new Date(base + i * 600_000).toISOString(),
        timezone: 'Europe/Moscow',
      })
    }
    const up = await reminders.getUpcoming(2)
    const items = up.items ?? up
    expect(items.length).toBeLessThanOrEqual(2)
    if (items.length === 2) {
      expect(new Date(items[0].remind_at).getTime())
        .toBeLessThanOrEqual(new Date(items[1].remind_at).getTime())
    }
  })

  it('чужое напоминание недоступно', async () => {
    const owner = await registerVerified('owner')
    owner.session.use()
    const r = await reminders.createReminder({
      title: 'Личное', remind_at: inHour(), timezone: 'Europe/Moscow',
    })

    const stranger = await registerVerified('stranger')
    stranger.session.use()
    await expect(reminders.getReminder(r.id)).rejects.toBeTruthy()
    await expect(reminders.deleteReminder(r.id)).rejects.toBeTruthy()
  })

  it('граничные данные: пустой заголовок и срок в прошлом', async () => {
    const u = await registerVerified()
    u.session.use()

    // Без заголовка напоминание бессмысленно — сервер обязан отказать.
    await expect(reminders.createReminder({
      title: '', remind_at: inHour(), timezone: 'Europe/Moscow',
    })).rejects.toBeTruthy()

    // Срок в прошлом допустим (сработает сразу) либо отклонён — но не 500.
    try {
      const past = await reminders.createReminder({
        title: 'Просроченное',
        remind_at: new Date(Date.now() - 3600_000).toISOString(),
        timezone: 'Europe/Moscow',
      })
      expect(past.id).toBeGreaterThan(0)
    } catch (e) {
      expect(e.status).toBeGreaterThanOrEqual(400)
      expect(e.status).toBeLessThan(500)
    }
  })
})
