import { describe, expect, it } from 'vitest'
import { safeDoc } from './safeDoc.js'

describe('safeDoc', () => {
  it('вырезает __proto__ на любой глубине', () => {
    const doc = JSON.parse('{"type":"doc","content":[{"type":"image","attrs":{"src":"x","__proto__":{"onerror":"alert(1)"}}}]}')
    const clean = safeDoc(doc)
    const attrs = clean.content[0].attrs
    expect(Object.keys(attrs)).toEqual(['src'])
    expect(attrs.onerror).toBeUndefined()
  })

  it('чистый документ не меняет по содержимому', () => {
    const doc = { type: 'doc', content: [{ type: 'text', text: '__proto__' }] }
    expect(safeDoc(doc)).toEqual(doc)
  })
})
