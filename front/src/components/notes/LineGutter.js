import { Extension } from '@tiptap/core'
import { Plugin, PluginKey, TextSelection } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'

/* Поле выделения слева от строки — как в Word.

   Клик в левом поле листа выделяет строку целиком, протяжка вниз или вверх —
   несколько строк подряд. Пока указатель в поле, строка под ним подсвечивается,
   а курсор меняется на стрелку — иначе о возможности никто не догадается.

   Зона поля — это левый padding самого редактора: отдельного элемента нет,
   поэтому попадание считается по координатам, а подсветка рисуется
   ProseMirror-декорацией (как курсоры соавторов и подсветка поиска). */

export const lineGutterKey = new PluginKey('lineGutter')

// Ширина зоны отсчитывается от левого края листа. Берём минимум из padding и
// этого числа: захватывать сам текст нельзя, иначе обычный клик по первой
// букве строки начнёт выделять строку целиком.
const MAX_GUTTER = 30

/** Левая граница текста и ширина поля выделения для текущей вёрстки. */
function gutterBox(view) {
  const rect = view.dom.getBoundingClientRect()
  const padLeft = parseFloat(getComputedStyle(view.dom).paddingLeft) || 0
  return { left: rect.left, width: Math.min(padLeft, MAX_GUTTER), textLeft: rect.left + padLeft }
}

/** Пользователь в поле выделения (а не в тексте и не за пределами листа)? */
function inGutter(view, event) {
  const box = gutterBox(view)
  return box.width > 4 && event.clientX >= box.left && event.clientX < box.left + box.width
}

/** Границы строки (текстового блока) под координатой Y. null — не нашли. */
function blockAt(view, clientY) {
  const box = gutterBox(view)
  const found = view.posAtCoords({ left: box.textLeft + 4, top: clientY })
  if (!found) return null
  const $pos = view.state.doc.resolve(found.inside >= 0 ? found.inside + 1 : found.pos)
  // Поднимаемся до текстового блока: клик мог прийтись на элемент списка или
  // ячейку таблицы, а выделять надо саму строку.
  for (let depth = $pos.depth; depth > 0; depth -= 1) {
    if ($pos.node(depth).isTextblock) {
      return { from: $pos.start(depth), to: $pos.end(depth) }
    }
  }
  return { from: $pos.start(), to: $pos.end() }
}

/** Выделить строки от якоря до строки под курсором. */
function selectRange(view, anchor, clientY) {
  const block = blockAt(view, clientY)
  if (!block) return
  const from = Math.min(anchor.from, block.from)
  const to = Math.max(anchor.to, block.to)
  const sel = view.state.selection
  // Ничего не изменилось — не гоняем транзакции на каждый пиксель протяжки.
  if (sel.from === from && sel.to === to) return
  const { doc, tr } = view.state
  view.dispatch(tr.setSelection(TextSelection.create(doc, from, to)).scrollIntoView())
  // Без фокуса выделение не «настоящее»: браузер не подсветит его и не отдаст
  // в буфер обмена по Ctrl+C.
  if (!view.hasFocus()) view.focus()
}

export const LineGutter = Extension.create({
  name: 'lineGutter',

  addProseMirrorPlugins() {
    // Якорь протяжки живёт вне состояния плагина: это жест мыши, а не часть
    // документа, и хранить его в транзакциях незачем.
    let drag = null

    return [
      new Plugin({
        key: lineGutterKey,

        state: {
          init: () => ({ hover: null }),
          apply(tr, value) {
            const next = tr.getMeta(lineGutterKey)
            if (next !== undefined) return { hover: next }
            // Документ поменялся — прежние координаты подсветки уже неверны.
            return tr.docChanged ? { hover: null } : value
          },
        },

        props: {
          decorations(state) {
            const hover = lineGutterKey.getState(state)?.hover
            if (!hover) return null
            return DecorationSet.create(state.doc, [
              Decoration.inline(hover.from, hover.to, { class: 'ne-line-hover' }),
            ])
          },

          handleDOMEvents: {
            mousemove(view, event) {
              // Во время протяжки ведём выделение, иначе — только подсветка.
              if (drag) {
                selectRange(view, drag, event.clientY)
                return false
              }
              const hover = inGutter(view, event) ? blockAt(view, event.clientY) : null
              const prev = lineGutterKey.getState(view.state)?.hover
              const same = prev && hover && prev.from === hover.from && prev.to === hover.to
              if (same || (!prev && !hover)) return false
              view.dom.classList.toggle('ne-gutter-zone', Boolean(hover))
              view.dispatch(view.state.tr.setMeta(lineGutterKey, hover))
              return false
            },

            mouseleave(view) {
              if (lineGutterKey.getState(view.state)?.hover) {
                view.dom.classList.remove('ne-gutter-zone')
                view.dispatch(view.state.tr.setMeta(lineGutterKey, null))
              }
              return false
            },

            mousedown(view, event) {
              if (event.button !== 0 || !inGutter(view, event)) return false
              const block = blockAt(view, event.clientY)
              if (!block) return false
              // Гасим родное поведение: иначе ProseMirror поставит каретку в
              // точку клика и наше выделение тут же схлопнется.
              event.preventDefault()
              event.stopPropagation()
              drag = block
              selectRange(view, block, event.clientY)

              // Протяжка продолжается и за пределами листа — слушаем окно.
              const move = (e) => selectRange(view, drag, e.clientY)
              const up = () => {
                drag = null
                window.removeEventListener('mousemove', move)
                window.removeEventListener('mouseup', up)
              }
              window.addEventListener('mousemove', move)
              window.addEventListener('mouseup', up)
              return true
            },
          },
        },
      }),
    ]
  },
})
