import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import { setActivePinia } from 'pinia'
import TopicManageDialog from './TopicManageDialog.vue'
import { useAuthStore } from '@/stores/auth.js'

// claims приватны в auth-сторе (см. usePermission.spec.js) — роль выставляем
// публичным applySession, а не initialState.
function factory({ roleLevel = 1, topics = [{ id: 1, name: 'Новости', color: 'blue', icon: 'campaign' }] } = {}) {
  const pinia = createTestingPinia({
    createSpy: vi.fn,
    stubActions: false,
    initialState: {
      portal: { topics, loadingTopics: false },
    },
  })
  setActivePinia(pinia)
  useAuthStore().applySession({ access_token: 't', role_level: roleLevel })
  return mount(TopicManageDialog, {
    props: { modelValue: true },
    global: { plugins: [pinia], stubs: { teleport: true } },
  })
}

describe('TopicManageDialog', () => {
  it('сотрудник (не администратор) видит список разделов read-only: без создания, редактирования и удаления', () => {
    const w = factory({ roleLevel: 1 })
    expect(w.find('.topic-row').exists()).toBe(true)
    expect(w.find('.topic-name').text()).toBe('Новости')
    expect(w.find('button.topic-row').exists()).toBe(false) // строка не кликабельна
    expect(w.find('.topic-add-btn').exists()).toBe(false)
    expect(w.find('.topic-icon-btn').exists()).toBe(false)
  })

  it('менеджер (роль 2) тоже read-only — нужен именно администратор', () => {
    const w = factory({ roleLevel: 2 })
    expect(w.find('.topic-add-btn').exists()).toBe(false)
    expect(w.find('button.topic-row').exists()).toBe(false)
  })

  it('администратор: кликабельные строки, удаление и кнопка «Новый раздел»; форма открывается по кнопке', async () => {
    const w = factory({ roleLevel: 3 })
    expect(w.find('button.topic-row').exists()).toBe(true)
    expect(w.findAll('.topic-icon-btn').length).toBeGreaterThan(0)
    // Список открывается без формы — форма отдельный шаг.
    expect(w.find('.topic-form').exists()).toBe(false)

    await w.find('.topic-add-btn').trigger('click')
    expect(w.find('.topic-form').exists()).toBe(true)
    expect(w.find('.topic-row').exists()).toBe(false)

    // «Назад» возвращает к списку.
    await w.find('.topic-btn-text').trigger('click')
    expect(w.find('.topic-form').exists()).toBe(false)
    expect(w.find('.topic-row').exists()).toBe(true)
  })

  it('тап по строке у администратора открывает форму редактирования с заполненными полями', async () => {
    const w = factory({ roleLevel: 3 })
    await w.find('button.topic-row').trigger('click')
    expect(w.find('.topic-form').exists()).toBe(true)
    expect(w.find('.topic-input').element.value).toBe('Новости')
  })

  it('пустой список разделов показывает EmptyState вместо строк', () => {
    const w = factory({ roleLevel: 3, topics: [] })
    expect(w.text()).toContain('Разделов пока нет')
    expect(w.find('.topic-row').exists()).toBe(false)
  })
})
