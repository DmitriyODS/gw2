import { describe, it, expect } from 'vitest'
import { Schema } from '@tiptap/pm/model'
import { findMatches } from './useNoteFind.js'

const schema = new Schema({
  nodes: {
    doc: { content: 'block+' },
    paragraph: { group: 'block', content: 'text*', toDOM: () => ['p', 0] },
    text: {},
  },
  marks: { bold: { toDOM: () => ['strong', 0] } },
})

const p = (...content) => schema.node('paragraph', null, content)
const bold = (text) => schema.text(text, [schema.mark('bold')])

describe('поиск совпадений в документе заметки', () => {
  it('находит слово, разорванное форматированием', () => {
    // «до|гово|р тут» — три текстовых узла одного абзаца.
    const doc = schema.node('doc', null, [p(schema.text('до'), bold('гово'), schema.text('р тут'))])
    expect(findMatches(doc, 'договор')).toEqual([{ from: 1, to: 8 }])
  })

  it('ищет без учёта регистра и возвращает все вхождения', () => {
    const doc = schema.node('doc', null, [
      p(schema.text('Договор тут')),
      p(schema.text('и договор снова')),
    ])
    expect(findMatches(doc, 'договор')).toEqual([
      { from: 1, to: 8 },
      { from: 16, to: 23 },
    ])
  })

  it('через границу абзацев совпадение не тянет', () => {
    const doc = schema.node('doc', null, [p(schema.text('дого')), p(schema.text('вор'))])
    expect(findMatches(doc, 'договор')).toEqual([])
  })
})
