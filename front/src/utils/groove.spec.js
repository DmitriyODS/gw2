import { describe, it, expect } from 'vitest'
import {
  KUDOS_CATEGORIES, PET_STAGES, PET_SPECIES, NATURAL_SPECIES, PERSONALITIES,
  petEmoji, formatMinutes, dayKey, avatarUrl,
} from './groove.js'

describe('groove константы (паритет с groovesvc)', () => {
  it('категории кудосов — ровно helped/quality/speed', () => {
    expect(Object.keys(KUDOS_CATEGORIES).sort()).toEqual(['helped', 'quality', 'speed'])
    for (const key of Object.keys(KUDOS_CATEGORIES)) {
      expect(KUDOS_CATEGORIES[key].icon).toBeTruthy()
      expect(KUDOS_CATEGORIES[key].title).toBeTruthy()
    }
  })

  it('7 стадий питомца', () => {
    expect(PET_STAGES).toHaveLength(7)
    expect(PET_STAGES[0]).toBe('Яйцо')
  })

  it('природные виды — подмножество каталога видов', () => {
    for (const s of NATURAL_SPECIES) expect(PET_SPECIES[s]).toBeTruthy()
  })

  it('у каждого характера есть эмодзи, заголовок и критерий', () => {
    for (const key of Object.keys(PERSONALITIES)) {
      const p = PERSONALITIES[key]
      expect(p.emoji && p.title && p.desc, key).toBeTruthy()
    }
  })
})

describe('groove.petEmoji', () => {
  it('стадия 0 (или нет питомца) → яйцо', () => {
    expect(petEmoji(null)).toBe('🥚')
    expect(petEmoji({ stage: 0, species: 'owl' })).toBe('🥚')
  })

  it('малыш природного вида на 1-й стадии — общий вид 🐣', () => {
    expect(petEmoji({ stage: 1, species: 'owl' })).toBe('🐣')
  })

  it('покупной вид показывается сразу с 1-й стадии', () => {
    expect(petEmoji({ stage: 1, species: 'cat' })).toBe(PET_SPECIES.cat.emoji)
  })

  it('со 2-й стадии — эмодзи вида, неизвестный вид → лиса', () => {
    expect(petEmoji({ stage: 3, species: 'lark' })).toBe(PET_SPECIES.lark.emoji)
    expect(petEmoji({ stage: 3, species: 'zzz' })).toBe('🦊')
  })
})

describe('groove.formatMinutes', () => {
  it('форматирует минуты в ч/мин', () => {
    expect(formatMinutes(0)).toBe('меньше минуты')
    expect(formatMinutes(-3)).toBe('меньше минуты')
    expect(formatMinutes(45)).toBe('45 мин')
    expect(formatMinutes(60)).toBe('1 ч')
    expect(formatMinutes(90)).toBe('1 ч 30 мин')
  })
})

describe('groove.dayKey', () => {
  it('локальный ключ дня YYYY-MM-DD с нулями', () => {
    // Фиксированная локальная дата — без Date.now в ассертах.
    expect(dayKey(new Date(2026, 0, 5, 23, 59))).toBe('2026-01-05')
    expect(dayKey(new Date(2026, 11, 31, 0, 0))).toBe('2026-12-31')
  })
})

describe('groove.avatarUrl', () => {
  it('аватар из файла, иначе identicon; null-пользователь → null', () => {
    expect(avatarUrl(null)).toBeNull()
    expect(avatarUrl({ id: 9, avatar_path: 'avatars/a.png' })).toBe('/uploads/avatars/a.png')
    expect(avatarUrl({ id: 9, avatar_path: null })).toBe('/api/users/9/identicon')
  })
})
