/* Реестры, календари и ежедневники: правка и удаление записей, выгрузки,
   публичные ссылки и адресный доступ — всё, чего не покрывал первый сценарий.

   Три раздела построены на одном ядре (pkg/records), но границы у них разные:
   реестр и календарь принадлежат КОМПАНИИ, ежедневник — ЧЕЛОВЕКУ. Отсюда и
   главные проверки: чужое не читается и не правится, публичная ссылка даёт
   только чтение и гаснет при отзыве, а адресный доступ к ежедневнику позволяет
   отмечать выполненное, но не переписывать чужие дела. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq, Session } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin, newMember } from '../setup/factory.js'
import * as reg from '@/api/registries.js'
import * as cal from '@/api/calendars.js'
import * as diaries from '@/api/diaries.js'
import * as tasks from '@/api/tasks.js'
import * as departments from '@/api/departments.js'

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

function fieldId(fields, label) {
  return String(fields.find((f) => f.label === label).id)
}

const ymd = (d) => d.toISOString().slice(0, 10)
const at = (day, time) => `${day}T${time}:00Z` // календарь принимает RFC3339
const today = ymd(new Date())

async function registryWithField(admin, label = 'Название') {
  admin.session.use()
  const r = await reg.createRegistry(uniq('Реестр '))
  const put = await reg.replaceFields(r.id, [
    { label, type: 'text', show_in_table: true },
  ])
  return { id: r.id, field: fieldId(put.fields, label) }
}

async function calendarWithField(admin, label = 'Тема') {
  admin.session.use()
  const c = await cal.createCalendar(uniq('Календарь '))
  const put = await cal.replaceFields(c.id, [
    { label, type: 'text', show_in_table: true },
  ])
  return { id: c.id, field: fieldId(put.fields, label) }
}

describeIntegration('registries API: записи, выгрузка, доступ', () => {
  it('запись правится, читается поштучно и удаляется', async () => {
    const admin = await newCompanyAdmin()
    const { id, field } = await registryWithField(admin)

    const rec = await reg.createRecord(id, { [field]: 'Первая' })
    await reg.updateRecord(id, rec.id, { [field]: 'Исправленная' })
    const one = await reg.getRecord(id, rec.id)
    expect(one.data[field]).toBe('Исправленная')

    await reg.deleteRecord(id, rec.id)
    await expectClientError(reg.getRecord(id, rec.id))
  })

  it('пакетное удаление уносит только указанные записи', async () => {
    const admin = await newCompanyAdmin()
    const { id, field } = await registryWithField(admin)
    const a = await reg.createRecord(id, { [field]: 'А' })
    const b = await reg.createRecord(id, { [field]: 'Б' })
    const c = await reg.createRecord(id, { [field]: 'В' })

    await reg.bulkDeleteRecords(id, [a.id, b.id])
    const left = await reg.getRecords(id)
    expect(left.items.map((r) => r.id)).toEqual([c.id])
  })

  it('переименование и удаление реестра', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    const r = await reg.createRegistry(uniq('Старое имя '))
    await reg.updateRegistry(r.id, 'Новое имя')
    expect((await reg.getRegistry(r.id)).name).toBe('Новое имя')

    await reg.deleteRegistry(r.id)
    await expectClientError(reg.getRegistry(r.id))
  })

  it('выгрузка отдаёт файл', async () => {
    const admin = await newCompanyAdmin()
    const { id, field } = await registryWithField(admin)
    await reg.createRecord(id, { [field]: 'Для выгрузки' })
    const res = await reg.exportRecords(id, {})
    expect(res.ok).toBe(true)
  })

  it('сквозной поиск находит запись по любому реестру компании', async () => {
    const admin = await newCompanyAdmin()
    const { id, field } = await registryWithField(admin)
    const marker = `маркер${Date.now().toString().slice(-5)}`
    await reg.createRecord(id, { [field]: marker })

    const found = await reg.searchRecords(marker)
    expect((found.items ?? found.records ?? []).length).toBeGreaterThan(0)
    const none = await reg.searchRecords('заведомо-небывалое-слово')
    expect((none.items ?? none.records ?? []).length).toBe(0)
  })

  it('реестр чужой компании не читается и не правится', async () => {
    const a = await newCompanyAdmin('a')
    const { id, field } = await registryWithField(a)
    const rec = await reg.createRecord(id, { [field]: 'Секрет' })

    const b = await newCompanyAdmin('b')
    b.session.use()
    await expectClientError(reg.getRegistry(id))
    await expectClientError(reg.getRecords(id))
    await expectClientError(reg.updateRecord(id, rec.id, { [field]: 'Подмена' }))
    await expectClientError(reg.deleteRegistry(id))
  })

  it('структуру правит администратор, записи ведёт любой участник', async () => {
    const admin = await newCompanyAdmin('admin')
    const { id, field } = await registryWithField(admin)
    const worker = await newMember(admin, admin.companyId, 1, 'worker')

    worker.session.use()
    // Записи — работа участника…
    const rec = await reg.createRecord(id, { [field]: 'Строка сотрудника' })
    expect(rec.id).toBeGreaterThan(0)
    // …а структура справочника — дело администратора.
    await expectClientError(reg.replaceFields(id, [{ label: 'Своё поле', type: 'text' }]))
    await expectClientError(reg.deleteRegistry(id))
  })

  it('публичная ссылка даёт чтение и гаснет при отзыве', async () => {
    const admin = await newCompanyAdmin()
    const { id, field } = await registryWithField(admin)
    await reg.createRecord(id, { [field]: 'Открытая строка' })

    const share = await reg.createShare(id)
    expect(share.code).toBeTruthy()
    const shares = await reg.getShares(id)
    expect(shares.shares.some((s) => s.code === share.code)).toBe(true)

    // Ссылку открывают без входа — это её смысл.
    const guest = new Session('guest')
    guest.use()
    const shared = await reg.getSharedRegistry(share.code)
    expect(shared.fields.length).toBe(1)
    const records = await reg.getSharedRecords(share.code)
    expect(records.items.length).toBe(1)
    expect((await reg.exportSharedRecords(share.code, {})).ok).toBe(true)

    admin.session.use()
    await reg.revokeShare(id, share.id ?? (await reg.getShares(id)).shares[0].id)

    guest.use()
    await expectClientError(reg.getSharedRegistry(share.code))
  })

  it('по чужому коду ничего не открывается', async () => {
    const guest = new Session('guest')
    guest.use()
    await expectClientError(reg.getSharedRegistry('нет-такого-кода'))
    await expectClientError(reg.getSharedRecords('нет-такого-кода'))
  })
})

describeIntegration('calendars API: записи, период, доступ', () => {
  it('запись создаётся со временем, правится и удаляется', async () => {
    const admin = await newCompanyAdmin()
    const { id, field } = await calendarWithField(admin)
    const entry = await cal.createEntry(id, at(today, '10:00'), { [field]: 'Планёрка' })
    expect(entry.id).toBeGreaterThan(0)

    await cal.updateEntry(id, entry.id, at(today, '11:30'), { [field]: 'Планёрка перенесена' })
    const one = await cal.getEntry(id, entry.id)
    expect(one.data[field]).toBe('Планёрка перенесена')

    await cal.deleteEntry(id, entry.id)
    await expectClientError(cal.getEntry(id, entry.id))
  })

  it('без даты и времени запись не создаётся', async () => {
    const admin = await newCompanyAdmin()
    const { id, field } = await calendarWithField(admin)
    // «Дата и время» — встроенное обязательное поле календаря.
    await expectClientError(cal.createEntry(id, '', { [field]: 'Без даты' }))
  })

  it('выборка идёт по диапазону дат', async () => {
    const admin = await newCompanyAdmin()
    const { id, field } = await calendarWithField(admin)

    const tomorrow = ymd(new Date(Date.now() + 86400000))
    const nextWeek = ymd(new Date(Date.now() + 7 * 86400000))
    await cal.createEntry(id, at(tomorrow, '09:00'), { [field]: 'Завтра' })
    await cal.createEntry(id, at(nextWeek, '09:00'), { [field]: 'Через неделю' })

    // Границы периода уходят в ISO — так их шлёт раздел (range → toISOString).
    const near = await cal.getEntries(id, { from: at(today, '00:00'), to: at(tomorrow, '23:59') })
    expect(near.items.length).toBe(1)
    expect(near.items[0].data[field]).toBe('Завтра')
  })

  it('живая сводка ближайших событий собирает все календари компании', async () => {
    const admin = await newCompanyAdmin()
    const first = await calendarWithField(admin, 'Тема')
    const second = await calendarWithField(admin, 'Тема')

    const soon = at(today, '23:00')
    await cal.createEntry(first.id, soon, { [first.field]: 'Из первого' })
    await cal.createEntry(second.id, soon, { [second.field]: 'Из второго' })

    const agenda = await cal.getAgenda(
      at(today, '00:00'), at(ymd(new Date(Date.now() + 3 * 86400000)), '23:59'), 10)
    const items = agenda.items ?? agenda.entries ?? []
    // Заголовок карточки считает сервер — плитке не с чем работать без него.
    expect(items.length).toBeGreaterThanOrEqual(2)
    expect(items.every((i) => typeof (i.title ?? '') === 'string')).toBe(true)
  })

  it('пакетное удаление и выгрузка за период', async () => {
    const admin = await newCompanyAdmin()
    const { id, field } = await calendarWithField(admin)
    const a = await cal.createEntry(id, at(today, '08:00'), { [field]: 'Раз' })
    const b = await cal.createEntry(id, at(today, '09:00'), { [field]: 'Два' })

    expect((await cal.exportEntries(id, { from: at(today, '00:00'), to: at(today, '23:59') })).ok).toBe(true)

    await cal.bulkDeleteEntries(id, [a.id, b.id])
    const left = await cal.getEntries(id, { from: at(today, '00:00'), to: at(today, '23:59') })
    expect(left.items.length).toBe(0)
  })

  it('переименование, удаление и чужой календарь', async () => {
    const a = await newCompanyAdmin('a')
    const { id } = await calendarWithField(a)
    await cal.updateCalendar(id, 'Переименованный')
    expect((await cal.getCalendar(id)).name).toBe('Переименованный')

    const b = await newCompanyAdmin('b')
    b.session.use()
    await expectClientError(cal.getCalendar(id))
    await expectClientError(cal.deleteCalendar(id))

    a.session.use()
    await cal.deleteCalendar(id)
    await expectClientError(cal.getCalendar(id))
  })

  it('публичная ссылка на календарь: чтение и выгрузка без входа', async () => {
    const admin = await newCompanyAdmin()
    const { id, field } = await calendarWithField(admin)
    await cal.createEntry(id, at(today, '12:00'), { [field]: 'Открытое событие' })

    const share = await cal.createShare(id)
    const guest = new Session('guest')
    guest.use()
    expect((await cal.getSharedCalendar(share.code)).fields.length).toBe(1)
    const entries = await cal.getSharedEntries(share.code, { from: at(today, '00:00'), to: at(today, '23:59') })
    expect(entries.items.length).toBe(1)
    expect((await cal.exportSharedEntries(share.code, {
      from: at(today, '00:00'), to: at(today, '23:59'),
    })).ok).toBe(true)

    admin.session.use()
    const shares = await cal.getShares(id)
    await cal.revokeShare(id, shares.shares[0].id)
    guest.use()
    await expectClientError(cal.getSharedCalendar(share.code))
  })
})

describeIntegration('diaries API: дела, перенос и выполнение', () => {
  it('дело заводится, правится, отмечается и удаляется', async () => {
    const u = await registerVerified()
    u.session.use()
    const diary = await diaries.createDiary(uniq('Ежедневник '))

    const entry = await diaries.createEntry(diary.id, { entry_date: today, title: 'Позвонить в банк' })
    expect(entry.id).toBeGreaterThan(0)

    await diaries.updateEntry(diary.id, entry.id, {
      entry_date: today, title: 'Позвонить в банк до обеда', description: '',
    })
    expect((await diaries.getEntry(diary.id, entry.id)).title).toContain('до обеда')

    await diaries.setEntryDone(diary.id, entry.id, true)
    const active = await diaries.getEntries(diary.id, { archived: 0 })
    expect((active.items ?? []).some((e) => e.id === entry.id)).toBe(false)

    // Выполненное уходит в архив, а не исчезает.
    const archived = await diaries.getEntries(diary.id, { archived: 1 })
    expect((archived.items ?? []).some((e) => e.id === entry.id)).toBe(true)

    await diaries.setEntryDone(diary.id, entry.id, false)
    await diaries.deleteEntry(diary.id, entry.id)
    await expectClientError(diaries.getEntry(diary.id, entry.id))
  })

  it('дело переносится на другой день и в другой свой ежедневник', async () => {
    const u = await registerVerified()
    u.session.use()
    const from = await diaries.createDiary(uniq('Рабочий '))
    const to = await diaries.createDiary(uniq('Личный '))
    const entry = await diaries.createEntry(from.id, { entry_date: today, title: 'Переносимое' })

    const tomorrow = ymd(new Date(Date.now() + 86400000))
    await diaries.moveEntry(from.id, entry.id, { entry_date: tomorrow, diary_id: to.id })

    const moved = await diaries.getEntries(to.id, { archived: 0 })
    expect((moved.items ?? []).some((e) => e.id === entry.id)).toBe(true)
    const left = await diaries.getEntries(from.id, { archived: 0 })
    expect((left.items ?? []).some((e) => e.id === entry.id)).toBe(false)
  })

  it('порядок дел за день сохраняется', async () => {
    const u = await registerVerified()
    u.session.use()
    const diary = await diaries.createDiary(uniq('Порядок '))
    const first = await diaries.createEntry(diary.id, { entry_date: today, title: 'Первое' })
    const second = await diaries.createEntry(diary.id, { entry_date: today, title: 'Второе' })

    await diaries.reorderEntries(diary.id, today, [second.id, first.id])
    const list = await diaries.getEntries(diary.id, { archived: 0 })
    const ids = (list.items ?? []).map((e) => e.id)
    expect(ids.indexOf(second.id)).toBeLessThan(ids.indexOf(first.id))
  })

  it('к делу привязывается задача', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    const dept = await departments.createDepartment({ name: uniq('Отдел ') })
    const task = await tasks.createTask({ name: 'Связанная задача', department_id: dept.id })

    const diary = await diaries.createDiary(uniq('Связи '))
    const entry = await diaries.createEntry(diary.id, { entry_date: today, title: 'С задачей' })
    await diaries.linkEntryTask(diary.id, entry.id, task.id)

    const one = await diaries.getEntry(diary.id, entry.id)
    expect(one.linked_task_id ?? one.task?.id).toBe(task.id)
  })

  it('сводка дел и поиск по всем своим ежедневникам', async () => {
    const u = await registerVerified()
    u.session.use()
    const diary = await diaries.createDiary(uniq('Поиск '))
    const marker = `дело${Date.now().toString().slice(-5)}`
    await diaries.createEntry(diary.id, { entry_date: today, title: marker })

    const agenda = await diaries.getAgenda(today, ymd(new Date(Date.now() + 86400000)), 10)
    expect((agenda.items ?? []).length).toBeGreaterThan(0)

    const found = await diaries.searchEntries(marker)
    expect((found.items ?? []).length).toBeGreaterThan(0)
  })

  it('пакетное удаление и выгрузка', async () => {
    const u = await registerVerified()
    u.session.use()
    const diary = await diaries.createDiary(uniq('Выгрузка '))
    const a = await diaries.createEntry(diary.id, { entry_date: today, title: 'Раз' })
    const b = await diaries.createEntry(diary.id, { entry_date: today, title: 'Два' })

    expect((await diaries.exportEntries(diary.id, {})).ok).toBe(true)

    await diaries.bulkDeleteEntries(diary.id, [a.id, b.id])
    expect(((await diaries.getEntries(diary.id, { archived: 0 })).items ?? []).length).toBe(0)
  })

  it('переименование и удаление ежедневника', async () => {
    const u = await registerVerified()
    u.session.use()
    const diary = await diaries.createDiary(uniq('Старый '))
    await diaries.updateDiary(diary.id, 'Новый')
    expect((await diaries.getDiary(diary.id)).name).toBe('Новый')
    await diaries.deleteDiary(diary.id)
    await expectClientError(diaries.getDiary(diary.id))
  })
})

describeIntegration('diaries API: чужой доступ и шаринг', () => {
  it('чужой ежедневник закрыт целиком', async () => {
    const owner = await registerVerified('owner')
    owner.session.use()
    const diary = await diaries.createDiary(uniq('Личное '))
    const entry = await diaries.createEntry(diary.id, { entry_date: today, title: 'Моё дело' })

    const stranger = await registerVerified('stranger')
    stranger.session.use()
    await expectClientError(diaries.getDiary(diary.id))
    await expectClientError(diaries.getEntries(diary.id, {}))
    await expectClientError(diaries.updateEntry(diary.id, entry.id, { entry_date: today, title: 'Подмена' }))
    await expectClientError(diaries.deleteDiary(diary.id))
  })

  it('адресный доступ: видит и отмечает, но не переписывает', async () => {
    const owner = await registerVerified('owner')
    const mate = await registerVerified('mate')

    owner.session.use()
    const diary = await diaries.createDiary(uniq('Раздача '))
    const entry = await diaries.createEntry(diary.id, { entry_date: today, title: 'Поручение' })
    await diaries.addMember(diary.id, mate.auth.userId, true)

    const members = await diaries.getMembers(diary.id)
    expect(members.members.some((m) => (m.user_id ?? m.user?.id) === mate.auth.userId)).toBe(true)

    mate.session.use()
    const shared = await diaries.getDiaries('shared')
    expect((shared.diaries ?? []).some((d) => d.id === diary.id)).toBe(true)

    // Отмечать разрешено (сценарий «руководитель раздаёт задачи»)…
    await diaries.setEntryDone(diary.id, entry.id, true)
    // …а править и удалять — нет: это чужой ежедневник.
    await expectClientError(diaries.updateEntry(diary.id, entry.id, { entry_date: today, title: 'Переписал' }))
    await expectClientError(diaries.deleteEntry(diary.id, entry.id))

    owner.session.use()
    await diaries.removeMember(diary.id, mate.auth.userId)
    mate.session.use()
    await expectClientError(diaries.getEntries(diary.id, {}))
  })

  it('доступ без права отметки не позволяет её ставить', async () => {
    const owner = await registerVerified('owner')
    const mate = await registerVerified('mate')

    owner.session.use()
    const diary = await diaries.createDiary(uniq('Только чтение '))
    const entry = await diaries.createEntry(diary.id, { entry_date: today, title: 'Читаемое' })
    await diaries.addMember(diary.id, mate.auth.userId, false)

    mate.session.use()
    expect(((await diaries.getEntries(diary.id, {})).items ?? []).length).toBe(1)
    await expectClientError(diaries.setEntryDone(diary.id, entry.id, true))
  })

  it('публичная ссылка на ежедневник — только чтение', async () => {
    const owner = await registerVerified('owner')
    owner.session.use()
    const diary = await diaries.createDiary(uniq('Публичный '))
    await diaries.createEntry(diary.id, { entry_date: today, title: 'Видно всем' })

    const share = await diaries.createShare(diary.id)
    const guest = new Session('guest')
    guest.use()
    expect((await diaries.getSharedDiary(share.code)).name).toContain('Публичный')
    expect(((await diaries.getSharedEntries(share.code, {})).items ?? []).length).toBe(1)
    expect((await diaries.exportSharedEntries(share.code, {})).ok).toBe(true)

    owner.session.use()
    const shares = await diaries.getShares(diary.id)
    await diaries.revokeShare(diary.id, shares.shares[0].id)
    guest.use()
    await expectClientError(diaries.getSharedDiary(share.code))
  })
})
