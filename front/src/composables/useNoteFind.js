import { computed, ref, watch } from 'vue'
import { Plugin, PluginKey, TextSelection } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'

/* Поиск по открытой заметке (Ctrl+F): подсветка всех совпадений, переход
   между ними и счётчик. Совпадения ищем по тексту блока целиком, а не по
   отдельным текстовым узлам, — иначе слово, разорванное жирным начертанием
   («до**гово**р»), не находилось бы. Через границу блоков совпадение не
   тянем: позиции там не сплошные.

   Декорации живут в своём ProseMirror-плагине, как курсоры соавторов
   (useNoteCollab) — редактор перерисовывает их сам, документ не трогаем. */

const findKey = new PluginKey('noteFind')

export function useNoteFind(editorRef) {
  const open = ref(false)
  const query = ref('')
  const matches = ref([])   // [{ from, to }]
  const index = ref(0)

  let registered = false

  const total = computed(() => matches.value.length)
  const current = computed(() => (total.value ? index.value + 1 : 0))

  const editor = () => {
    const ed = editorRef.value?.editor
    return ed && !ed.isDestroyed ? ed : null
  }

  /* jump=false — пересчёт после правки текста: подсветку обновляем, но курсор
     не трогаем, иначе набор в заметке при открытой панели уводил бы каретку
     на очередное совпадение. */
  function search({ jump = true } = {}) {
    const ed = editor()
    const q = query.value.trim()
    if (!ed || !q) {
      matches.value = []
      index.value = 0
      redraw()
      return
    }
    matches.value = findMatches(ed.state.doc, q)
    index.value = Math.min(index.value, Math.max(matches.value.length - 1, 0))
    redraw()
    if (jump) scrollToCurrent()
  }

  function step(delta) {
    if (!total.value) return
    index.value = (index.value + delta + total.value) % total.value
    redraw()
    scrollToCurrent()
  }

  /* Курсор ставим в найденное место (и прокручиваем к нему) — закрыв панель,
     пользователь продолжит править ровно там. Фокус при этом остаётся в поле
     поиска: так же ведёт себя поиск в браузере. */
  function scrollToCurrent() {
    const ed = editor()
    const hit = matches.value[index.value]
    if (!ed || !hit) return
    const { doc, tr } = ed.state
    try {
      ed.view.dispatch(tr.setSelection(TextSelection.create(doc, hit.from, hit.to)).scrollIntoView())
    } catch {
      // Позиция устарела (документ правили) — подсветку оставляем как есть.
    }
  }

  function redraw() {
    const ed = editor()
    if (!ed) return
    // Пустая транзакция — декорации пересчитываются из props.decorations.
    ed.view.dispatch(ed.state.tr)
  }

  function decorations(state) {
    if (!open.value || !matches.value.length) return DecorationSet.empty
    const size = state.doc.content.size
    const decos = matches.value
      .filter((m) => m.to <= size)
      .map((m, i) => Decoration.inline(m.from, m.to, {
        class: i === index.value ? 'nf-hit nf-hit-current' : 'nf-hit',
      }))
    return DecorationSet.create(state.doc, decos)
  }

  function register() {
    const ed = editor()
    if (!ed || registered) return
    ed.registerPlugin(new Plugin({ key: findKey, props: { decorations } }))
    // Текст правят (в том числе соавторы) — подсветка обязана следовать за ним.
    ed.on('update', refresh)
    registered = true
  }

  function show() {
    register()
    open.value = true
    // Выделенный фрагмент — самый вероятный запрос (как в редакторах кода).
    const ed = editor()
    if (ed) {
      const { from, to } = ed.state.selection
      const picked = to > from ? ed.state.doc.textBetween(from, to, ' ', ' ').trim() : ''
      if (picked && picked.length <= 80) query.value = picked
    }
    search()
  }

  function hide() {
    open.value = false
    matches.value = []
    index.value = 0
    redraw()
  }

  const toggle = () => (open.value ? hide() : show())

  function refresh() {
    if (open.value) search({ jump: false })
  }

  watch(query, () => {
    index.value = 0
    search()
  })

  /** Ctrl/Cmd+F — панель поиска вместо браузерной; Esc её закрывает. */
  function onKeydown(e) {
    if ((e.ctrlKey || e.metaKey) && (e.key === 'f' || e.key === 'F' || e.key === 'а' || e.key === 'А')) {
      e.preventDefault()
      show()
      return true
    }
    if (e.key === 'Escape' && open.value) {
      e.preventDefault()
      hide()
      return true
    }
    return false
  }

  return { open, query, matches, index, total, current, show, hide, toggle, step, refresh, onKeydown }
}

/**
 * Позиции совпадений в документе. Текстовые узлы одного блока склеиваем —
 * марки (жирный, ссылка) рвут слово на узлы, но позиции внутри блока идут
 * подряд, поэтому смещение в склейке равно смещению в документе.
 */
export function findMatches(doc, query) {
  const needle = query.toLowerCase()
  const out = []
  let chunk = null

  const flush = () => {
    if (!chunk) return
    const hay = chunk.text.toLowerCase()
    let at = hay.indexOf(needle)
    while (at !== -1) {
      out.push({ from: chunk.pos + at, to: chunk.pos + at + needle.length })
      at = hay.indexOf(needle, at + needle.length)
    }
    chunk = null
  }

  doc.descendants((node, pos) => {
    if (node.isText) {
      if (chunk && chunk.pos + chunk.text.length === pos) chunk.text += node.text
      else { flush(); chunk = { text: node.text, pos } }
      return false
    }
    flush()
    return true
  })
  flush()
  return out
}
