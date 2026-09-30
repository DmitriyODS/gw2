import { defineStore } from 'pinia'

/* Состояние связи для всего приложения: нет интернета, связь с сервером
   восстанавливается, всё в порядке. Три источника, потому что каждый в
   одиночку врёт: `navigator.onLine` не видит Wi-Fi без интернета, сокет
   молчит до входа в аккаунт, а запросы бывают редкими. */

// Короткий обрыв сокета (сон вкладки, смена сети) не должен мигать плашкой.
const CONNECTING_GRACE_MS = 3000
// Сбой запроса сам по себе «забывается»: иначе на экране без сокета
// (лендинг, вход) плашка висела бы до следующего запроса.
const REQUEST_FAILURE_TTL_MS = 15000
const RESTORED_SHOW_MS = 2500

export const useNetworkStore = defineStore('network', {
  state: () => ({
    online: typeof navigator === 'undefined' || navigator.onLine !== false,
    /** Сокет должен быть поднят (пользователь вошёл). */
    socketActive: false,
    socketConnected: false,
    requestFailed: false,
    /** Сбой держится дольше грейса — показываем «Подключение…». */
    connecting: false,
    /** Связь только что вернулась — короткое подтверждение. */
    restored: false,
    /** Показанный статус — от него считаются переходы («восстановлено»). */
    _shown: typeof navigator === 'undefined' || navigator.onLine !== false ? 'online' : 'offline',
    _timers: { grace: null, failure: null, restored: null },
    _listening: false,
  }),

  getters: {
    /** 'offline' | 'connecting' | 'online' */
    status: (s) => {
      if (!s.online) return 'offline'
      return s.connecting ? 'connecting' : 'online'
    },
  },

  actions: {
    init() {
      if (this._listening || typeof window === 'undefined') return
      this._listening = true
      window.addEventListener('online', () => this._setOnline(true))
      window.addEventListener('offline', () => this._setOnline(false))
    },

    _setOnline(v) {
      this.online = v
      if (v) this.requestFailed = false
      this._recompute()
    },

    setSocketActive(v) {
      this.socketActive = v
      if (!v) this.socketConnected = false
      this._recompute()
    },

    setSocketConnected(v) {
      this.socketConnected = v
      // Живой сокет — сервер точно доступен.
      if (v) this.requestFailed = false
      this._recompute()
    },

    /** Итог HTTP-запроса: дошёл до сервера или упал на сети. */
    reportRequest(ok) {
      clearTimeout(this._timers.failure)
      this.requestFailed = !ok
      if (!ok) {
        this._timers.failure = setTimeout(() => {
          this.requestFailed = false
          this._recompute()
        }, REQUEST_FAILURE_TTL_MS)
      }
      this._recompute()
    },

    _recompute() {
      const t = this._timers
      const before = this._shown
      const trouble = this.online
        && ((this.socketActive && !this.socketConnected) || this.requestFailed)
      if (!trouble) {
        clearTimeout(t.grace)
        t.grace = null
        this.connecting = false
      } else if (before === 'offline') {
        // Сеть вернулась, а сервер ещё не достигнут: сразу «Подключение…»,
        // иначе на время грейса мелькнуло бы «восстановлено».
        this.connecting = true
      } else if (!this.connecting && !t.grace) {
        t.grace = setTimeout(() => {
          t.grace = null
          this.connecting = true
          this._shown = this.status
        }, CONNECTING_GRACE_MS)
      }
      if (before !== 'online' && this.status === 'online') {
        this.restored = true
        clearTimeout(t.restored)
        t.restored = setTimeout(() => { this.restored = false }, RESTORED_SHOW_MS)
      } else if (this.status !== 'online') {
        this.restored = false
      }
      this._shown = this.status
    },
  },
})
