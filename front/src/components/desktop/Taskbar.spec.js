import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useDesktopStore } from '@/stores/desktop.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { pushNotification, clearNotificationJournal } from '@/composables/useDesktopNotifications.js'
import { isNotifyMuted, unmuteNotifications } from '@/utils/systemNotify.js'
import { useAuthStore } from '@/stores/auth.js'
import Taskbar from './Taskbar.vue'

function setup() {
  setActivePinia(createPinia())
  // Закреплённые разделы фильтруются по правам — даём сессии активную компанию
  // (клеймы в сторе приватные, поэтому применяем их штатным applySession).
  useAuthStore().applySession({ access_token: 't', role_level: 1, company_id: 1 })
  const desktop = useDesktopStore()
  desktop.setScreen({ x: 0, y: 0, w: 1424, h: 904 })
  desktop.setArea({ x: 12, y: 12, w: 1400, h: 800 })
  const wrapper = mount(Taskbar, {
    global: { stubs: { Logo: true, ContextMenu: true, NotificationsPanel: true } },
  })
  return { desktop, wrapper }
}

describe('панель задач', () => {
  // Журнал уведомлений и тишина — модульное состояние (общее у бейджа и
  // панели), поэтому чистим его между проверками отдельно от localStorage.
  beforeEach(() => {
    localStorage.clear()
    clearNotificationJournal()
    unmuteNotifications()
  })

  it('показывает кнопку каждого открытого окна', async () => {
    const { desktop, wrapper } = setup()
    desktop.open('/notes')
    desktop.open('/notes', { newWindow: true })
    desktop.open('/tasks')
    await wrapper.vm.$nextTick()
    expect(wrapper.findAll('.tb-win')).toHaveLength(3)
    expect(wrapper.findAll('.tb-win-label').map((n) => n.text())).toEqual(['Заметки', 'Заметки', 'Задачи'])
  })

  it('клик по кнопке активного окна сворачивает его, повторный — возвращает', async () => {
    const { desktop, wrapper } = setup()
    const win = desktop.open('/tasks')
    await wrapper.vm.$nextTick()

    await wrapper.find('.tb-win').trigger('click')
    expect(win.minimized).toBe(true)

    await wrapper.find('.tb-win').trigger('click')
    expect(win.minimized).toBe(false)
    expect(desktop.focusedId).toBe(win.id)
  })

  it('кнопка «Пуск» открывает и закрывает меню', async () => {
    const { desktop, wrapper } = setup()
    await wrapper.find('.tb-start').trigger('click')
    expect(desktop.startOpen).toBe(true)
    await wrapper.find('.tb-start').trigger('click')
    expect(desktop.startOpen).toBe(false)
  })

  it('закреплённый раздел показывается ярлыком и открывается по клику', async () => {
    const { desktop, wrapper } = setup()
    useDesktopPrefsStore().pin('tasks')
    await wrapper.vm.$nextTick()

    const shortcut = wrapper.find('.tb-win.shortcut')
    expect(shortcut.exists()).toBe(true)

    await shortcut.trigger('click')
    expect(desktop.windows.map((w) => w.appId)).toEqual(['tasks'])
    // Открытый раздел занимает место своего ярлыка, а не добавляет вторую кнопку.
    await wrapper.vm.$nextTick()
    expect(wrapper.findAll('.tb-win')).toHaveLength(1)
    expect(wrapper.find('.tb-win.shortcut').exists()).toBe(false)
  })

  it('перетаскивание меняет порядок закреплённых разделов', async () => {
    const { wrapper } = setup()
    const prefs = useDesktopPrefsStore()
    prefs.pin('tasks')
    prefs.pin('notes')
    await wrapper.vm.$nextTick()

    const [first, second] = wrapper.findAll('.tb-win')
    await first.trigger('dragstart', { dataTransfer: { setData() {} } })
    await second.trigger('dragover')
    expect(prefs.pinned).toEqual(['notes', 'tasks'])

    await first.trigger('dragend')
    expect(wrapper.find('.tb-win.dragging').exists()).toBe(false)
  })

  it('в полноэкранном режиме панель прячется и возвращается по «подглядыванию»', async () => {
    const { desktop, wrapper } = setup()
    const win = desktop.open('/tasks')
    desktop.maximize(win.id)
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.taskbar').classes()).toContain('hidden')

    desktop.taskbarPeek = true
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.taskbar').classes()).not.toContain('hidden')
  })

  /* Бейдж считает КАРТОЧКИ центра уведомлений, а не события: три сообщения в
     одной переписке — одна карточка и единица на бейдже. */
  it('счётчик уведомлений считает карточки центра', async () => {
    const { wrapper } = setup()
    expect(wrapper.find('.tb-bell-dot').exists()).toBe(false)

    useMessengerStore().conversations = [
      { id: 1, unread_count: 3, other_user: { id: 2, fio: 'Иванов Иван' } },
    ]
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.tb-bell-dot').text()).toBe('1')

    // Сработавшее напоминание — запись журнала: она переживает перезагрузку,
    // поэтому и приходит в центр отдельным путём.
    pushNotification({ key: 'reminder-7', title: 'Позвонить в банк', text: 'Пора!', path: '/reminders' })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.tb-bell-dot').text()).toBe('2')
  })

  it('ПКМ по колокольчику отключает уведомления на срок и включает обратно', async () => {
    const { wrapper } = setup()
    const bell = wrapper.find('.tb-bell')
    await bell.trigger('contextmenu')

    // Пункт один, у него подменю сроков — выбираем «30 минут».
    wrapper.findComponent({ name: 'ContextMenu' }).vm.$emit('select', 'mute:30')
    await wrapper.vm.$nextTick()
    expect(isNotifyMuted()).toBe(true)
    expect(bell.classes()).toContain('muted')

    wrapper.findComponent({ name: 'ContextMenu' }).vm.$emit('select', 'unmute')
    await wrapper.vm.$nextTick()
    expect(isNotifyMuted()).toBe(false)
    expect(bell.classes()).not.toContain('muted')
  })
})
