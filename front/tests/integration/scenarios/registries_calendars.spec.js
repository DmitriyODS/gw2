// Сценарий «реестры и календари» через api-модули против живых
// registrysvc/calendarsvc: структура (админ), запись, список, поиск —
// проверяем, что api-модуль корректно строит пути/тела и разбирает ответы.
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { newCompanyAdmin } from '../setup/factory.js'
import * as reg from '@/api/registries.js'
import * as cal from '@/api/calendars.js'

function fieldId(fields, label) {
  const f = fields.find((x) => x.label === label)
  if (!f) throw new Error('поле не найдено: ' + label)
  return String(f.id)
}

describeIntegration('registries api', () => {
  it('структура, запись, список и поиск', async () => {
    const admin = await newCompanyAdmin('regadmin')
    admin.session.use()

    const created = await reg.createRegistry(uniq('Клиенты '))
    expect(created.id).toBeGreaterThan(0)

    const put = await reg.replaceFields(created.id, [
      { label: 'Название', type: 'text', col_span: 2, show_in_table: true },
      { label: 'Код', type: 'number', show_in_table: true },
      { label: 'Статус', type: 'select', config: { options: ['новый', 'в работе'] } },
    ])
    expect(Array.isArray(put.fields)).toBe(true)
    expect(put.fields.length).toBe(3)

    // Структура читается обратно.
    const full = await reg.getRegistry(created.id)
    expect(full.fields.length).toBe(3)

    const nameId = fieldId(put.fields, 'Название')
    const codeId = fieldId(put.fields, 'Код')
    const rec = await reg.createRecord(created.id, { [nameId]: 'Альфа-Групп', [codeId]: 42 })
    expect(rec.id).toBeGreaterThan(0)

    const list = await reg.getRecords(created.id)
    expect(list.items.length).toBe(1)
    expect(list.items[0].data[nameId]).toBe('Альфа-Групп')

    // Сквозной поиск по тексту записи.
    const found = await reg.getRecords(created.id, { search: 'Альфа' })
    expect(found.items.length).toBe(1)
    const none = await reg.getRecords(created.id, { search: 'нетакого' })
    expect(none.items.length).toBe(0)
  })

  it('список реестров содержит созданный', async () => {
    const admin = await newCompanyAdmin('reglist')
    admin.session.use()
    const c = await reg.createRegistry(uniq('Справочник '))
    const all = await reg.getRegistries()
    const arr = all.registries ?? all.items ?? all
    expect(arr.some((x) => x.id === c.id)).toBe(true)
  })
})

describeIntegration('calendars api', () => {
  it('структура, событие с event_at, выборка за диапазон и поиск', async () => {
    const admin = await newCompanyAdmin('caladmin')
    admin.session.use()

    const created = await cal.createCalendar(uniq('Мероприятия '))
    expect(created.id).toBeGreaterThan(0)

    const put = await cal.replaceFields(created.id, [
      { label: 'Тема', type: 'text', show_in_table: true, show_in_card: true },
      { label: 'Зал', type: 'text', show_in_table: true },
    ])
    expect(put.fields.length).toBe(2)
    const themeId = fieldId(put.fields, 'Тема')

    // Событие с обязательным event_at (без секунд на выходе).
    const eventAt = '2026-07-05T10:00:00Z'
    const entry = await cal.createEntry(created.id, eventAt, { [themeId]: 'Планёрка' })
    expect(entry.id).toBeGreaterThan(0)
    expect(entry.event_at.startsWith('2026-07-05T10:00:00')).toBe(true)

    // Выборка за июль.
    const items = await cal.getEntries(created.id, { from: '2026-07-01T00:00:00Z', to: '2026-08-01T00:00:00Z' })
    expect(items.items.length).toBe(1)
    expect(items.items[0].data[themeId]).toBe('Планёрка')

    // Поиск по тексту.
    const found = await cal.getEntries(created.id, {
      from: '2026-07-01T00:00:00Z', to: '2026-08-01T00:00:00Z', search: 'Планёрка',
    })
    expect(found.items.length).toBe(1)
  })
})
