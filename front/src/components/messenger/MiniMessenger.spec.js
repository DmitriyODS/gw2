import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import MiniMessenger from './MiniMessenger.vue'
import { useAuthStore } from '@/stores/auth.js'

const mockRoute = { path: '/tasks', meta: {} }
vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal()
  return { ...actual, useRoute: () => mockRoute, useRouter: () => ({ push: vi.fn() }) }
})

vi.mock('@/api/assistant.js', () => ({
  sendAssistantMessage: vi.fn(() => Promise.resolve({ id: 1, text: 'Ответ ассистента' })),
  getAssistantHistory: vi.fn(() => Promise.resolve([])),
  sendAssistantFeedback: vi.fn(() => Promise.resolve({ status: 'ok' })),
}))

// Мессенджер и вложенные диалоги — тяжёлый функционал вне зоны этого теста
// (список диалогов/тред уже проверяются e2e/интеграционно); здесь важны
// только переключение вкладок и видимость FAB.
const STUBS = {
  teleport: true,
  MessageBubble: true,
  MessageInput: true,
  ForwardDialog: true,
  DeleteScopeDialog: true,
  AttachTaskDialog: true,
  MessageContextMenu: true,
  ProgressSpinner: true,
}

function factory({ route = {}, state = {}, companyId = 10 } = {}) {
  Object.assign(mockRoute, { path: '/tasks', meta: {} }, route)
  const pinia = createTestingPinia({
    createSpy: vi.fn,
    stubActions: false,
    initialState: {
      messenger: { conversations: [], totalUnread: 0 },
      call: { phase: 'idle', isMinimized: false },
      ...state,
    },
  })
  // claims — не обычное поле state, а производное applySession(); как и в
  // stores/portal.spec.js, надёжнее вызвать реальный экшен, а не initialState.
  useAuthStore(pinia).applySession({ access_token: 't', user_id: 1, company_id: companyId, role_level: 1 })
  return mount(MiniMessenger, { global: { plugins: [pinia], stubs: STUBS } })
}

describe('MiniMessenger (хаб Ассистент/Сообщения)', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('по умолчанию открывает вкладку «Ассистент»', async () => {
    const w = factory()
    await w.find('.mini-fab').trigger('click')
    expect(w.find('.assistant-thread').exists()).toBe(true)
    expect(w.find('.mini-list').exists()).toBe(false)
  })

  it('переключение на вкладку «Сообщения» показывает список диалогов', async () => {
    const w = factory({
      state: { messenger: { conversations: [{ id: 1, other_user: { id: 2, fio: 'Иван' } }], totalUnread: 0 } },
    })
    await w.find('.mini-fab').trigger('click')
    const tabs = w.findAll('.seg-tab')
    const messagesTab = tabs.find(t => t.text().includes('Сообщения'))
    await messagesTab.trigger('click')
    expect(w.find('.assistant-thread').exists()).toBe(false)
    expect(w.find('.mini-conv').exists()).toBe(true)
  })

  it('запоминает последнюю вкладку в localStorage', async () => {
    const w = factory()
    await w.find('.mini-fab').trigger('click')
    const tabs = w.findAll('.seg-tab')
    await tabs.find(t => t.text().includes('Сообщения')).trigger('click')
    expect(localStorage.getItem('gw_assistant_hub_tab')).toBe('messages')
  })

  it('бейдж непрочитанных отображается на вкладке «Сообщения»', async () => {
    const w = factory({ state: { messenger: { conversations: [], totalUnread: 5 } } })
    await w.find('.mini-fab').trigger('click')
    const tabs = w.findAll('.seg-tab')
    const messagesTab = tabs.find(t => t.text().includes('Сообщения'))
    expect(messagesTab.find('.seg-tab-badge').text()).toBe('5')
  })

  it('FAB виден на /tasks (не только на /messenger)', () => {
    const w = factory({ route: { path: '/tasks' } })
    expect(w.find('.mini-fab').exists()).toBe(true)
  })

  it('FAB виден на /messenger (раньше скрывался целиком)', () => {
    const w = factory({ route: { path: '/messenger' } })
    expect(w.find('.mini-fab').exists()).toBe(true)
  })

  it('FAB скрыт на fullscreen-роуте (например /tv)', () => {
    const w = factory({ route: { path: '/tv', meta: { fullscreen: true } } })
    expect(w.find('.mini-fab').exists()).toBe(false)
  })

  it('FAB скрыт во время активного полноэкранного звонка', () => {
    const w = factory({ state: { call: { phase: 'active', isMinimized: false } } })
    expect(w.find('.mini-fab').exists()).toBe(false)
  })

  it('FAB виден, если звонок активен, но свёрнут в мини-режим', () => {
    const w = factory({ state: { call: { phase: 'active', isMinimized: true } } })
    expect(w.find('.mini-fab').exists()).toBe(true)
  })
})
