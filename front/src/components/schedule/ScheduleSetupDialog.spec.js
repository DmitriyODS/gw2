import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ScheduleSetupDialog from './ScheduleSetupDialog.vue'

/* Диалог настроек сохраняет ДВЕ вещи подряд: сами настройки и набор полей.
   Первое обновляет расписание в сторе — и раньше это возвращало черновик полей
   к серверному набору прямо посреди сохранения: только что добавленное поле
   исчезало, не успев уехать. Плюс диалог обязан закрыться после успеха. */

const schedule = {
  id: 1,
  name: 'Учёба',
  cycle_weeks: 1,
  cycle_anchor: '2026-09-01',
  week_labels: [],
  timezone: 'Europe/Moscow',
  categories: [],
  fields: [{ id: 7, label: 'Аудитория', type: 'text', config: {}, col_span: 1, row_span: 1, show_in_card: true }],
}

function setup({ submit, saveFields, schedule: sc = schedule } = {}) {
  return mount(ScheduleSetupDialog, {
    props: {
      modelValue: true,
      schedule: sc,
      submit: submit || vi.fn(async () => {}),
      saveFields: saveFields || vi.fn(async () => {}),
      addCategoryFn: vi.fn(),
      updateCategoryFn: vi.fn(),
      removeCategoryFn: vi.fn(),
      removeFn: vi.fn(),
    },
    global: {
      stubs: {
        AppDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        InputText: { props: ['modelValue'], template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />' },
        Textarea: true,
        Select: true,
        DatePicker: true,
      },
    },
  })
}

describe('ScheduleSetupDialog', () => {
  it('сохраняет добавленное поле и закрывает диалог', async () => {
    const saveFields = vi.fn(async () => {})
    /* Сохранение настроек ЗАМЕНЯЕТ расписание новым объектом — так делает стор
       (applyView кладёт в список ответ сервера). Именно на этом месте watch
       возвращал черновик к серверному набору, и новое поле пропадало. */
    let wrapper
    const submit = vi.fn(async () => {
      await wrapper.setProps({ schedule: { ...schedule, name: 'Учёба (сохранено)' } })
    })
    wrapper = setup({ submit, saveFields })

    wrapper.vm.tab = 'fields'
    wrapper.vm.addField()
    await flushPromises()
    wrapper.vm.fields[wrapper.vm.fields.length - 1].label = 'Преподаватель'

    await wrapper.vm.save()
    await flushPromises()

    const sent = saveFields.mock.calls[0][0]
    expect(sent.map((f) => f.label)).toEqual(['Аудитория', 'Преподаватель'])
    expect(sent[1].id).toBe(0) // новое поле уходит без id — его заведёт сервер
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([false])
  })

  it('закрытие с несохранёнными правками сперва спрашивает', async () => {
    const wrapper = setup()

    wrapper.vm.dismiss()
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([false])

    const again = setup()
    again.vm.addField()
    await flushPromises()
    again.vm.dismiss()
    expect(again.emitted('update:modelValue')).toBeUndefined()
    expect(again.vm.confirmClose).toBe(true)
  })
})
