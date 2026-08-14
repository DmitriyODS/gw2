import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import RegistryExportDialog from './RegistryExportDialog.vue'

const fields = [
  { id: 10, label: 'Инвентарный номер', type: 'text' },
  { id: 11, label: 'Кабинет', type: 'text' },
]

const records = Array.from({ length: 4 }, (_, i) => ({
  id: i + 1,
  data: { 10: `INV-${i + 1}`, 11: `К-${i + 1}` },
}))

// Диалог один и тот же у раздела и у страницы ссылки — различается лишь ручка
// выгрузки, поэтому она приходит пропом и здесь подменяется шпионом.
function factory(props = {}) {
  const request = vi.fn(async () => ({ ok: true, blob: async () => new Blob(['x']) }))
  const w = mount(RegistryExportDialog, {
    props: { modelValue: false, fields, records, request, ...props },
    global: { stubs: { Checkbox: true } },
  })
  return { w, request }
}

async function open(props = {}) {
  const { w, request } = factory(props)
  await w.setProps({ modelValue: true })
  await flushPromises()
  return { w, request }
}

const previewRows = (w) => w.findAll('.rx-sheet tbody tr').map((r) => r.findAll('td')[1].text())

describe('RegistryExportDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    global.URL.createObjectURL = vi.fn(() => 'blob:x')
    global.URL.revokeObjectURL = vi.fn()
  })

  it('без выбора: область не предлагается, в файл идут все записи по фильтру', async () => {
    const { w, request } = await open()
    expect(w.find('.app-tabs').exists()).toBe(false)
    expect(previewRows(w)).toEqual(['INV-1', 'INV-2', 'INV-3', 'INV-4'])

    await w.findComponent({ name: 'AppDialog' }).vm.$emit('confirm')
    await flushPromises()
    expect(request.mock.calls[0][0].selection).toEqual({ all: true })
  })

  it('отмеченные галочками: и предпросмотр, и запрос идут перечнем id', async () => {
    const { w, request } = await open({ selectedIds: [2, 3], selectionCount: 2 })
    expect(w.find('.app-tabs').exists()).toBe(true)
    expect(w.text()).toContain('Выбранные (2)')
    expect(previewRows(w)).toEqual(['INV-2', 'INV-3'])

    await w.findComponent({ name: 'AppDialog' }).vm.$emit('confirm')
    await flushPromises()
    expect(request.mock.calls[0][0].selection).toEqual({ ids: [2, 3] })
  })

  /* Режим «выбрано всё по фильтру» не тянет id на клиент — он описывается
     набором с исключениями. Пока область выбиралась по selectedIds.length,
     снятые галочки молча уезжали в файл. */
  it('режим «выбрано всё»: снятые галочки не попадают ни в предпросмотр, ни в запрос', async () => {
    const { w, request } = await open({
      selectedIds: [],
      selection: { all: true, exclude: [1, 4] },
      selectionCount: 2,
    })
    expect(w.text()).toContain('Выбранные (2)')
    expect(previewRows(w)).toEqual(['INV-2', 'INV-3'])

    await w.findComponent({ name: 'AppDialog' }).vm.$emit('confirm')
    await flushPromises()
    expect(request.mock.calls[0][0].selection).toEqual({ all: true, exclude: [1, 4] })
  })

  it('переключение на «Все записи» возвращает полную выборку', async () => {
    const { w, request } = await open({ selectedIds: [2], selectionCount: 1 })
    await w.findAll('.app-tab')[0].trigger('click')
    await flushPromises()
    expect(previewRows(w)).toHaveLength(4)

    await w.findComponent({ name: 'AppDialog' }).vm.$emit('confirm')
    await flushPromises()
    expect(request.mock.calls[0][0].selection).toEqual({ all: true })
  })

  it('вкладка «все» называет фильтр, когда он задан', async () => {
    const { w } = await open({ selectedIds: [1], selectionCount: 1, filter: { search: 'стол' } })
    expect(w.findAll('.app-tab')[0].text()).toContain('Все по фильтру')
  })
})
