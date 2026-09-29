/* Тонкая WS-обёртка realtime-шлюза (gatewaysvc) с socket.io-подобным API.

   Протокол — JSON-кадры {"event": "...", "data": {...}}:
   - первый кадр после открытия — {"event": "auth", "data": {"token"}};
   - сервер отвечает {"event": "_connected"} (диспатчится слушателям как
     'connect') либо {"event": "_error"} и закрывает соединение;
   - дальше события ходят в обе стороны как есть.

   Реконнект встроенный, экспоненциальный: 1 с → 5 с на видимой вкладке и до
   минуты на скрытой (обёртка в трее всё ещё должна получать события, но
   долбиться в сервер каждые 5 с ей незачем). Без сети попыток нет вовсе —
   ждём события `online`; вернулись на вкладку — пробуем сразу.
   Отказ в авторизации (протух access-токен, а шлюз проверяет его только при
   входе) сначала обновляет токен через `refreshAuth`, и лишь потом идёт
   повторная попытка — иначе каждая попытка была бы заведомо отвергнута.
   emit при отсутствии соединения буферизуется (с потолком) и уходит после
   повторной авторизации. Слушатели переживают реконнект.
   Новый токен (refresh, смена активной компании) уходит на живое соединение
   тем же кадром auth — шлюз пересаживает его в комнату новой компании без
   переподключения. */

const RECONNECT_MIN_MS = 1000
const RECONNECT_MAX_MS = 5000
const RECONNECT_HIDDEN_MAX_MS = 60_000
const QUEUE_LIMIT = 200

export class GatewaySocket {
  constructor(url, { auth, refreshAuth } = {}) {
    this.url = url
    this.auth = auth || {}
    this.connected = false

    this._ws = null
    this._listeners = new Map()
    this._queue = []
    this._attempt = 0
    this._reconnectTimer = null
    this._manualClose = false
    this._authFailed = false
    this._refreshAuth = refreshAuth || null
    this._waitingOnline = false
    this._sleeping = false

    this._onOnline = () => {
      if (!this._waitingOnline || this._sleeping) return
      this._waitingOnline = false
      this._attempt = 0
      this._open()
    }
    this._onVisible = () => {
      // Вернулись на вкладку, а попытка отложена надолго — пробуем сейчас.
      if (document.hidden || this.connected || !this._reconnectTimer || this._sleeping) return
      clearTimeout(this._reconnectTimer)
      this._reconnectTimer = null
      this._attempt = 0
      this._open()
    }
    window.addEventListener('online', this._onOnline)
    document.addEventListener('visibilitychange', this._onVisible)

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
      if (this._queue.length > QUEUE_LIMIT) this._queue.shift()
    }
  }

  /** Сменить токен: следующая авторизация пойдёт с ним, а живое соединение
      получит его сразу. */
  setToken(token) {
    this.auth = { token }
    if (token && this.connected && this._ws?.readyState === WebSocket.OPEN) {
      try { this._ws.send(JSON.stringify({ event: 'auth', data: { token } })) } catch {}
    }
  }

  /** Усыпить: закрыть соединение и не переподключаться до wake(). Слушатели
      и токен остаются — пробуждение идёт обычным путём авторизации. */
  sleep() {
    if (this._manualClose || this._sleeping) return
    this._sleeping = true
    clearTimeout(this._reconnectTimer)
    this._reconnectTimer = null
    try { this._ws?.close() } catch {}
  }

  wake() {
    if (this._manualClose || !this._sleeping) return
    this._sleeping = false
    this._attempt = 0
    this._open()
  }

  disconnect() {
    this._manualClose = true
    this._waitingOnline = false
    window.removeEventListener('online', this._onOnline)
    document.removeEventListener('visibilitychange', this._onVisible)
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
    if (this._manualClose || this._sleeping) return
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
        this._authFailed = false
        // Накопленное за время обрыва — после повторной авторизации.
        const queued = this._queue.splice(0)
        for (const f of queued) {
          try { ws.send(f) } catch { this._queue.push(f) }
        }
        this._dispatch('connect')
        return
      }
      if (frame.event === '_error') {
        const code = frame.data?.code || 'AUTH_FAILED'
        if (code === 'AUTH_FAILED') this._authFailed = true
        this._dispatch('connect_error', new Error(code))
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
    if (this._manualClose || this._sleeping || this._reconnectTimer || this._waitingOnline) return
    if (navigator.onLine === false) {
      this._waitingOnline = true
      return
    }
    const cap = document.hidden ? RECONNECT_HIDDEN_MAX_MS : RECONNECT_MAX_MS
    const delay = Math.min(RECONNECT_MIN_MS * 2 ** this._attempt, cap)
    this._attempt += 1
    this._reconnectTimer = setTimeout(async () => {
      if (this._authFailed && this._refreshAuth) {
        try {
          const token = await this._refreshAuth()
          if (token) this.auth = { token }
          this._authFailed = false
        } catch {
          // Сеть или отказ: отказ сервера разлогинит приложение и закроет сокет,
          // сетевую ошибку переживёт следующая попытка.
        }
      }
      this._reconnectTimer = null
      this._open()
    }, delay)
  }
}
