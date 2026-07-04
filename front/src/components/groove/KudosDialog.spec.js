import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'

vi.mock('@/api/users.js', () => ({
  getDirectory: vi.fn(() => Promise.resolve([
    { id: 1, fio: 'Иван Петров' },
    { id: 2, fio: 'Анна Смирнова' },
  ])),
}))

import KudosDialog from './KudosDialog.vue'
import { useGrooveStore } from '@/stores/groove.js'

async function open() {
  const pinia = createTestingPinia({ createSpy: vi.fn })
  const w = mount(KudosDialog, {
    props: { modelValue: false },
    global: { plugins: [pinia] },
  })
  await w.setProps({ modelValue: true }) // watch → грузит справочник
  await flushPromises()
  return { w, store: useGrooveStore(pinia) }
}

const sendBtn = (w) => w.find('button.kudos-send')

describe('KudosDialog', () => {
  beforeEach(() => vi.clearAllMocks())

  it('загружает коллег в список при открытии', async () => {
    const { w } = await open()
    expect(w.findAll('button.kudos-user')).toHaveLength(2)
  })

  it('кнопка отправки заблокирована без выбора получателя/категории/текста', async () => {
    const { w } = await open()
    expect(sendBtn(w).attributes('disabled')).toBeDefined()
  })

  it('остаётся заблокированной без категории и текста, даже если выбран получатель', async () => {
    const { w } = await open()
    await w.findAll('button.kudos-user')[0].trigger('click')
    expect(sendBtn(w).attributes('disabled')).toBeDefined()
  })

  it('разблокируется и передаёт category+text в store.sendKudos', async () => {
    const { w, store } = await open()
    await w.findAll('button.kudos-user')[0].trigger('click')   // получатель id=1
    await w.findAll('button.kudos-cat')[0].trigger('click')     // первая категория (helped)
    const textarea = w.find('textarea.kudos-text')
    await textarea.setValue('Разобрал блокер по релизу')
    expect(sendBtn(w).attributes('disabled')).toBeUndefined()

    await sendBtn(w).trigger('click')
    expect(store.sendKudos).toHaveBeenCalledWith(1, 'helped', 'Разобрал блокер по релизу')
  })
})
