// Сценарий «ежедневники» через useDiariesStore против живого diarysvc:
// список/создание/записи/выполнение/перенос/реордер + адресный шаринг с правом
// отметки (can_check) и проверка canToggle у адресата.
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { registerVerified } from '../setup/factory.js'
import { useDiariesStore, dayKey } from '@/stores/diaries.js'
import * as diariesApi from '@/api/diaries.js'

describeIntegration('diaries store: полный цикл', () => {
  it('создание ежедневника, записей, выполнение и прогресс active/done', async () => {
    const u = await registerVerified()
    u.session.use()
    const store = useDiariesStore()

    await store.fetchDiaries()
    expect(store.diaries.length).toBe(0)

    const d = await store.createDiary(uniq('План '))
    expect(d.id).toBeGreaterThan(0)
    expect(store.diaries.some((x) => x.id === d.id)).toBe(true)

    store.select(d.id)
    // По умолчанию вид — неделя; курсор сегодня. Создаём запись на сегодня.
    const today = dayKey(new Date())
    const e1 = await store.createEntry({ entry_date: today, title: 'Первая задача' })
    expect(e1.id).toBeGreaterThan(0)
    await store.createEntry({ entry_date: today, title: 'Вторая задача', start_min: 600, end_min: 660 })

    await store.fetchEntries()
    expect(store.entries.length).toBe(2)

    // Отметить выполнение → уходит из активных, прогресс сдвигается.
    await store.toggleDone(e1.id, true)
    const dd = store.diaries.find((x) => x.id === d.id)
    expect(dd.done_count).toBeGreaterThanOrEqual(1)
    // После выполнения активных записей на сегодня осталась одна.
    await store.fetchEntries()
    expect(store.entries.some((x) => x.id === e1.id)).toBe(false)
  })

  it('перенос записи на другой день и ручной порядок дня', async () => {
    const u = await registerVerified()
    u.session.use()
    const store = useDiariesStore()
    const d = await store.createDiary(uniq('Порядок '))
    store.select(d.id)

    const today = new Date()
    const key = dayKey(today)
    const a = await store.createEntry({ entry_date: key, title: 'A' })
    const b = await store.createEntry({ entry_date: key, title: 'B' })
    const c = await store.createEntry({ entry_date: key, title: 'C' })
    await store.fetchEntries()

    // Реордер дня: C, A, B.
    await store.reorderDay(key, [c.id, a.id, b.id])
    await store.fetchEntries()
    const order = store.entries.filter((e) => e.entry_date === key).map((e) => e.id)
    expect(order).toEqual([c.id, a.id, b.id])

    // Перенос B на завтра — исчезает из выборки текущего дня (вид «день»).
    store.setView('day')
    await store.fetchEntries()
    const tomorrow = dayKey(new Date(today.getTime() + 86400000))
    await store.moveEntry(b.id, { entryDate: tomorrow })
    await store.fetchEntries()
    expect(store.entries.some((e) => e.id === b.id)).toBe(false)
  })

  it('адресный шаринг с can_check: адресат видит ежедневник и может отмечать', async () => {
    const owner = await registerVerified('owner')
    const guest = await registerVerified('guest')

    owner.session.use()
    const ownerStore = useDiariesStore()
    const d = await ownerStore.createDiary(uniq('Общий '))
    ownerStore.select(d.id)
    const key = dayKey(new Date())
    const entry = await ownerStore.createEntry({ entry_date: key, title: 'Поручение' })

    // Владелец делится с правом отметки.
    await diariesApi.addMember(d.id, guest.auth.userId, true)

    // Адресат видит ежедневник во вкладке «Поделились» с can_check.
    guest.session.use()
    const guestStore = useDiariesStore()
    guestStore.setTab('shared')
    await guestStore.fetchDiaries()
    const shared = guestStore.diaries.find((x) => x.id === d.id)
    expect(shared).toBeTruthy()
    expect(shared.shared).toBe(true)
    expect(shared.can_check).toBe(true)

    guestStore.select(d.id)
    expect(guestStore.readonly).toBe(true)   // чужой ежедневник — структура read-only
    expect(guestStore.canToggle).toBe(true)  // но отмечать выполнение можно

    // Адресат отмечает запись выполненной — сервер принимает.
    await guestStore.toggleDone(entry.id, true)
    await guestStore.fetchEntries()
    // Запись ушла из активных у адресата.
    expect(guestStore.entries.some((e) => e.id === entry.id)).toBe(false)
  })

  it('шаринг без can_check: адресат видит, но отмечать не может', async () => {
    const owner = await registerVerified('owner2')
    const guest = await registerVerified('guest2')
    owner.session.use()
    const ownerStore = useDiariesStore()
    const d = await ownerStore.createDiary(uniq('ТолькоЧтение '))
    await diariesApi.addMember(d.id, guest.auth.userId, false)

    guest.session.use()
    const guestStore = useDiariesStore()
    guestStore.setTab('shared')
    await guestStore.fetchDiaries()
    const shared = guestStore.diaries.find((x) => x.id === d.id)
    expect(shared.can_check).toBe(false)
    guestStore.select(d.id)
    expect(guestStore.canToggle).toBe(false)
  })
})
