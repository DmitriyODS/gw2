import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import * as api from '@/api/messenger.js'
import { useAuthStore } from './auth.js'
import { useMessengerStore } from './messenger.js'
import { useNetworkStore } from './network.js'
import { useNotificationsStore } from './notifications.js'

/* Очередь отправки сообщений — как в Telegram: сообщение сразу встаёт в ленту
   «с часиками» и уходит, когда есть связь, а поле ввода свободно. Очередь
   переживает перезагрузку (localStorage: на телефоне система выгружает
   свёрнутое приложение). Повтор безопасен: у каждого сообщения ключ
   идемпотентности client_id, и сервер вернёт уже созданное, если ответ на
   прошлую попытку потерялся по дороге. */

const STORAGE_KEY = 'gw_msg_outbox'
// Отправка из очереди ждёт дольше обычного запроса: на плохой сети лучше
// дождаться ответа, чем повторять.
const SEND_TIMEOUT_MS = 20000
const RETRY_STEPS_MS = [2000, 5000, 10000, 30000]

function newClientId() {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID()
  const hex = (n) => Array.from({ length: n }, () => Math.floor(Math.random() * 16).toString(16)).join('')
  return `${hex(8)}-${hex(4)}-4${hex(3)}-a${hex(3)}-${hex(12)}`
}

function load() {
  try {
    const raw = JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]')
    if (!Array.isArray(raw)) return []
    // «Отправлялось» в момент закрытия — неизвестно, дошло ли: повторим.
    return raw.map((it) => (it.status === 'sending' ? { ...it, status: 'pending' } : it))
  } catch {
    return []
  }
}

// Сбой, который пройдёт сам: нет сети, сервер перегружен или перезапускается.
function isTransient(e) {
  const status = e?.status ?? 0
  return e?.error === 'NETWORK_ERROR' || status === 0 || status === 408 || status === 429 || status >= 500
}

export const useMessageOutboxStore = defineStore('messageOutbox', () => {
  const items = ref(load())
  let flushing = false
  let retryTimer = null
  let attempt = 0
  let listening = false

  // Запись синхронная: телефон может выгрузить приложение сразу после
  // нажатия «Отправить», и отложенный до следующего тика снимок не успел бы.
  watch(items, (v) => {
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(v)) } catch { /* приватный режим */ }
  }, { deep: true, flush: 'sync' })

  const myId = () => useAuthStore().user?.id ?? null

  function listen() {
    if (listening || typeof window === 'undefined') return
    listening = true
    window.addEventListener('online', () => flush())
    watch(() => useNetworkStore().status, (s) => { if (s === 'online') flush() })
  }

  /** Неотправленные сообщения диалога в форме сообщения ленты. */
  function messagesFor(conversationId) {
    const me = myId()
    return items.value
      .filter((it) => it.user_id === me && it.conversation_id === conversationId)
      .map((it) => ({
        id: `out:${it.client_id}`,
        client_id: it.client_id,
        conversation_id: it.conversation_id,
        sender_id: it.user_id,
        text: it.payload.text || null,
        created_at: it.created_at,
        read_at: null,
        attachments: it.preview.attachments || [],
        reactions: [],
        reply_to: it.preview.reply_to || null,
        forwarded_from: null,
        kind: it.payload.task_id ? 'task' : 'text',
        task: it.preview.task || null,
        // 'pending' | 'sending' | 'failed' — пузырь рисует часики или ошибку.
        outbox: it.status,
        outbox_error: it.error || null,
      }))
  }

  /** Поставить сообщение в очередь; отправка — сразу, если есть связь. */
  function enqueue(conversationId, payload, preview = {}) {
    listen()
    items.value.push({
      client_id: newClientId(),
      user_id: myId(),
      conversation_id: conversationId,
      payload: {
        text: payload.text || null,
        attachment_ids: payload.attachment_ids || [],
        reply_to_id: payload.reply_to_id || null,
        task_id: payload.task_id || null,
      },
      preview,
      created_at: new Date().toISOString(),
      status: 'pending',
      error: null,
    })
    flush()
  }

  function find(clientId) {
    return items.value.find((it) => it.client_id === clientId)
  }

  /** Сообщение дошло (HTTP-ответ или сокет-эхо с тем же client_id). */
  function confirm(clientId) {
    if (!clientId) return
    const i = items.value.findIndex((it) => it.client_id === clientId)
    if (i !== -1) items.value.splice(i, 1)
  }

  function retry(clientId) {
    const it = find(clientId)
    if (!it) return
    it.status = 'pending'
    it.error = null
    attempt = 0
    flush()
  }

  function discard(clientId) {
    confirm(clientId)
  }

  function scheduleRetry() {
    clearTimeout(retryTimer)
    const delay = RETRY_STEPS_MS[Math.min(attempt, RETRY_STEPS_MS.length - 1)]
    attempt++
    retryTimer = setTimeout(() => { retryTimer = null; flush() }, delay)
  }

  /** Отправить очередь по порядку. Сеть пропала — ждём и повторяем, не
   *  перескакивая через застрявшее: порядок сообщений важнее скорости. */
  async function flush() {
    listen()
    if (flushing) return
    flushing = true
    clearTimeout(retryTimer)
    retryTimer = null
    try {
      for (;;) {
        const me = myId()
        const it = items.value.find((x) => x.user_id === me && x.status === 'pending')
        if (!it || me == null) {
          attempt = 0
          return
        }
        it.status = 'sending'
        try {
          const msg = await api.sendMessage(it.conversation_id,
            { ...it.payload, client_id: it.client_id }, { timeout: SEND_TIMEOUT_MS })
          confirm(it.client_id)
          const messenger = useMessengerStore()
          messenger.applyIncomingMessage(it.conversation_id, msg, true)
          // Отвечаю в диалог — значит, входящие в нём прочитаны.
          messenger.markRead(it.conversation_id)
          attempt = 0
        } catch (e) {
          if (!find(it.client_id)) continue // успели подтвердить сокетом
          if (isTransient(e)) {
            it.status = 'pending'
            scheduleRetry()
            return
          }
          it.status = 'failed'
          it.error = e?.message || 'Не удалось отправить сообщение'
          try { useNotificationsStore().error(`Сообщение не отправлено: ${it.error}`) } catch { /* без тостов */ }
        }
      }
    } finally {
      flushing = false
    }
  }

  /** Выход из аккаунта: очередь — личные данные, на устройстве не оставляем. */
  function reset() {
    clearTimeout(retryTimer)
    retryTimer = null
    attempt = 0
    items.value = []
  }

  return { items, messagesFor, enqueue, confirm, retry, discard, flush, reset }
})
