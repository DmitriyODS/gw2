import { describe, it, expect } from 'vitest'
import { BACKUP_SECTIONS, ALL_SECTION_KEYS } from './backupSections.js'

// Смоук: оснастка жива + инварианты набора разделов бэкапа.
describe('backupSections', () => {
  it('ключи уникальны и покрывают все известные разделы', () => {
    const keys = BACKUP_SECTIONS.map((s) => s.key)
    expect(new Set(keys).size).toBe(keys.length)
    expect(ALL_SECTION_KEYS).toEqual(keys)
    // «other» — обязательный ловец не классифицированных таблиц (см. бэкенд).
    expect(keys).toContain('other')
  })

  it('у каждого раздела есть подпись и описание', () => {
    for (const s of BACKUP_SECTIONS) {
      expect(s.label, s.key).toBeTruthy()
      expect(s.desc, s.key).toBeTruthy()
    }
  })
})
