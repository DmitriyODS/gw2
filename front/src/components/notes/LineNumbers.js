import { Extension } from '@tiptap/core'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'

/* Номера строк в левом поле листа — как в редакторах кода.

   «Строка» здесь — блок верхнего уровня (абзац, заголовок, цитата, код), а не
   визуальная строка: длинный абзац переносится, но остаётся одной записью
   документа, и нумеровать его куски было бы враньём.

   Списки и таблицы пропускаем сознательно: их номер пришёлся бы на маркер или
   на первую ячейку и читался бы как часть содержимого. Счётчик идёт только по
   пронумерованным блокам, поэтому в колонке нет дыр.

   Сам номер рисует CSS (`content: attr(data-line)`) — декорация лишь проставляет
   атрибут: так номера не попадают ни в документ, ни в копирование, ни в
   выгрузку. */

export const lineNumbersKey = new PluginKey('lineNumbers')

function numbering(doc) {
  const decorations = []
  let line = 0

  doc.forEach((node, offset) => {
    if (!node.isTextblock) return
    line += 1
    decorations.push(Decoration.node(offset, offset + node.nodeSize, {
      class: 'ne-numbered',
      'data-line': String(line),
    }))
  })

  return DecorationSet.create(doc, decorations)
}

export const LineNumbers = Extension.create({
  name: 'lineNumbers',

  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: lineNumbersKey,

        state: {
          init: (_, { doc }) => numbering(doc),
          // Пересчитываем только на правках документа: перемещение каретки
          // нумерацию не меняет, а пересборка набора декораций не бесплатна.
          apply: (tr, value) => (tr.docChanged ? numbering(tr.doc) : value),
        },

        props: {
          decorations(state) {
            return lineNumbersKey.getState(state)
          },
        },
      }),
    ]
  },
})
