import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '@/stores/auth.js'
import { useDesktopStore } from '@/stores/desktop.js'
import Spotlight from './Spotlight.vue'

vi.mock('@/api/tasks.js', () => ({
  getTasks: vi.fn(() => Promise.resolve({ tasks: [{ id: 12, name: 'Отчёт за июль', department_name: 'Бухгалтерия' }] })),
}))
vi.mock('@/api/notes.js', () => ({
  getNotes: vi.fn(() => Promise.resolve({ notes: [{ id: 5, title: 'Отчёт: черновик', text_content: 'план' }] })),
}))
vi.mock('@/api/diaries.js', () => ({
  searchEntries: vi.fn(() => Promise.resolve({
    items: [{ diary_id: 3, diary_name: 'Личный', entry_id: 9, title: 'Сдать отчёт', entry_date: '2026-07-27' }],
  })),
}))
vi.mock('@/api/registries.js', () => ({
  searchRecords: vi.fn(() => Promise.resolve({
    items: [{ registry_id: 2, registry_name: 'Контрагенты', record_id: 44, snippet: 'ООО Отчёт' }],
  })),
}))

vi.mock('@/api/portal.js', () => ({
  getPosts: vi.fn(() => Promise.resolve({ pinned: [], posts: [{ id: 7, title: 'Отчёт компании', body: 'текст' }] })),
}))
vi.mock('@/api/users.js', () => ({
  getDirectory: vi.fn(() => Promise.resolve([{ id: 4, fio: 'Отчётова Анна', post: 'Бухгалтер' }])),
  getDesktopPrefs: vi.fn(() => Promise.resolve({ prefs: {} })),
  saveDesktopPrefs: vi.fn(() => Promise.resolve({})),
}))
vi.mock('@/api/boards.js', () => ({
  getBoards: vi.fn(() => Promise.resolve({ boards: [] })),
  getFolders: vi.fn(() => Promise.resolve({ folders: [], shared: [] })),
  createBoard: vi.fn((title) => Promise.resolve({ id: 21, title })),
}))
vi.mock('@/api/reminders.js', () => ({
  createReminder: vi.fn((body) => Promise.resolve({ id: 33, active: true, ...body })),
}))
vi.mock('@/api/messenger.js', () => ({
  openConversation: vi.fn(() => Promise.resolve({ id: 77, other_user: { id: 4, fio: 'Отчётова Анна' } })),
  sendMessage: vi.fn(() => Promise.resolve({ id: 500, text: 'привет' })),
  listMessages: vi.fn(() => Promise.resolve([])),
  markRead: vi.fn(() => Promise.resolve({})),
}))

import * as tasksApi from '@/api/tasks.js'
import * as boardsApi from '@/api/boards.js'
import * as remindersApi from '@/api/reminders.js'
import * as messengerApi from '@/api/messenger.js'
import { useMessengerStore } from '@/stores/messenger.js'

function setup() {
  setActivePinia(createPinia())
  useAuthStore().applySession({ access_token: 't', role_level: 1, company_id: 1 })
  const desktop = useDesktopStore()
  desktop.setScreen({ x: 0, y: 0, w: 1440, h: 900 })
  desktop.setArea({ x: 0, y: 0, w: 1440, h: 808 })
  const wrapper = mount(Spotlight, { attachTo: document.body })
  return { desktop, wrapper }
}

async function type(wrapper, text) {
  await wrapper.find('.sp-input').setValue(text)
  vi.advanceTimersByTime(300)
  await flushPromises()
}

describe('глобальный поиск', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  it('ищет по разделам, задачам, заметкам, ежедневникам и реестрам', async () => {
    const { wrapper } = setup()
    await type(wrapper, 'отчёт')

    const titles = wrapper.findAll('.sp-item-title').map((n) => n.text())
    expect(titles).toContain('Отчёт за июль')
    expect(titles).toContain('Отчёт: черновик')
    expect(titles).toContain('Сдать отчёт')
    expect(titles).toContain('ООО Отчёт')
    expect(tasksApi.getTasks).toHaveBeenCalledTimes(1)
  })

  it('ищет по публикациям портала и сотрудникам компании', async () => {
    const { wrapper } = setup()
    await type(wrapper, 'отчёт')

    const titles = wrapper.findAll('.sp-item-title').map((n) => n.text())
    expect(titles).toContain('Отчёт компании')
    expect(titles).toContain('Отчётова Анна')
  })

  it('карточка сотрудника открывается в разделе «Сотрудники»', async () => {
    const { desktop, wrapper } = setup()
    await type(wrapper, 'отчёт')

    const person = wrapper.findAll('.sp-item').find((n) => n.text().includes('Отчётова Анна'))
    await person.trigger('click')
    expect(desktop.windows[0].path).toBe('/employees?user=4')
  })

  it('ищет переписки по названию чата и разделы настроек', async () => {
    const { wrapper } = setup()
    useMessengerStore().conversations = [
      { id: 8, is_group: true, title: 'Отчётность', last_message: { text: 'сдали' } },
    ]
    await type(wrapper, 'отчёт')
    expect(wrapper.findAll('.sp-item-title').map((n) => n.text())).toContain('Отчётность')

    await type(wrapper, 'тёмная тема')
    const settings = wrapper.findAll('.sp-item').find((n) => n.text().includes('Темы и оформление'))
    await settings.trigger('click')
    expect(useDesktopStore().windows.at(-1).path).toBe('/settings?section=theme')
  })

  it('команда «создай задачу …» открывает форму с готовым названием', async () => {
    const { desktop, wrapper } = setup()
    await type(wrapper, 'создай задачу институт что-то там')

    const cmd = wrapper.find('.sp-item.command')
    expect(cmd.text()).toContain('Создать задачу «Институт что-то там»')

    await cmd.trigger('click')
    expect(desktop.windows[0].path).toBe('/tasks?new=1&title=' + encodeURIComponent('Институт что-то там'))
  })

  it('команда «создай доску …» создаёт доску и открывает её', async () => {
    const { desktop, wrapper } = setup()
    await type(wrapper, 'создай доску схема процесса')

    const cmd = wrapper.find('.sp-item.command')
    expect(cmd.text()).toContain('Создать доску «Схема процесса»')

    await cmd.trigger('click')
    await flushPromises()
    expect(boardsApi.createBoard).toHaveBeenCalledWith('Схема процесса', null)
    expect(desktop.windows[0].path).toBe('/boards/21')
  })

  it('«напомни … завтра в 9» создаёт напоминание сразу', async () => {
    const { wrapper } = setup()
    await type(wrapper, 'напомни позвонить в банк завтра в 9:00')

    const cmd = wrapper.find('.sp-item.command')
    expect(cmd.text()).toContain('Напомнить «Позвонить в банк»')
    expect(cmd.text()).toContain('завтра в 09:00')

    await cmd.trigger('click')
    await flushPromises()
    const body = remindersApi.createReminder.mock.calls[0][0]
    expect(body.title).toBe('Позвонить в банк')
    expect(new Date(body.remind_at).getHours()).toBe(9)
  })

  it('напоминание без срока открывает форму с готовым названием', async () => {
    const { desktop, wrapper } = setup()
    await type(wrapper, 'создай напоминание позвонить в банк')

    const cmd = wrapper.find('.sp-item.command')
    await cmd.trigger('click')
    expect(remindersApi.createReminder).not.toHaveBeenCalled()
    expect(desktop.windows[0].path).toBe('/reminders?new=1&title=' + encodeURIComponent('Позвонить в банк'))
  })

  it('«напиши васе …» отправляет сообщение и открывает чат', async () => {
    const { desktop, wrapper } = setup()
    useMessengerStore().conversations = [
      { id: 12, other_user: { id: 3, fio: 'Васильев Пётр', login: 'vasilev.p' } },
    ]
    await type(wrapper, 'напиши васе созвон в 15:00')

    const row = wrapper.find('.sp-item.command')
    expect(row.text()).toContain('Васильев Пётр — «созвон в 15:00»')

    await row.trigger('click')
    await flushPromises()
    expect(messengerApi.sendMessage).toHaveBeenCalledWith(12, expect.objectContaining({ text: 'созвон в 15:00' }))
    expect(desktop.windows[0].path).toBe('/messenger/12')
  })

  it('«напиши васе» без текста просто открывает переписку', async () => {
    const { desktop, wrapper } = setup()
    useMessengerStore().conversations = [
      { id: 12, other_user: { id: 3, fio: 'Васильев Пётр', login: 'vasilev.p' } },
    ]
    await type(wrapper, 'напиши васе')

    await wrapper.find('.sp-item.command').trigger('click')
    await flushPromises()
    expect(messengerApi.sendMessage).not.toHaveBeenCalled()
    expect(desktop.windows[0].path).toBe('/messenger/12')
  })

  it('находит раздел по названию', async () => {
    const { wrapper } = setup()
    await type(wrapper, 'замет')
    expect(wrapper.findAll('.sp-item-title').map((n) => n.text())).toContain('Заметки')
  })

  it('клик по результату открывает окно нужного раздела', async () => {
    const { desktop, wrapper } = setup()
    await type(wrapper, 'отчёт')

    const task = wrapper.findAll('.sp-item').find((n) => n.text().includes('Отчёт за июль'))
    await task.trigger('click')

    expect(desktop.windows).toHaveLength(1)
    expect(desktop.windows[0].path).toBe('/tasks/12')
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('стрелки и Enter открывают выбранный результат', async () => {
    const { desktop, wrapper } = setup()
    await type(wrapper, 'отчёт')

    const input = wrapper.find('.sp-input')
    await input.trigger('keydown', { key: 'ArrowDown' })
    await input.trigger('keydown', { key: 'Enter' })

    expect(desktop.windows).toHaveLength(1)
  })

  it('короткий запрос сеть не дёргает', async () => {
    const { wrapper } = setup()
    await type(wrapper, 'о')
    expect(tasksApi.getTasks).not.toHaveBeenCalled()
  })

  it('арифметику считает вместо поиска', async () => {
    const { wrapper } = setup()
    await wrapper.find('.sp-input').setValue('1200*3')
    await flushPromises()
    // Разряды разделяет неразрывный пробел — нормализуем перед сравнением.
    expect(wrapper.find('.sp-calc-value').text().replace(/\u00a0/g, ' ')).toBe('= 3 600')
  })
})
