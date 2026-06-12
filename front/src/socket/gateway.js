/* Тонкая WS-обёртка realtime-шлюза (gatewaysvc) с socket.io-подобным API.

   Протокол — JSON-кадры {"event": "...", "data": {...}}:
   - первый кадр после открытия — {"event": "auth", "data": {"token"}};
   - сервер отвечает {"event": "_connected"} (диспатчится слушателям как
     'connect') либо {"event": "_error"} и закрывает соединение;
   - дальше события ходят в обе стороны как есть.

   Реконнект встроенный: экспоненциальная задержка 1с → 5с, бесконечно
   (как прежние опции socket.io-client). emit при отсутствии соединения
   буферизуется и уходит после повторной авторизации. Слушатели переживают
   реконнект — повторная регистрация не нужна. */

const RECONNECT_MIN_MS = 1000
const RECONNECT_MAX_MS = 5000

export class GatewaySocket {
  constructor(url, { auth } = {}) {
    this.url = url
    this.auth = auth || {}
    this.connected = false

    this._ws = null
    this._listeners = new Map()
    this._queue = []
    this._attempt = 0
    this._reconnectTimer = null
    this._manualClose = false

    this._open()
  }

  on(event, handler) {
    if (!this._listeners.has(event)) this._listeners.set(event, new Set())
    this._listeners.get(event).add(handler)
  }

  off(event, handler) {
    this._listeners.get(event)?.delete(handler)
  }

  emit(event, data) {
    const frame = JSON.stringify({ event, data: data ?? null })
    if (this.connected && this._ws?.readyState === WebSocket.OPEN) {
      try { this._ws.send(frame) } catch { this._queue.push(frame) }
    } else {
      this._queue.push(frame)
    }
  }

  disconnect() {
    this._manualClose = true
    clearTimeout(this._reconnectTimer)
    this._queue = []
    try { this._ws?.close() } catch {}
    this._setDisconnected()
  }

  _dispatch(event, data) {
    for (const handler of this._listeners.get(event) ?? []) {
      try { handler(data) } catch (e) { console.error('socket handler error:', e) }
    }
  }

  _open() {
    if (this._manualClose) return
    let ws
    try {
      ws = new WebSocket(this.url)
    } catch (e) {
      this._dispatch('connect_error', e)
      this._scheduleReconnect()
      return
    }
    this._ws = ws

    ws.onopen = () => {
      ws.send(JSON.stringify({ event: 'auth', data: { token: this.auth.token } }))
    }

    ws.onmessage = (msg) => {
      let frame
      try { frame = JSON.parse(msg.data) } catch { return }
      if (!frame?.event) return

      if (frame.event === '_connected') {
        this.connected = true
        this._attempt = 0
        // Накопленное за время обрыва — после повторной авторизации.
        const queued = this._queue.splice(0)
        for (const f of queued) {
          try { ws.send(f) } catch { this._queue.push(f) }
        }
        this._dispatch('connect')
        return
      }
      if (frame.event === '_error') {
        this._dispatch('connect_error', new Error(frame.data?.code || 'AUTH_FAILED'))
        return
      }
      this._dispatch(frame.event, frame.data)
    }

    ws.onclose = () => {
      const wasConnected = this.connected
      this._setDisconnected()
      if (wasConnected) this._dispatch('disconnect')
      this._scheduleReconnect()
    }

    ws.onerror = () => { /* за error всегда следует close */ }
  }

  _setDisconnected() {
    this.connected = false
    this._ws = null
  }

  _scheduleReconnect() {
    if (this._manualClose || this._reconnectTimer) return
    const delay = Math.min(RECONNECT_MIN_MS * 2 ** this._attempt, RECONNECT_MAX_MS)
    this._attempt += 1
    this._reconnectTimer = setTimeout(() => {
      this._reconnectTimer = null
      this._open()
    }, delay)
  }
}
