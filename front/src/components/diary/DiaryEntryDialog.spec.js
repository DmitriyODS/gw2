import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import DiaryEntryDialog from './DiaryEntryDialog.vue'
import { useDiariesStore } from '@/stores/diaries.js'

const byText = (w, t) => w.findAll('button').filter((b) => b.text().includes(t))

function factory(props) {
  const pinia = createTestingPinia({ createSpy: vi.fn })
  const w = mount(DiaryEntryDialog, {
    props: { modelValue: true, ...props },
    global: { plugins: [pinia] },
  })
  return { w, store: useDiariesStore(pinia) }
}

const entry = { id: 5, title: 'Позвонить клиенту', done: false, entry_date: '2026-07-08', description: '' }

describe('DiaryEntryDialog — ролевые гейты кнопок', () => {
  beforeEach(() => vi.clearAllMocks())

  it('read-only без права отметки: только «Закрыть», без правки/удаления/отметки', () => {
    const { w } = factory({ entry, readonly: true, canToggle: false })
    expect(byText(w, 'Изменить')).toHaveLength(0)
    expect(byText(w, 'Удалить')).toHaveLength(0)
    expect(byText(w, 'Выполнено')).toHaveLength(0)
    expect(byText(w, 'Закрыть')).toHaveLength(1)
  })

  it('read-only + can_check: показывает «Выполнено», клик зовёт store.toggleDone(id, true)', async () => {
    const { w, store } = factory({ entry, readonly: true, canToggle: true })
    const btn = byText(w, 'Выполнено')
    expect(btn).toHaveLength(1)
    await btn[0].trigger('click')
    expect(store.toggleDone).toHaveBeenCalledWith(5, true)
  })

  it('выполненная запись у владельца: кнопка «В активные» зовёт toggleDone(id, false)', async () => {
    const { w, store } = factory({ entry: { ...entry, done: true }, readonly: false, canToggle: false })
    const btn = byText(w, 'В активные')
    expect(btn).toHaveLength(1)
    await btn[0].trigger('click')
    expect(store.toggleDone).toHaveBeenCalledWith(5, false)
  })

  it('владелец видит «Изменить» и «Удалить»', () => {
    const { w } = factory({ entry, readonly: false, canToggle: false })
    expect(byText(w, 'Изменить')).toHaveLength(1)
    // «Удалить» — иконка-кнопка (title), ищем по атрибуту.
    expect(w.find('button.btn-icon.danger').exists()).toBe(true)
  })

  it('ссылки в описании рендерятся через LinkifiedText', () => {
    const { w } = factory({ entry: { ...entry, description: 'дока http://a.ru тут' }, readonly: true, canToggle: false })
    const a = w.find('a.linkified-a')
    expect(a.exists()).toBe(true)
    expect(a.attributes('href')).toBe('http://a.ru')
  })
})
