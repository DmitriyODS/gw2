import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useListPrefsStore } from '@/stores/listPrefs.js'
import EntityList from './EntityList.vue'

// Настройки списка живут на сервере — в тестах запрос не нужен: стор
// fail-open остаётся на локальном кэше.
vi.mock('@/api/users.js', () => ({
  getListPrefs: () => Promise.reject(new Error('offline')),
  saveListPrefs: () => Promise.resolve({}),
}))

const items = [
  { id: 1, name: 'Личное' },
  { id: 2, name: 'Работа' },
  { id: 3, name: 'Учёба' },
]

function setup(props = {}) {
  setActivePinia(createPinia())
  const prefs = useListPrefsStore()
  const wrapper = mount(EntityList, {
    props: { section: 'diaries', title: 'Ежедневники', items, ...props },
    global: { stubs: { teleport: true } },
  })
  return { prefs, wrapper }
}

const titles = (wrapper) => wrapper.findAll('.row-title').map((n) => n.text())

/* Минимальный dataTransfer: jsdom своего не даёт, а компоненту нужны setData,
   types и dropEffect. */
function dataTransfer() {
  const data = new Map()
  return {
    effectAllowed: '',
    dropEffect: '',
    get types() { return [...data.keys()] },
    setData: (type, value) => data.set(type, value),
    getData: (type) => data.get(type),
  }
}

describe('EntityList', () => {
  beforeEach(() => localStorage.clear())

  it('показывает записи в порядке сервера, пока порядок не задан', () => {
    const { wrapper } = setup()
    expect(titles(wrapper)).toEqual(['Личное', 'Работа', 'Учёба'])
  })

  it('закреплённые уходят в свою группу наверх', async () => {
    const { prefs, wrapper } = setup()
    prefs.togglePin('diaries', 3)
    await wrapper.vm.$nextTick()
    expect(titles(wrapper)).toEqual(['Учёба', 'Личное', 'Работа'])
    expect(wrapper.text()).toContain('Закреплённые')
  })

  it('папка показывается отдельной группой и собирает свои записи', async () => {
    const { prefs, wrapper } = setup()
    const id = prefs.addFolder('diaries', 'Дом')
    prefs.setFolder('diaries', 2, id)
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Дом')
    // Папка идёт перед общим списком, поэтому «Работа» оказывается первой.
    expect(titles(wrapper)).toEqual(['Работа', 'Личное', 'Учёба'])
  })

  it('ручной порядок переживает перерисовку списка', async () => {
    const { prefs, wrapper } = setup()
    prefs.setOrder('diaries', ['3', '1', '2'])
    await wrapper.vm.$nextTick()
    expect(titles(wrapper)).toEqual(['Учёба', 'Личное', 'Работа'])
  })

  it('фильтр появляется только у длинного списка', async () => {
    const { wrapper } = setup()
    expect(wrapper.find('.el-filter').exists()).toBe(false)
    const many = Array.from({ length: 9 }, (_, i) => ({ id: i + 1, name: `Список ${i + 1}` }))
    await wrapper.setProps({ items: many })
    expect(wrapper.find('.el-filter').exists()).toBe(true)
  })

  it('свёрнутая группа прячет свои записи', async () => {
    const { prefs, wrapper } = setup()
    prefs.togglePin('diaries', 1)
    await wrapper.vm.$nextTick()
    expect(titles(wrapper)).toContain('Личное')

    await wrapper.find('.el-gtoggle').trigger('click')
    await wrapper.vm.$nextTick()
    expect(titles(wrapper)).not.toContain('Личное')
  })

  it('в пустой области показывает только заглушку — папок там нет', async () => {
    const { prefs, wrapper } = setup()
    prefs.addFolder('diaries', 'Дом')
    await wrapper.setProps({ items: [] })
    expect(wrapper.text()).not.toContain('Дом')
  })

  it('папку удаляет только подтверждение', async () => {
    const { prefs, wrapper } = setup()
    prefs.addFolder('diaries', 'Дом')
    await wrapper.vm.$nextTick()

    // Меню папки → «Удалить папку»: сама папка при этом остаётся на месте,
    // пока человек не подтвердил (ConfirmDialog в тестах — .confirm-stub).
    await wrapper.find('.el-ghead .btn').trigger('click')
    const remove = wrapper.findAll('.ctxm-item').find((b) => b.text().includes('Удалить папку'))
    await remove.trigger('click')
    expect(prefs.section('diaries').folders).toHaveLength(1)
    expect(wrapper.find('.confirm-stub').exists()).toBe(true)
  })

  it('перетаскивание пункта меняет порядок', async () => {
    const { prefs, wrapper } = setup()
    const rows = wrapper.findAll('.el-item')
    const transfer = dataTransfer()
    await rows[2].trigger('dragstart', { dataTransfer: transfer })
    await rows[0].trigger('dragover', { dataTransfer: transfer, clientY: 0 })
    await rows[0].trigger('drop', { dataTransfer: transfer })
    expect(prefs.section('diaries').order).toEqual(['1', '3', '2'])
    expect(titles(wrapper)).toEqual(['Личное', 'Учёба', 'Работа'])
  })

  it('дроп срабатывает и без последнего dragover — строка состоит из вложенных элементов', async () => {
    const { prefs, wrapper } = setup()
    const rows = wrapper.findAll('.el-item')
    const transfer = dataTransfer()
    await rows[0].trigger('dragstart', { dataTransfer: transfer })
    // dragover был по соседу, а отпустили на третьем пункте — считается он.
    await rows[1].trigger('dragover', { dataTransfer: transfer, clientY: 0 })
    await rows[2].trigger('drop', { dataTransfer: transfer })
    expect(prefs.section('diaries').order).toEqual(['2', '3', '1'])
  })

  it('перетаскивание в папку переносит пункт в неё', async () => {
    const { prefs, wrapper } = setup()
    const folder = prefs.addFolder('diaries', 'Дом')
    await wrapper.vm.$nextTick()
    const transfer = dataTransfer()
    await wrapper.findAll('.el-item')[0].trigger('dragstart', { dataTransfer: transfer })
    await wrapper.find('.el-ghead').trigger('dragover', { dataTransfer: transfer })
    await wrapper.find('.el-ghead').trigger('drop', { dataTransfer: transfer })
    expect(prefs.section('diaries').folderOf).toEqual({ 1: folder })
  })

  it('выбор записи уходит наверх идентификатором', async () => {
    const { wrapper } = setup()
    await wrapper.findAll('.row')[1].trigger('click')
    expect(wrapper.emitted('select')[0]).toEqual([2])
  })
})
