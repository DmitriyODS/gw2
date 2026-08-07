import { beforeEach, describe, expect, it } from 'vitest'
import { QUICK_REACTIONS, rememberReaction, suggestedReactions } from './reactions.js'

describe('reactions', () => {
  beforeEach(() => localStorage.clear())

  it('без истории показывает набор как есть', () => {
    expect(suggestedReactions.value).toEqual(QUICK_REACTIONS)
  })

  it('часто поставленные выходят вперёд', () => {
    rememberReaction('🤣') // последний в наборе
    rememberReaction('🤣')
    rememberReaction('👀')

    expect(suggestedReactions.value.slice(0, 2)).toEqual(['🤣', '👀'])
  })

  it('набор не теряет и не дублирует эмодзи', () => {
    rememberReaction('💯')
    rememberReaction('🙏')

    const shown = suggestedReactions.value
    expect(shown).toHaveLength(QUICK_REACTIONS.length)
    expect(new Set(shown).size).toBe(QUICK_REACTIONS.length)
  })

  it('остальные сохраняют исходный порядок', () => {
    rememberReaction('🤝')

    const rest = suggestedReactions.value.slice(1)
    expect(rest).toEqual(QUICK_REACTIONS.filter((e) => e !== '🤝'))
  })
})
