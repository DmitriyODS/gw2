import { describe, it, expect } from 'vitest'
import { nameStem, recipientSplits, resolveRecipients, searchStem } from './recipients.js'

const pool = [
  { key: 'u1', names: ['Васильев Пётр Ильич', 'vasilev.p'] },
  { key: 'u2', names: ['Иванов Иван Иванович', 'ivanov.i'] },
  { key: 'g9', names: ['Отчётность'] },
]

describe('адресат сообщения из фразы', () => {
  it('снимает окончание дательного падежа', () => {
    expect(nameStem('Васе')).toBe('вас')
    expect(nameStem('Ивану')).toBe('иван')
    expect(nameStem('Ли')).toBe('ли')
  })

  it('находит человека по имени в дательном падеже', () => {
    const r = resolveRecipients('петру созвон в 15:00', pool)
    expect(r.matches.map((m) => m.key)).toEqual(['u1'])
    expect(r.text).toBe('созвон в 15:00')
  })

  it('двусловное имя выигрывает у односложного', () => {
    const r = resolveRecipients('иванову ивану привет', pool)
    expect(r.matches.map((m) => m.key)).toEqual(['u2'])
    expect(r.text).toBe('привет')
  })

  it('явный разделитель главнее перебора слов', () => {
    const r = resolveRecipients('васильеву: привет иванову', pool)
    expect(r.matches.map((m) => m.key)).toEqual(['u1'])
    expect(r.text).toBe('привет иванову')
  })

  it('находит по логину и по названию группы', () => {
    expect(resolveRecipients('ivanov привет', pool).matches.map((m) => m.key)).toEqual(['u2'])
    expect(resolveRecipients('отчётность сдали', pool).matches.map((m) => m.key)).toEqual(['g9'])
  })

  it('без текста — только адресат (открыть чат)', () => {
    const r = resolveRecipients('васе', pool)
    expect(r.matches.map((m) => m.key)).toEqual(['u1'])
    expect(r.text).toBe('')
  })

  it('неизвестный адресат — пустой список', () => {
    expect(resolveRecipients('сидорову привет', pool).matches).toEqual([])
  })

  it('варианты разбиения — длинные имена первыми', () => {
    expect(recipientSplits('а б в г')).toEqual([
      { name: 'а б в', text: 'г' },
      { name: 'а б', text: 'в г' },
      { name: 'а', text: 'б в г' },
    ])
  })

  it('на сервере ищем по основе первого слова', () => {
    expect(searchStem('васе созвон')).toBe('вас')
  })
})
