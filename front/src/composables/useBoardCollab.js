// Совместное рисование на доске: присутствие («кто сейчас на доске»), курсоры
// соавторов и трансляция сцены. Транспорт — POST /api/boards/:id/collab
// (broadcast без сохранения) + сокет-события board_collab:* через gateway.
// Правки едут ПООБЪЕКТНО (kind=ops): у каждого объекта свой id, поэтому
// одновременная работа над разными объектами не приводит к потере чужих
// штрихов — в отличие от пересылки всей сцены. Сцену целиком шлём только на
// «структурные» изменения (слои, фон): kind=scene, и она применяется как LWW,
// КРОМЕ момента, когда прямо сейчас рисуют локально.
import { computed, onBeforeUnmount, ref } from 'vue'
import { sendCollab } from '@/api/boards.js'
import { getSocket } from '@/socket/index.js'
import { useAuthStore } from '@/stores/auth.js'
import { TASK_COLORS } from '@/utils/taskColors.js'

const CURSOR_THROTTLE_MS = 120
const SCENE_THROTTLE_MS = 500
const HEARTBEAT_MS = 10_000
const STALE_MS = 30_000

function colorFor(userId) {
  return TASK_COLORS[Math.abs(userId) % TASK_COLORS.length].id
}

export function useBoardCollab({ boardId, canEdit, isDrawing, getScene, getTitle, onRemoteOps, onRemoteScene, onRemoteTitle }) {
  const auth = useAuthStore()
  const participants = ref(new Map()) // userId → {fio, color, x, y, lastSeen}

  const others = computed(() => [...participants.value.entries()]
    .map(([id, p]) => ({ user_id: id, ...p })))

  let started = false
  let heartbeatTimer = null
  let pruneTimer = null
  let cursorTimer = null
  let sceneTimer = null
  let pendingCursor = null

  const send = (body) => sendCollab(boardId.value, body).catch(() => { /* collab не критичен */ })

  function touch(userId, patch = {}) {
    const map = new Map(participants.value)
    const prev = map.get(userId) || { fio: 'Участник', color: colorFor(userId), x: null, y: null }
    map.set(userId, { ...prev, ...patch, lastSeen: Date.now() })
    participants.value = map
  }

  function drop(userId) {
    if (!participants.value.has(userId)) return
    const map = new Map(participants.value)
    map.delete(userId)
    participants.value = map
  }

  // ── Исходящие сигналы ──
  /** Курсор — самый частый сигнал, поэтому строго троттлится. */
  function sendCursor(point) {
    pendingCursor = point
    if (cursorTimer) return
    cursorTimer = setTimeout(() => {
      cursorTimer = null
      if (pendingCursor) send({ kind: 'cursor', cursor: { from: Math.round(pendingCursor.x), to: Math.round(pendingCursor.y) } })
    }, CURSOR_THROTTLE_MS)
  }

  /** Операции холста: их шлём немедленно — это горячий путь совместной работы. */
  function sendOps(ops) {
    if (!canEdit.value || !ops?.length) return
    send({ kind: 'ops', ops })
  }

  function sendScene() {
    if (!canEdit.value || sceneTimer) return
    sceneTimer = setTimeout(() => {
      sceneTimer = null
      const scene = getScene?.()
      if (!scene) return
      const body = { kind: 'scene', scene }
      const title = getTitle?.()
      if (title != null) body.title = title
      send(body)
    }, SCENE_THROTTLE_MS)
  }

  // ── Входящие события ──
  const notMine = (p) => p.board_id === boardId.value && p.user_id !== auth.userId

  function onJoin(p) {
    if (!notMine(p)) return
    touch(p.user_id, p.fio ? { fio: p.fio } : {})
    // Отвечаем присутствием, чтобы новоприбывший узнал о нас.
    setTimeout(() => send({ kind: 'cursor', cursor: { from: 0, to: 0 } }), 300)
  }

  function onCursor(p) {
    if (!notMine(p)) return
    touch(p.user_id, p.cursor ? { x: p.cursor.from, y: p.cursor.to } : {})
  }

  function onLeave(p) {
    if (!notMine(p)) return
    drop(p.user_id)
  }

  function onScene(p) {
    if (!notMine(p)) return
    touch(p.user_id, p.cursor ? { x: p.cursor.from, y: p.cursor.to } : {})
    if (p.title != null) onRemoteTitle?.(p.title)
    // Пока идёт свой жест — чужую сцену не применяем, иначе штрих оборвётся.
    if (p.scene && !isDrawing?.()) onRemoteScene?.(p.scene)
  }

  function onOps(p) {
    if (!notMine(p) || !Array.isArray(p.ops)) return
    // Операции адресны — их можно применять даже посреди своего жеста.
    onRemoteOps?.(p.ops)
  }

  // ── Жизненный цикл ──
  function start() {
    if (started || !boardId.value) return
    started = true
    const socket = getSocket()
    socket?.on('board_collab:join', onJoin)
    socket?.on('board_collab:cursor', onCursor)
    socket?.on('board_collab:leave', onLeave)
    socket?.on('board_collab:scene', onScene)
    socket?.on('board_collab:ops', onOps)
    send({ kind: 'join' })
    heartbeatTimer = setInterval(() => send({ kind: 'cursor', cursor: { from: 0, to: 0 } }), HEARTBEAT_MS)
    pruneTimer = setInterval(() => {
      const now = Date.now()
      let changed = false
      const map = new Map(participants.value)
      for (const [id, p] of map) {
        if (now - p.lastSeen > STALE_MS) { map.delete(id); changed = true }
      }
      if (changed) participants.value = map
    }, STALE_MS / 2)
    window.addEventListener('beforeunload', sendLeave)
  }

  function sendLeave() { send({ kind: 'leave' }) }

  function stop() {
    if (!started) return
    started = false
    sendLeave()
    const socket = getSocket()
    socket?.off('board_collab:join', onJoin)
    socket?.off('board_collab:cursor', onCursor)
    socket?.off('board_collab:leave', onLeave)
    socket?.off('board_collab:scene', onScene)
    socket?.off('board_collab:ops', onOps)
    clearInterval(heartbeatTimer)
    clearInterval(pruneTimer)
    clearTimeout(cursorTimer)
    clearTimeout(sceneTimer)
    cursorTimer = sceneTimer = null
    window.removeEventListener('beforeunload', sendLeave)
    participants.value = new Map()
  }

  onBeforeUnmount(stop)

  return { others, start, stop, sendCursor, sendScene, sendOps }
}
