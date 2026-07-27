import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useMessengerStore } from '@/stores/messenger.js'

/* Модуль держит журнал в памяти и в localStorage, поэтому «перезагрузку
   страницы» изображаем vi.resetModules() + повторным импортом: свежий модуль
   поднимает журнал из хранилища ровно так же, как это делает новая вкладка. */
const load = () => import('@/composables/useDesktopNotifications.js')

describe('центр уведомлений', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.resetModules()
    setActivePinia(createPinia())
  })

  it('журнал переживает перезагрузку страницы', async () => {
    const first = await load()
    first.pushNotification({ key: 'reminder-7', title: 'Позвонить в банк', text: 'Пора!', path: '/reminders' })
    expect(first.useDesktopNotifications().count.value).toBe(1)

    vi.resetModules()
    setActivePinia(createPinia())
    const reloaded = await load()
    const { items, count } = reloaded.useDesktopNotifications()
    expect(count.value).toBe(1)
    expect(items.value[0]).toMatchObject({ title: 'Позвонить в банк', path: '/reminders', journal: true })
  })

  it('повтор того же события поднимает прежнюю строку, а не плодит дубль', async () => {
    const m = await load()
    m.pushNotification({ key: 'reminder-7', title: 'Позвонить в банк', text: 'Пора!' })
    m.pushNotification({ key: 'reminder-7', title: 'Позвонить в банк', text: 'Пора! (отложено)' })
    const { items } = m.useDesktopNotifications()
    expect(items.value).toHaveLength(1)
    expect(items.value[0].text).toBe('Пора! (отложено)')
  })

  it('запись журнала убирается насовсем, живая строка — до следующего изменения', async () => {
    const m = await load()
    m.pushNotification({ key: 'reminder-7', title: 'Напоминание', text: 'Пора!' })
    const messenger = useMessengerStore()
    messenger.conversations = [{ id: 5, unread_count: 2, other_user: { id: 9, fio: 'Иванов Иван' } }]

    const { items, count, dismiss } = m.useDesktopNotifications()
    expect(count.value).toBe(2)

    dismiss(items.value.find((i) => i.journal))
    dismiss(items.value.find((i) => !i.journal))
    expect(count.value).toBe(0)

    // Новое сообщение в том же чате — строка возвращается (отпечаток изменился).
    messenger.conversations = [{ id: 5, unread_count: 3, other_user: { id: 9, fio: 'Иванов Иван' } }]
    expect(count.value).toBe(1)

    // А убранное напоминание не вернётся: оно уже случилось.
    vi.resetModules()
    setActivePinia(createPinia())
    const reloaded = await load()
    expect(reloaded.useDesktopNotifications().count.value).toBe(0)
  })
})
