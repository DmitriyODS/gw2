import { defineStore } from 'pinia'
import { getActiveCall, getCallToken, joinCallByCode } from '@/api/calls.js'
import { callRoom, parseParticipantMetadata } from '@/services/livekit.js'
import { getSocket } from '@/socket/index.js'
import { useAuthStore } from './auth.js'
import { useNotificationsStore } from './notifications.js'
import { useMessengerStore } from './messenger.js'
import { requestNotificationPermission } from '@/utils/systemNotify.js'

/**
 * Store текущего звонка. В каждый момент времени активный звонок один.
 *
 * Жизненный цикл:
 *   idle → outgoing (я позвонил, жду пока кто-то войдёт в комнату) → active → idle
 *   idle → incoming (мне звонят) → active → idle  (или decline → idle)
 *   idle → active (вход по ссылке-приглашению, в т. ч. гостем)
 *
 * Медиа целиком на LiveKit: стор хранит только лёгкие реактивные снимки
 * участников (имя/флаги/«тик» для пере-attach плиток), сами Room/Track живут
 * в services/livekit.js вне Vue-реактивности. Ринг-фаза (invite/accept/
 * decline) по-прежнему ходит через Socket.IO; гость по ссылке сокет не
 * использует вовсе — для него звонок начинается и заканчивается комнатой.
 */
export const useCallStore = defineStore('call', {
  state: () => ({
    /** 'idle' | 'incoming' | 'outgoing' | 'active' */
    phase: 'idle',
    /** Метаданные текущего звонка с бэка (CallSchema). Для гостя — усечённые. */
    call: null,
    /** Гостевой режим (вход по ссылке без аккаунта). */
    guest: false,
    /** Имя гостя (для подписи «Вы» и собственных сообщений чата). */
    guestName: null,
    /**
     * Снимок участников комнаты + плейсхолдеры ещё не вошедших приглашённых.
     * identity → { identity, name, userId, avatarPath, guest, audio, video,
     *              screen, speaking, pending, tick }
     */
    participants: {},
    /** Бамп для пере-attach локальных плиток (локальные треки сменились). */
    localTick: 0,
    /** Локальные настройки. */
    audioEnabled: true,
    videoEnabled: true,
    screenEnabled: false,
    /** Стартовый медиа-режим звонка ('audio' | 'video'). */
    media: 'video',
    /** UI-флаги. */
    isMinimized: false,
    /** Боковая панель: null | 'participants' | 'chat'. */
    sidePanel: null,
    error: null,
    /** Чат звонка (data-канал LiveKit, живёт только пока идёт звонок). */
    chatMessages: [],
    chatUnread: 0,
    /** Звонок, к которому можно вернуться после перезагрузки страницы. */
    rejoinCall: null,
  }),

  getters: {
    isInCall: (s) => s.phase === 'active' || s.phase === 'outgoing',
    isIncoming: (s) => s.phase === 'incoming',
    initiatorId: (s) => s.call?.initiator_id,
    callId: (s) => s.call?.id,
    participantList: (s) => Object.values(s.participants),
    /** Все в комнате, включая меня. */
    participantCount: (s) =>
      1 + Object.values(s.participants).filter(p => !p.pending).length,
    /** Ссылка-приглашение в текущий звонок. */
    inviteLink: (s) => s.call?.share_code
      ? `${window.location.origin}/call/${s.call.share_code}`
      : null,
    myIdentity: (s) => {
      if (s.guest) return callRoom.localIdentity
      const auth = useAuthStore()
      return auth.user ? `u${auth.user.id}` : null
    },
  },

  actions: {
    /** Подписка на события LiveKit-комнаты (один раз на сессию стора). */
    _bindRoomEvents() {
      if (this._roomBound) return
      this._roomBound = true
      this._declined = new Set()

      callRoom.addEventListener('connected', () => this.resyncParticipants())
      callRoom.addEventListener('participant-joined', () => {
        // Кто-то вошёл в комнату — дозвон состоялся.
        if (this.phase === 'outgoing') {
          this.phase = 'active'
          this._clearOutgoingTimeout()
        }
        this.resyncParticipants()
      })
      callRoom.addEventListener('participant-left', () => this.resyncParticipants())
      callRoom.addEventListener('track-changed', (e) => {
        if (e.detail?.local) {
          this.localTick = Date.now()
          this.screenEnabled = callRoom.mediaState(callRoom.localIdentity).screen
        }
        this.resyncParticipants()
      })
      callRoom.addEventListener('speakers', (e) => {
        const speaking = new Set(e.detail.identities)
        const next = { ...this.participants }
        for (const id of Object.keys(next)) {
          next[id] = { ...next[id], speaking: speaking.has(id) }
        }
        this.participants = next
      })
      callRoom.addEventListener('chat', (e) => {
        const { identity, name, text, ts } = e.detail
        if (!text) return
        this._chatSeq = (this._chatSeq || 0) + 1
        this.chatMessages.push({
          id: this._chatSeq, identity, name, text, ts, own: false,
        })
        if (this.sidePanel !== 'chat') this.chatUnread++
      })
      callRoom.addEventListener('media-error', (e) => {
        const msg = e.detail.kind === 'video'
          ? 'Не удалось включить камеру. Проверьте разрешение в браузере.'
          : 'Не удалось включить микрофон. Проверьте разрешение в браузере.'
        this.error = msg
        try { useNotificationsStore().warn(msg) } catch {}
        if (e.detail.kind === 'audio') this.audioEnabled = false
        else this.videoEnabled = false
      })
      callRoom.addEventListener('disconnected', (e) => {
        // Комнату закрыл сервер (звонок завершён/нас удалили) — выходим.
        // Свой собственный disconnect() сюда не попадает (слушатели сняты).
        if (this.phase !== 'idle' && e.detail?.byServer) {
          this.reset()
        }
      })
    },

    /** Пересобрать реактивный снимок участников из комнаты LiveKit. */
    resyncParticipants() {
      const auth = useAuthStore()
      const myId = this.guest ? null : auth.user?.id
      const next = {}

      for (const p of callRoom.remoteParticipants()) {
        const meta = parseParticipantMetadata(p)
        const st = callRoom.mediaState(p.identity)
        next[p.identity] = {
          identity: p.identity,
          name: p.name || 'Участник',
          userId: meta.user_id ?? null,
          avatarPath: meta.avatar_path ?? null,
          guest: !!meta.guest,
          audio: st.audio,
          video: st.video,
          screen: st.screen,
          speaking: this.participants[p.identity]?.speaking || false,
          pending: false,
          tick: Date.now(),
        }
      }

      // Плейсхолдеры приглашённых, которые ещё не вошли в комнату.
      for (const part of this.call?.participants || []) {
        if (part.user_id === myId) continue
        if (part.declined || part.left_at) continue
        if (this._declined?.has(part.user_id)) continue
        const ident = `u${part.user_id}`
        if (next[ident]) continue
        next[ident] = {
          identity: ident,
          name: part.fio || 'Участник',
          userId: part.user_id,
          avatarPath: part.avatar_path ?? null,
          guest: false,
          audio: false, video: false, screen: false,
          speaking: false,
          pending: true,
          tick: 0,
        }
      }

      this.participants = next
    },

    /** Подключение к комнате LiveKit по выданному бэком токену. */
    async _connectRoom(livekit) {
      this._bindRoomEvents()
      try {
        await callRoom.connect({
          url: livekit.url,
          token: livekit.token,
          audio: this.audioEnabled,
          video: this.media === 'video' && this.videoEnabled,
        })
      } catch (e) {
        const msg = 'Не удалось подключиться к серверу звонков'
        this.error = msg
        try { useNotificationsStore().warn(msg) } catch {}
        this.hangup()
        throw e
      }
      this.resyncParticipants()
    },

    /** Я звоню кому-то — отправляем call:start, ждём call:started с токеном.
     *  videoOff — звонок остаётся видео-звонком, но своя камера при старте
     *  выключена (конференция «для одного»: видео включают по желанию). */
    async startCall({ userIds, media = 'video', videoOff = false }) {
      if (this.phase !== 'idle') return
      this.media = media
      this.audioEnabled = true
      this.videoEnabled = media === 'video' && !videoOff
      this.phase = 'outgoing'
      this.error = null

      // Жест клика «позвонить» — самый надёжный момент попросить разрешение
      // на OS-уведомления (нужны собеседнику; здесь это безопасный no-op,
      // если уже выдано/отказано).
      requestNotificationPermission().catch(() => {})

      const socket = getSocket()
      if (!socket) {
        this.error = 'Нет соединения с сервером'
        this.phase = 'idle'
        return
      }
      socket.emit('call:start', { user_ids: userIds, media })
      this._armOutgoingTimeout()
    },

    /** Если за разумное время никто не вошёл — завершаем «не дозвонился». */
    _armOutgoingTimeout() {
      this._clearOutgoingTimeout()
      this._outgoingTimer = setTimeout(() => {
        if (this.phase === 'outgoing') {
          try { useNotificationsStore().info('Абонент не отвечает') } catch {}
          this.hangup()
        }
      }, 45000)
    },

    _clearOutgoingTimeout() {
      if (this._outgoingTimer) {
        clearTimeout(this._outgoingTimer)
        this._outgoingTimer = null
      }
    },

    /** Сервер зарегистрировал звонок и выдал мне (инициатору) токен комнаты. */
    async handleStarted({ call, livekit }) {
      if (this.phase !== 'outgoing') return
      this.call = call
      // Пустой звонок (без приглашённых) сразу активен — дозваниваться некому,
      // людей зовут уже из звонка (person_add / ссылка-приглашение).
      const hasInvitees = (call?.participants || []).some(p => p.role === 'invitee')
      if (!hasInvitees) {
        this._clearOutgoingTimeout()
        this.phase = 'active'
      }
      this.resyncParticipants() // плейсхолдеры приглашённых
      await this._connectRoom(livekit)
    },

    /** Мне позвонили. Камеру не трогаем, пока пользователь не примет. */
    handleIncoming(callPayload) {
      if (this.phase !== 'idle') {
        // Я уже в звонке — отказываем сразу автоматически, чтобы у звонящего
        // звонок не висел в ringing до таймаута.
        const socket = getSocket()
        socket?.emit('call:decline', { call_id: callPayload.id })
        return
      }
      this.call = callPayload
      this.media = callPayload.media || 'video'
      this.phase = 'incoming'
    },

    /** Я принимаю входящий: сервер в ответ пришлёт call:accepted с токеном. */
    accept() {
      if (this.phase !== 'incoming') return
      const socket = getSocket()
      if (!socket) {
        this.error = 'Нет соединения с сервером'
        this.reset()
        return
      }
      this.audioEnabled = true
      this.videoEnabled = this.media === 'video'
      socket.emit('call:accept', { call_id: this.call.id })
    },

    /** Сервер подтвердил accept и выдал токен — подключаемся к комнате. */
    async handleAccepted({ call_id, call, livekit }) {
      if (!this.call || this.call.id !== call_id) {
        // accept мог уйти с другой вкладки — просто синхронизируемся.
        this.call = call
        this.media = call?.media || this.media
      } else if (call) {
        this.call = call
      }
      this._clearOutgoingTimeout()
      this.rejoinCall = null
      this.phase = 'active'
      this.resyncParticipants()
      if (livekit?.token) {
        await this._connectRoom(livekit)
      }
    },

    /** Присоединение к уже идущему звонку из чата (плашка kind='call'). */
    async joinExistingCall(callPayload) {
      const callId = callPayload?.id || callPayload?.call_id || callPayload
      if (!callId) return
      this.rejoinCall = null
      // Уже в этом звонке — просто разворачиваем окно.
      if (this.call?.id === callId && this.phase !== 'idle') {
        this.expand()
        return
      }
      if (this.phase !== 'idle') return

      let data
      try {
        data = await getCallToken(callId)
      } catch (e) {
        this.handleError({ code: e?.code || 'NOT_IN_CALL', message: e?.message })
        return
      }
      this.call = data.call
      this.media = data.call?.media || 'video'
      this.audioEnabled = true
      this.videoEnabled = this.media === 'video'
      this.phase = 'active'
      this.error = null
      await this._connectRoom(data.livekit)
    },

    /** Гость (или сотрудник) входит по ссылке-приглашению /call/<code>. */
    async joinAsGuest({ code, name = null }) {
      if (this.phase !== 'idle') return
      const data = await joinCallByCode(code, { name }) // ошибки ловит вьюха
      this.guest = data.guest
      this.guestName = data.guest ? name : null
      this.call = data.call
      this.media = data.call?.media || 'video'
      this.audioEnabled = true
      this.videoEnabled = this.media === 'video'
      this.phase = 'active'
      this.error = null
      await this._connectRoom(data.livekit)
    },

    /** Синхронизировать состояние звонка с сервером (загрузка страницы и
     *  каждый reconnect сокета). Лечит зависший phase и предлагает вернуться
     *  в живой звонок после перезагрузки. */
    async checkRejoin() {
      let call
      try {
        ({ call } = await getActiveCall())
      } catch { return } // сервер недоступен — не трогаем состояние
      const live = (call && call.status !== 'ended' && call.status !== 'missed') ? call : null

      if (this.phase !== 'idle') {
        if (!live || live.id !== this.call?.id) {
          this.reset()
          if (live) this.rejoinCall = live
        }
        return
      }
      if (!live) {
        // Сервер не видит за мной живого звонка — баннер «Вернуться» устарел
        // (call:ended мог потеряться, пока не было соединения).
        this.rejoinCall = null
        return
      }
      if (this.rejoinCall?.id !== live.id) {
        this.rejoinCall = live
      }
    },

    dismissRejoin() {
      const call = this.rejoinCall
      this.rejoinCall = null
      // Явно «не возвращаюсь» — выходим из звонка на сервере.
      if (call) {
        const socket = getSocket()
        socket?.emit('call:leave', { call_id: call.id })
      }
    },

    /** «Вернуться к звонку» после перезагрузки: берём свежий токен и входим. */
    async confirmRejoin() {
      const call = this.rejoinCall
      this.rejoinCall = null
      if (!call) return
      await this.joinExistingCall(call)
    },

    /** Я отклоняю входящий. */
    decline() {
      if (!this.call) {
        this.reset()
        return
      }
      const socket = getSocket()
      socket?.emit('call:decline', { call_id: this.call.id })
      this.reset()
    },

    /** Я отменяю исходящий или ухожу из активного. */
    hangup() {
      if (!this.call) {
        this.reset()
        return
      }
      if (!this.guest) {
        const isInitiator = this.call.initiator_id === useAuthStore().user?.id
        const socket = getSocket()
        if (socket) {
          // Отмена недозвонившегося исходящего — завершение для всех; иначе
          // просто выходим сами (звонок продолжается без нас, LiveKit пришлёт
          // participant_left вебхуком).
          if (isInitiator && this.phase === 'outgoing') {
            socket.emit('call:end', { call_id: this.call.id })
          } else {
            socket.emit('call:leave', { call_id: this.call.id })
          }
        }
      }
      this.reset()
    },

    /** Я приглашаю ещё людей в текущий звонок. */
    inviteToCall(userIds) {
      if (!this.call || !userIds?.length) return
      const socket = getSocket()
      socket?.emit('call:invite', { call_id: this.call.id, user_ids: userIds })
    },

    /** Сервер сообщил, что в звонок позвали новых людей — обновляем
     *  метаданные, resync добавит плитки-плейсхолдеры. */
    handleInvited({ call_id, call }) {
      if (!this.call || this.call.id !== call_id) return
      if (call) this.call = call
      this.resyncParticipants()
    },

    handleParticipantDeclined({ user_id }) {
      // Убираем плитку-плейсхолдер; если p2p — следом придёт call:ended.
      this._declined?.add(user_id)
      this.resyncParticipants()
    },

    handleEnded({ call_id } = {}) {
      // Вебхук room_finished доезжает через Redis-мост асинхронно — запоздавшее
      // call:ended от предыдущего звонка не должно сбросить уже новый.
      if (call_id && this.rejoinCall?.id === call_id) this.rejoinCall = null
      if (call_id && this.call?.id !== call_id) return
      this.reset()
    },

    async toggleMic() {
      this.audioEnabled = !this.audioEnabled
      try {
        await callRoom.setMicEnabled(this.audioEnabled)
      } catch {
        this.audioEnabled = !this.audioEnabled
        try { useNotificationsStore().warn('Не удалось переключить микрофон') } catch {}
      }
    },

    async toggleCam() {
      this.videoEnabled = !this.videoEnabled
      try {
        await callRoom.setCamEnabled(this.videoEnabled)
      } catch {
        this.videoEnabled = !this.videoEnabled
        try { useNotificationsStore().warn('Не удалось переключить камеру') } catch {}
      }
      this.localTick = Date.now()
    },

    async toggleScreenShare() {
      const target = !this.screenEnabled
      try {
        await callRoom.setScreenShareEnabled(target)
        this.screenEnabled = target
      } catch {
        // Пользователь отменил выбор экрана — не ошибка.
      }
      this.localTick = Date.now()
    },

    /** Сообщение в чат звонка (data-канал, к собеседникам и гостям). */
    sendChat(text) {
      const value = (text || '').trim()
      if (!value || !callRoom.connected) return
      callRoom.sendChat(value)
      const auth = useAuthStore()
      this._chatSeq = (this._chatSeq || 0) + 1
      this.chatMessages.push({
        id: this._chatSeq,
        identity: this.myIdentity,
        name: this.guest ? (this.guestName || 'Вы') : (auth.user?.fio || 'Вы'),
        text: value,
        ts: Date.now(),
        own: true,
      })
    },

    openPanel(name) {
      this.sidePanel = name
      if (name === 'chat') this.chatUnread = 0
    },

    togglePanel(name) {
      if (this.sidePanel === name) this.sidePanel = null
      else this.openPanel(name)
    },

    minimize() { this.isMinimized = true },
    expand() { this.isMinimized = false },

    reset() {
      this._clearOutgoingTimeout()
      callRoom.disconnect().catch(() => {})
      this.participants = {}
      this.localTick = 0
      this.phase = 'idle'
      this.call = null
      this.guest = false
      this.guestName = null
      this.audioEnabled = true
      this.videoEnabled = true
      this.screenEnabled = false
      this.media = 'video'
      this.isMinimized = false
      this.sidePanel = null
      this.chatMessages = []
      this.chatUnread = 0
      this.error = null
      this.rejoinCall = null
      this._declined?.clear?.()
    },

    handleError({ code, message }) {
      // Звонок уже завершён, а мы пытались принять/присоединиться по
      // устаревшей плашке — сбрасываем и обновляем переписку, чтобы плашка
      // перерисовалась в «завершён».
      const isStale = code === 'NOT_INVITED' || code === 'NOT_IN_CALL' || code === 'CALL_NOT_FOUND'
      const text = isStale
        ? 'Звонок уже завершён'
        : (message || 'Ошибка звонка')
      this.error = text
      try { useNotificationsStore().warn(text) } catch {}
      if (isStale) {
        this.reset()
        try {
          const m = useMessengerStore()
          if (m.activeConversationId) m.fetchMessages(m.activeConversationId)
        } catch { /* мессенджер не инициализирован */ }
        return
      }
      if (this.phase === 'outgoing' || this.phase === 'incoming') {
        this.reset()
      }
    },
  },
})
