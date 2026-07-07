import { describe, it, expect } from 'vitest'
import {
  PET_STAGES, PET_SPECIES, NATURAL_SPECIES, PERSONALITIES, RARITY_TAG,
  petEmoji, shopItemTitle, shopItemEmoji, activityText, activityIcon,
  formatMinutes, avatarUrl,
} from './pets.js'

describe('pets константы (паритет с petsvc)', () => {
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

  it('4 редкости смаплены на существующие теги проекта', () => {
    expect(Object.keys(RARITY_TAG).sort()).toEqual(['common', 'epic', 'legendary', 'rare'])
  })
})

describe('pets.petEmoji', () => {
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

describe('pets.shopItemTitle/shopItemEmoji', () => {
  it('species — из каталога видов', () => {
    expect(shopItemTitle({ kind: 'species', key: 'cat' })).toBe('Котёнок')
    expect(shopItemEmoji({ kind: 'species', key: 'cat' })).toBe('🐱')
  })

  it('accessory/skin — из каталога товаров', () => {
    expect(shopItemTitle({ kind: 'accessory', key: 'crown' })).toBe('Корона')
    expect(shopItemEmoji({ kind: 'accessory', key: 'crown' })).toBe('👑')
  })

  it('неизвестный ключ — фолбэк на сам ключ/дефолтный эмодзи', () => {
    expect(shopItemTitle({ kind: 'accessory', key: 'zzz' })).toBe('zzz')
    expect(shopItemEmoji({ kind: 'accessory', key: 'zzz' })).toBe('🎁')
  })
})

describe('pets.activityText/activityIcon', () => {
  it('известный kind — человекочитаемый текст и иконка', () => {
    expect(activityText({ kind: 'walked', payload: {} })).toBe('Прогулка')
    expect(activityIcon({ kind: 'walked' })).toBe('directions_walk')
  })

  it('fed — подставляет стрик в текст', () => {
    expect(activityText({ kind: 'fed', payload: { streak: 5 } })).toContain('5 дн.')
  })

  it('item_bought — подставляет название товара', () => {
    expect(activityText({ kind: 'item_bought', payload: { key: 'crown' } })).toContain('Корона')
  })

  it('неизвестный kind — не падает', () => {
    expect(activityText({ kind: 'zzz', payload: {} })).toBe('zzz')
    expect(activityIcon({ kind: 'zzz' })).toBe('info')
  })
})

describe('pets.formatMinutes', () => {
  it('форматирует минуты в ч/мин', () => {
    expect(formatMinutes(0)).toBe('меньше минуты')
    expect(formatMinutes(-3)).toBe('меньше минуты')
    expect(formatMinutes(45)).toBe('45 мин')
    expect(formatMinutes(60)).toBe('1 ч')
    expect(formatMinutes(90)).toBe('1 ч 30 мин')
  })
})

describe('pets.avatarUrl', () => {
  it('аватар из файла, иначе identicon; null-пользователь → null', () => {
    expect(avatarUrl(null)).toBeNull()
    expect(avatarUrl({ id: 9, avatar_path: 'avatars/a.png' })).toBe('/uploads/avatars/a.png')
    expect(avatarUrl({ id: 9, avatar_path: null })).toBe('/api/users/9/identicon')
  })
})
