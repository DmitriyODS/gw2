import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/utils/systemNotify.js', () => ({ playNotifySound: vi.fn() }))
const session = { loggingOut: false, token: 't' }
vi.mock('./auth.js', () => ({ useAuthStore: () => session }))

import { notifyPrefs, setSourceEnabled } from '@/utils/notifySettings.js'
import { useNotificationsStore } from './notifications.js'

describe('всплывающие уведомления', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.assign(session, { loggingOut: false, token: 't' })
    Object.assign(notifyPrefs, { corner: 'top-right', life: 5000, onLockScreen: false, sources: {} })
  })

  it('новое встаёт первым в стопке', () => {
    const notif = useNotificationsStore()
    notif.success('Сохранено')
    notif.error('Не сохранилось')

    expect(notif.toasts.map((t) => t.detail)).toEqual(['Не сохранилось', 'Сохранено'])
    expect(notif.toasts[0].severity).toBe('error')
  })

  it('больше пяти карточек не копится — давние вытесняются', () => {
    const notif = useNotificationsStore()
    for (let i = 1; i <= 7; i++) notif.success(`Событие ${i}`)

    expect(notif.toasts).toHaveLength(5)
    expect(notif.toasts.at(-1).detail).toBe('Событие 3')
  })

  it('карточка убирается по своему id', () => {
    const notif = useNotificationsStore()
    notif.success('Первое')
    notif.success('Второе')
    notif.dismiss(notif.toasts[0].id)

    expect(notif.toasts.map((t) => t.detail)).toEqual(['Первое'])
  })

  // Хвостовые запросы закрытых экранов отваливаются по 401 — это ожидаемо.
  it('ошибки после выхода не показываются', () => {
    const notif = useNotificationsStore()
    session.loggingOut = true
    notif.error('401 на хвостовом запросе')
    expect(notif.toasts).toHaveLength(0)

    session.loggingOut = false
    notif.warn('А предупреждение доходит и без сессии')
    expect(notif.toasts).toHaveLength(1)
  })
})

describe('настройки уведомлений', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    Object.assign(notifyPrefs, { corner: 'top-right', life: 5000, onLockScreen: false, sources: {} })
  })

  it('выключенный раздел не всплывает', () => {
    const notif = useNotificationsStore()
    setSourceEnabled('messenger', false)

    notif.notify({ severity: 'info', detail: 'Новое сообщение', source: 'messenger' })
    expect(notif.toasts).toHaveLength(0)

    // Ответ на собственное действие источника не имеет — приходит всегда.
    notif.notify({ severity: 'info', detail: 'Сохранено' })
    expect(notif.toasts).toHaveLength(1)
  })

  it('срок жизни берётся из настроек, ошибке дают дольше', () => {
    const notif = useNotificationsStore()
    notifyPrefs.life = 3000

    notif.notify({ severity: 'info', detail: 'Обычное' })
    notif.error('Сломалось')

    expect(notif.toasts[1].life).toBe(3000)
    expect(notif.toasts[0].life).toBe(5000)
  })
})
