import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import { setActivePinia } from 'pinia'
import RegistryQrPrintDialog from './RegistryQrPrintDialog.vue'

// Отрисовка кода не проверяется — она про картинку, а не про то, ЧТО печатаем.
vi.mock('qrcode', () => ({ default: { toDataURL: vi.fn(async () => 'data:image/png;base64,x') } }))

const registry = {
  id: 1,
  name: 'Склад',
  fields: [{ id: 10, label: 'Инвентарный номер', type: 'text', config: { qr: true } }],
}

// Реестр на две страницы: выбранная запись может лежать и не на первой.
const ALL = Array.from({ length: 5 }, (_, i) => ({ id: i + 1, data: { 10: `INV-${i + 1}` } }))

// Диалог собирает записи на ОТКРЫТИИ, поэтому монтируем закрытым и открываем —
// как это и происходит в разделе.
async function factory(props = {}) {
  const pinia = createTestingPinia({ createSpy: vi.fn })
  setActivePinia(pinia)
  const fetchPage = vi.fn(async ({ page = 1, per_page: per = 200 }) => ({
    items: ALL.slice((page - 1) * per, page * per),
    total: ALL.length,
  }))
  const w = mount(RegistryQrPrintDialog, {
    props: { modelValue: false, registry, fetchPage, ...props },
    global: { plugins: [pinia], stubs: { Select: true, teleport: true } },
  })
  await w.setProps({ modelValue: true })
  await flushPromises()
  return { w, fetchPage }
}

const values = (w) => w.findAll('.qp-item-value').map((n) => n.text())

describe('RegistryQrPrintDialog', () => {
  beforeEach(() => vi.clearAllMocks())

  it('без выбора печатает все записи и не показывает переключатель области', async () => {
    const { w } = await factory()
    expect(w.find('.qp-scope').exists()).toBe(false)
    expect(values(w)).toEqual(['INV-1', 'INV-2', 'INV-3', 'INV-4', 'INV-5'])
  })

  /* Раздел отдаёт МАССИВ id (как и выгрузке). Пока проп ждал Set, у массива не
     было ни .size, ни .has — область молча становилась «все записи». */
  it('отмеченные галочками записи: печатаются только они', async () => {
    const { w } = await factory({ selectedIds: [2, 4], selectionCount: 2 })
    expect(w.find('.qp-scope').exists()).toBe(true)
    expect(w.text()).toContain('Только выбранные записи (2)')
    expect(values(w)).toEqual(['INV-2', 'INV-4'])
  })

  it('режим «выбрано всё по фильтру» уважает снятые галочки', async () => {
    const { w } = await factory({
      selectedIds: [],
      selection: { all: true, exclude: [1, 5] },
      selectionCount: 3,
    })
    await flushPromises()
    expect(values(w)).toEqual(['INV-2', 'INV-3', 'INV-4'])
  })

  it('переключение на «Все записи» возвращает полный список', async () => {
    const { w } = await factory({ selectedIds: [2], selectionCount: 1 })
    expect(values(w)).toEqual(['INV-2'])

    await w.findAll('.qp-radio input')[1].setValue()
    await flushPromises()
    expect(values(w)).toHaveLength(5)
  })

  it('количество копий задаётся по позиции и считается в итог', async () => {
    const { w } = await factory({ selectedIds: [1, 2], selectionCount: 2 })

    await w.findAll('.qp-count-input')[0].setValue('3')
    expect(w.find('.qp-head .qp-hint').text()).toContain('4 код(ов)')

    // Ноль — законный способ исключить позицию, не снимая галочку в таблице.
    await w.findAll('.qp-count-input')[1].setValue('0')
    expect(w.find('.qp-head .qp-hint').text()).toContain('3 код(ов)')
  })
})
