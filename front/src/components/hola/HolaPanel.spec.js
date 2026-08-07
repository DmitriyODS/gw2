import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { routerKey, routeLocationKey } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useDesktopStore } from '@/stores/desktop.js'
import { shellActive } from '@/desktop/layout.js'
import HolaPanel from './HolaPanel.vue'

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
vi.mock('@/api/ai.js', () => ({
  // Личный ключ ассистента не подключён — вкладка «Чат» должна запереться.
  getMyAiSettings: vi.fn(() => Promise.resolve({ has_key: false, enabled: true })),
}))
vi.mock('@/api/assistant.js', () => ({
  sendAssistantMessage: vi.fn(() => Promise.resolve({ id: 1, text: 'ответ' })),
  getAssistantHistory: vi.fn(() => Promise.resolve([])),
  sendAssistantFeedback: vi.fn(() => Promise.resolve({})),
}))

import * as tasksApi from '@/api/tasks.js'
import * as boardsApi from '@/api/boards.js'
import * as remindersApi from '@/api/reminders.js'
import * as messengerApi from '@/api/messenger.js'
import * as aiApi from '@/api/ai.js'
import { useMessengerStore } from '@/stores/messenger.js'

/* Роутер панели — заглушка: на рабочем столе переходы уходят в оконный
   менеджер, на мобильном каркасе — в router.push, и оба пути проверяем. */
const router = {
  push: vi.fn(),
  replace: vi.fn(),
  resolve: (p) => ({ path: String(p).split('?')[0], fullPath: String(p) }),
}

function setup({ shell = true } = {}) {
  setActivePinia(createPinia())
  useAuthStore().applySession({ access_token: 't', role_level: 1, company_id: 1 })
  const desktop = useDesktopStore()
  desktop.setScreen({ x: 0, y: 0, w: 1440, h: 900 })
  desktop.setArea({ x: 0, y: 0, w: 1440, h: 808 })
  shellActive.value = shell
  const wrapper = mount(HolaPanel, {
    attachTo: document.body,
    global: {
      provide: { [routerKey]: router, [routeLocationKey]: { path: '/', query: {} } },
      stubs: { MarkdownView: true },
    },
  })
  return { desktop, wrapper }
}

async function type(wrapper, text) {
  await wrapper.find('.hola-input').setValue(text)
  vi.advanceTimersByTime(300)
  await flushPromises()
}

const tabButton = (wrapper, label) =>
  wrapper.findAll('.hola-tab').find((n) => n.text().includes(label))

describe('Hola: поиск, команды и чат', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
    vi.useFakeTimers()
    shellActive.value = false
  })

  it('ищет по разделам, задачам, заметкам, ежедневникам и реестрам', async () => {
    const { wrapper } = setup()
    await type(wrapper, 'отчёт')

    const titles = wrapper.findAll('.hr-item-title').map((n) => n.text())
    expect(titles).toContain('Отчёт за июль')
    expect(titles).toContain('Отчёт: черновик')
    expect(titles).toContain('Сдать отчёт')
    expect(titles).toContain('ООО Отчёт')
    expect(tasksApi.getTasks).toHaveBeenCalledTimes(1)
  })

  it('ищет по публикациям портала и сотрудникам компании', async () => {
    const { wrapper } = setup()
    await type(wrapper, 'отчёт')

    const titles = wrapper.findAll('.hr-item-title').map((n) => n.text())
    expect(titles).toContain('Отчёт компании')
    expect(titles).toContain('Отчётова Анна')
  })

  it('предлагает выдачу поисковика и открывает её новой вкладкой', async () => {
    const open = vi.spyOn(window, 'open').mockImplementation(() => null)
    const { wrapper } = setup()
    await type(wrapper, 'отчёт')

    const web = wrapper.findAll('.hr-item.web')
    // По умолчанию первым идёт Яндекс, следом остальные поисковики.
    expect(web[0].text()).toContain('Найти в Яндекс')

    await web[0].trigger('click')
    await flushPromises()
    expect(open).toHaveBeenCalledWith(
      'https://yandex.ru/search/?text=' + encodeURIComponent('отчёт'),
      '_blank',
      'noopener,noreferrer',
    )
    // Уход наружу закрывает всплывающую панель.
    expect(wrapper.emitted('navigate')).toBeTruthy()
  })

  it('введённый адрес сайта открывается напрямую и идёт первой строкой', async () => {
    const open = vi.spyOn(window, 'open').mockImplementation(() => null)
    const { wrapper } = setup()
    await type(wrapper, 'vk.com')

    const first = wrapper.findAll('.hr-item')[0]
    expect(first.text()).toContain('Открыть vk.com')

    await first.trigger('click')
    await flushPromises()
    expect(open).toHaveBeenCalledWith('https://vk.com/', '_blank', 'noopener,noreferrer')
  })

  it('выполненный поиск попадает в историю запросов', async () => {
    const { wrapper } = setup()
    await type(wrapper, 'отчёт')

    const task = wrapper.findAll('.hr-item').find((n) => n.text().includes('Отчёт за июль'))
    await task.trigger('click')
    await flushPromises()

    expect(JSON.parse(localStorage.getItem('gw_hola_history'))[0].text).toBe('отчёт')
  })

  it('историю поиска можно очистить целиком', async () => {
    const { wrapper } = setup()
    await type(wrapper, 'отчёт')
    await wrapper.findAll('.hr-item').find((n) => n.text().includes('Отчёт за июль')).trigger('click')
    await flushPromises()

    // Пустая строка возвращает вкладку к истории — там и живёт кнопка очистки.
    await wrapper.find('.hola-input').setValue('')
    await flushPromises()
    expect(wrapper.findAll('.hola-hist-row')).toHaveLength(1)

    await wrapper.find('.hola-hist-clear').trigger('click')
    await flushPromises()
    expect(wrapper.find('.hola-history').exists()).toBe(false)
    expect(JSON.parse(localStorage.getItem('gw_hola_history'))).toEqual([])
  })

  it('клик по результату открывает окно нужного раздела', async () => {
    const { desktop, wrapper } = setup()
    await type(wrapper, 'отчёт')

    const task = wrapper.findAll('.hr-item').find((n) => n.text().includes('Отчёт за июль'))
    await task.trigger('click')
    await flushPromises()

    expect(desktop.windows).toHaveLength(1)
    expect(desktop.windows[0].path).toBe('/tasks/12')
  })

  it('без рабочего стола (мобильный каркас) переход идёт сменой экрана', async () => {
    const { desktop, wrapper } = setup({ shell: false })
    await type(wrapper, 'отчёт')

    const task = wrapper.findAll('.hr-item').find((n) => n.text().includes('Отчёт за июль'))
    await task.trigger('click')
    await flushPromises()

    expect(router.push).toHaveBeenCalledWith('/tasks/12')
    expect(desktop.windows).toHaveLength(0)
  })

  it('команда «создай задачу …» открывает форму с готовым названием', async () => {
    const { desktop, wrapper } = setup()
    await type(wrapper, 'создай задачу институт что-то там')

    const cmd = wrapper.find('.hr-item.command')
    expect(cmd.text()).toContain('Создать задачу «Институт что-то там»')

    await cmd.trigger('click')
    await flushPromises()
    expect(desktop.windows[0].path).toBe('/tasks?new=1&title=' + encodeURIComponent('Институт что-то там'))
  })

  it('команда «создай доску …» создаёт доску и открывает её', async () => {
    const { desktop, wrapper } = setup()
    await type(wrapper, 'создай доску схема процесса')

    await wrapper.find('.hr-item.command').trigger('click')
    await flushPromises()
    expect(boardsApi.createBoard).toHaveBeenCalledWith('Схема процесса', null)
    expect(desktop.windows[0].path).toBe('/boards/21')
  })

  it('«напомни … завтра в 9» создаёт напоминание сразу', async () => {
    const { wrapper } = setup()
    await type(wrapper, 'напомни позвонить в банк завтра в 9:00')

    const cmd = wrapper.find('.hr-item.command')
    expect(cmd.text()).toContain('Напомнить «Позвонить в банк»')

    await cmd.trigger('click')
    await flushPromises()
    const body = remindersApi.createReminder.mock.calls[0][0]
    expect(body.title).toBe('Позвонить в банк')
    expect(new Date(body.remind_at).getHours()).toBe(9)
  })

  it('«напиши васе …» отправляет сообщение и открывает чат', async () => {
    const { desktop, wrapper } = setup()
    useMessengerStore().conversations = [
      { id: 12, other_user: { id: 3, fio: 'Васильев Пётр', login: 'vasilev.p' } },
    ]
    await type(wrapper, 'напиши васе созвон в 15:00')

    const row = wrapper.find('.hr-item.command')
    expect(row.text()).toContain('Васильев Пётр — «созвон в 15:00»')

    await row.trigger('click')
    await flushPromises()
    expect(messengerApi.sendMessage).toHaveBeenCalledWith(12, expect.objectContaining({ text: 'созвон в 15:00' }))
    expect(desktop.windows[0].path).toBe('/messenger/12')
  })

  it('короткий запрос сеть не дёргает', async () => {
    const { wrapper } = setup()
    await type(wrapper, 'о')
    expect(tasksApi.getTasks).not.toHaveBeenCalled()
  })

  it('арифметику считает вместо поиска', async () => {
    const { wrapper } = setup()
    await wrapper.find('.hola-input').setValue('1200*3')
    await flushPromises()
    // Разряды разделяет неразрывный пробел — нормализуем перед сравнением.
    expect(wrapper.find('.hola-calc-value').text().replace(/ /g, ' ')).toBe('= 3 600')
  })

  it('вкладка «Команды» показывает каталог быстрых действий', async () => {
    const { wrapper } = setup()
    await tabButton(wrapper, 'Команды').trigger('click')
    await flushPromises()

    const titles = wrapper.findAll('.hr-item-title').map((n) => n.text())
    expect(titles).toEqual(expect.arrayContaining(['Новая заметка', 'Новый чат', 'Калькулятор', 'Создать напоминание']))
  })

  it('«Калькулятор» открывает окно калькулятора', async () => {
    const { desktop, wrapper } = setup()
    await tabButton(wrapper, 'Команды').trigger('click')
    await flushPromises()

    const calc = wrapper.findAll('.hr-item').find((n) => n.text().includes('Калькулятор'))
    await calc.trigger('click')
    await flushPromises()
    expect(desktop.windows[0].path).toBe('/calculator')
  })

  // Ассистент работает на ЛИЧНОМ ключе: пока его нет, вкладка заперта — и
  // запирается заранее, по настройкам, а не после неудачной отправки.
  it('чат заблокирован, пока не подключён личный ключ', async () => {
    const { wrapper } = setup()
    await flushPromises()
    await tabButton(wrapper, 'Чат').trigger('click')
    await flushPromises()

    expect(aiApi.getMyAiSettings).toHaveBeenCalled()
    expect(wrapper.find('.hola-locked').exists()).toBe(true)
  })

  // Из запертого чата есть выход — кнопка ведёт в настройки, где живёт ключ.
  it('из запертого чата кнопка ведёт в настройки за ключом', async () => {
    const { wrapper } = setup()
    await flushPromises()
    await tabButton(wrapper, 'Чат').trigger('click')
    await flushPromises()

    await wrapper.find('.hola-locked-btn').trigger('click')
    expect(router.push).toHaveBeenCalledWith('/settings?section=ai')
  })
})
