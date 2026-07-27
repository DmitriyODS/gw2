import { describe, it, expect } from 'vitest'
import { parseQuickCommand } from './quickCommands.js'

describe('быстрые команды поиска', () => {
  it('разбирает создание задачи и поднимает первую букву названия', () => {
    expect(parseQuickCommand('создай задачу институт что-то там'))
      .toEqual({ kind: 'task', title: 'Институт что-то там' })
  })

  it('понимает разные глаголы и падежи', () => {
    expect(parseQuickCommand('добавь заметку список покупок').kind).toBe('note')
    expect(parseQuickCommand('Новая задача: отчёт за июль'))
      .toEqual({ kind: 'task', title: 'Отчёт за июль' })
    expect(parseQuickCommand('create note ideas').kind).toBe('note')
  })

  it('снимает кавычки вокруг названия', () => {
    expect(parseQuickCommand('создать заметку «Черновик»').title).toBe('Черновик')
  })

  it('без названия возвращает пустой заголовок — форма откроется чистой', () => {
    expect(parseQuickCommand('создай задачу')).toEqual({ kind: 'task', title: '' })
  })

  it('обычный поиск командой не считает', () => {
    expect(parseQuickCommand('задача про институт')).toBeNull()
    expect(parseQuickCommand('заметки по проекту')).toBeNull()
    expect(parseQuickCommand('')).toBeNull()
  })

  it('разбирает создание доски', () => {
    expect(parseQuickCommand('создай доску схема процесса'))
      .toEqual({ kind: 'board', title: 'Схема процесса' })
  })

  describe('напоминания', () => {
    // Понедельник, 27 июля 2026, 10:00.
    const now = new Date(2026, 6, 27, 10, 0, 0)

    it('вынимает срок из фразы и оставляет название', () => {
      const cmd = parseQuickCommand('напомни позвонить в банк завтра в 9:00', now)
      expect(cmd.kind).toBe('reminder')
      expect(cmd.title).toBe('Позвонить в банк')
      expect(cmd.at.getDate()).toBe(28)
      expect(cmd.at.getHours()).toBe(9)
    })

    it('понимает и форму с существительным, и повтор', () => {
      const cmd = parseQuickCommand('создай напоминание зарядка каждый день в 7:00', now)
      expect(cmd.title).toBe('Зарядка')
      expect(cmd.repeat).toEqual({ kind: 'daily', interval: 1, days: [] })
    })

    it('без срока отдаёт только название — время дозадаст форма', () => {
      const cmd = parseQuickCommand('напомни мне позвонить в банк', now)
      expect(cmd).toEqual({ kind: 'reminder', title: 'Позвонить в банк', at: null, repeat: null })
    })
  })

  it('разбирает отправку сообщения (адресата ищет utils/recipients)', () => {
    expect(parseQuickCommand('напиши васе созвон в 15:00'))
      .toEqual({ kind: 'message', rest: 'васе созвон в 15:00' })
    expect(parseQuickCommand('отправь ivanov привет').kind).toBe('message')
    // «запиши …» — это создание, а не отправка.
    expect(parseQuickCommand('запиши задачу отчёт').kind).toBe('task')
  })
})
