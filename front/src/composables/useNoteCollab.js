// Совместное редактирование заметки: присутствие («кто сейчас в заметке»),
// живые курсоры и трансляция правок (документ + название). Транспорт —
// POST /api/notes/:id/collab (broadcast без сохранения) + сокет-события
// note_collab:* через gateway. Конфликты — last-write-wins: удалённый
// документ применяется всегда, КРОМЕ момента, когда локально печатают прямо
// сейчас (isTyping — фокус в редакторе + свежий ввод); «грязный» автосейв
// сам по себе применение не блокирует — иначе при одновременном наборе
// участники глушили бы правки друг друга на всё окно дебаунса.
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import { sendCollab } from '@/api/notes.js'
import { getSocket } from '@/socket/index.js'
import { useAuthStore } from '@/stores/auth.js'
import { TASK_COLORS } from '@/utils/taskColors.js'

const CURSOR_THROTTLE_MS = 300
const DOC_THROTTLE_MS = 700
const HEARTBEAT_MS = 10_000
const STALE_MS = 30_000

const collabKey = new PluginKey('noteCollabCursors')

// Детеминированный цвет участника из палитры тегов.
function colorFor(userId) {
  return TASK_COLORS[Math.abs(userId) % TASK_COLORS.length].id
}

export function useNoteCollab({ noteId, editorRef, canEdit, isTyping, getTitle, onRemoteTitle, fallbackNames }) {
  const auth = useAuthStore()
  const participants = ref(new Map()) // userId → {fio, color, cursor, lastSeen}

  const others = computed(() => [...participants.value.entries()]
    .map(([id, p]) => ({ id, ...p })))

  let started = false
  let heartbeatTimer = null
  let pruneTimer = null
  let cursorTimer = null
  let docTimer = null
  let pluginRegistered = false

  const send = (body) => sendCollab(noteId.value, body).catch(() => { /* collab не критичен */ })

  function currentCursor() {
    const ed = editorRef.value?.editor
    if (!ed) return null
    const { from, to } = ed.state.selection
    return { from, to }
  }

  function touch(userId, patch = {}) {
    const map = new Map(participants.value)
    const prev = map.get(userId) || {
      fio: fallbackNames?.(userId) || 'Участник',
      color: colorFor(userId),
      cursor: null,
    }
    map.set(userId, { ...prev, ...patch, lastSeen: Date.now() })
    participants.value = map
    refreshDecorations()
  }

  function drop(userId) {
    if (!participants.value.has(userId)) return
    const map = new Map(participants.value)
    map.delete(userId)
    participants.value = map
    refreshDecorations()
  }

  // ── Исходящие сигналы ──
  function sendCursorThrottled() {
    if (cursorTimer) return
    cursorTimer = setTimeout(() => {
      cursorTimer = null
      const cursor = currentCursor()
      if (cursor) send({ kind: 'cursor', cursor })
    }, CURSOR_THROTTLE_MS)
  }

  function sendDocThrottled() {
    if (!canEdit.value || docTimer) return
    docTimer = setTimeout(() => {
      docTimer = null
      const ed = editorRef.value?.editor
      if (!ed) return
      const body = { kind: 'doc', doc: ed.getJSON(), cursor: currentCursor() }
      // Название едет вместе с документом — у соавторов оно меняется живьём.
      const title = getTitle?.()
      if (title != null) body.title = title
      send(body)
    }, DOC_THROTTLE_MS)
  }

  // ── Входящие события ──
  const isMine = (p) => p.note_id !== noteId.value || p.user_id === auth.userId

  function onJoin(p) {
    if (isMine(p)) return
    touch(p.user_id, p.fio ? { fio: p.fio } : {})
    // Отвечаем курсором, чтобы новоприбывший узнал о нас (и наше ФИО он
    // возьмёт из members/владельца — cursor ФИО не несёт).
    setTimeout(() => send({ kind: 'cursor', cursor: currentCursor() || { from: 0, to: 0 } }), 300)
  }

  function onCursor(p) {
    if (isMine(p)) return
    touch(p.user_id, { cursor: p.cursor || null })
  }

  function onLeave(p) {
    if (isMine(p)) return
    drop(p.user_id)
  }

  function onDoc(p) {
    if (isMine(p)) return
    touch(p.user_id, { cursor: p.cursor || null })
    // Название применяется независимо от набора в теле — решение «не затирать,
    // пока курсор в поле названия» принимает редактор.
    if (p.title != null) onRemoteTitle?.(p.title)
    const ed = editorRef.value?.editor
    if (!ed || !p.doc) return
    // Не затираем только живой набор (фокус + свежий ввод); победит последний.
    if (isTyping?.()) return
    const sel = ed.state.selection
    ed.commands.setContent(p.doc, false) // без emitUpdate — не наш ввод
    const size = ed.state.doc.content.size
    ed.commands.setTextSelection({ from: Math.min(sel.from, size), to: Math.min(sel.to, size) })
  }

  // ── Курсоры в тексте (ProseMirror-декорации) ──
  function buildDecorations(state) {
    const decos = []
    const size = state.doc.content.size
    for (const [id, p] of participants.value) {
      if (!p.cursor) continue
      const from = Math.min(p.cursor.from ?? 0, size)
      const to = Math.min(p.cursor.to ?? from, size)
      if (to > from) {
        decos.push(Decoration.inline(from, to, {
          class: 'nc-selection',
          style: `--nc-color: var(--tag-${p.color}-accent); background: var(--tag-${p.color}-surface);`,
        }))
      }
      const caret = document.createElement('span')
      caret.className = 'nc-caret'
      caret.style.setProperty('--nc-color', `var(--tag-${p.color}-accent)`)
      const label = document.createElement('span')
      label.className = 'nc-caret-label'
      label.textContent = p.fio
      caret.appendChild(label)
      decos.push(Decoration.widget(to, caret, { key: `nc-${id}`, side: 1 }))
    }
    return DecorationSet.create(state.doc, decos)
  }

  function refreshDecorations() {
    const ed = editorRef.value?.editor
    if (!ed || ed.isDestroyed) return
    // Пустая транзакция — декорации пересчитываются из props.decorations.
    ed.view.dispatch(ed.state.tr)
  }

  function registerPlugin() {
    const ed = editorRef.value?.editor
    if (!ed || pluginRegistered) return
    ed.registerPlugin(new Plugin({
      key: collabKey,
      props: { decorations: buildDecorations },
    }))
    ed.on('selectionUpdate', sendCursorThrottled)
    ed.on('update', sendDocThrottled)
    pluginRegistered = true
  }

  // ── Жизненный цикл ──
  function start() {
    if (started || !noteId.value) return
    started = true
    const socket = getSocket()
    socket?.on('note_collab:join', onJoin)
    socket?.on('note_collab:cursor', onCursor)
    socket?.on('note_collab:leave', onLeave)
    socket?.on('note_collab:doc', onDoc)
    registerPlugin()
    send({ kind: 'join' })
    heartbeatTimer = setInterval(() => send({ kind: 'cursor', cursor: currentCursor() || { from: 0, to: 0 } }), HEARTBEAT_MS)
    pruneTimer = setInterval(() => {
      const now = Date.now()
      let changed = false
      const map = new Map(participants.value)
      for (const [id, p] of map) {
        if (now - p.lastSeen > STALE_MS) { map.delete(id); changed = true }
      }
      if (changed) { participants.value = map; refreshDecorations() }
    }, STALE_MS / 2)
    window.addEventListener('beforeunload', sendLeave)
  }

  function sendLeave() { send({ kind: 'leave' }) }

  function stop() {
    if (!started) return
    started = false
    sendLeave()
    const socket = getSocket()
    socket?.off('note_collab:join', onJoin)
    socket?.off('note_collab:cursor', onCursor)
    socket?.off('note_collab:leave', onLeave)
    socket?.off('note_collab:doc', onDoc)
    const ed = editorRef.value?.editor
    if (ed && !ed.isDestroyed) {
      ed.off('selectionUpdate', sendCursorThrottled)
      ed.off('update', sendDocThrottled)
      if (pluginRegistered) ed.unregisterPlugin(collabKey)
    }
    pluginRegistered = false
    clearInterval(heartbeatTimer)
    clearInterval(pruneTimer)
    clearTimeout(cursorTimer)
    clearTimeout(docTimer)
    cursorTimer = docTimer = null
    window.removeEventListener('beforeunload', sendLeave)
    participants.value = new Map()
  }

  // Редактор может смонтироваться позже start() (лоадер заметки).
  watch(() => editorRef.value?.editor, (ed) => { if (ed && started) registerPlugin() })

  onBeforeUnmount(stop)

  return { others, start, stop, sendDoc: sendDocThrottled }
}
