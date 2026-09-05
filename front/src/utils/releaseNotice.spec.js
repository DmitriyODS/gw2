import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

/* Выпуск помнит устройство (localStorage), а сведения о нём кэширует модуль
   версии — поэтому «новый запуск приложения» изображаем vi.resetModules() с
   повторным импортом, как в тестах центра уведомлений. */
const changelog = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/api/changelog.js', () => ({ changelogApi: changelog }))

async function run() {
  const { announceRelease } = await import('@/utils/releaseNotice.js')
  await announceRelease()
  const { useDesktopNotifications } = await import('@/composables/useDesktopNotifications.js')
  const { useNotificationsStore } = await import('@/stores/notifications.js')
  return { center: useDesktopNotifications(), toasts: useNotificationsStore().toasts }
}

const release = (version, build) => ({ version, build, title: 'Расписания и формы' })

describe('уведомление о новом выпуске', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.resetModules()
    vi.clearAllMocks()
    setActivePinia(createPinia())
  })

  it('на первом запуске молчит, а после обновления зовёт в «О приложении»', async () => {
    changelog.get.mockResolvedValue(release('7.4.0', '2609051'))
    const first = await run()
    expect(first.center.count.value).toBe(0)
    expect(first.toasts).toHaveLength(0)

    // Выкатили новую версию — следующий запуск о ней рассказывает.
    vi.resetModules()
    setActivePinia(createPinia())
    changelog.get.mockResolvedValue(release('7.5.0', '2609061'))
    const after = await run()
    expect(after.center.items.value[0]).toMatchObject({
      title: 'Обновление 7.5.0',
      text: 'Расписания и формы',
      path: '/settings?section=about',
    })
    expect(after.toasts[0]).toMatchObject({ summary: 'Обновление 7.5.0', path: '/settings?section=about' })
  })

  it('на том же выпуске повторно не напоминает', async () => {
    changelog.get.mockResolvedValue(release('7.4.0', '2609051'))
    await run()

    vi.resetModules()
    setActivePinia(createPinia())
    const again = await run()
    expect(again.center.count.value).toBe(0)
    expect(again.toasts).toHaveLength(0)
  })

  it('пересобранная сборка той же версии тоже считается обновлением', async () => {
    changelog.get.mockResolvedValue(release('7.4.0', '2609051'))
    await run()

    vi.resetModules()
    setActivePinia(createPinia())
    changelog.get.mockResolvedValue(release('7.4.0', '2609052'))
    const after = await run()
    expect(after.center.count.value).toBe(1)
  })

  it('без ответа сервера ничего не показывает и отметку не портит', async () => {
    changelog.get.mockRejectedValue(new Error('нет сети'))
    const quiet = await run()
    expect(quiet.center.count.value).toBe(0)
    expect(localStorage.getItem('gw_seen_release')).toBe(null)
  })
})
