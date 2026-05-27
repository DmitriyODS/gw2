import { defineStore } from 'pinia'
import { getIceServers } from '@/api/calls.js'
import { WebRTCManager } from '@/services/webrtc.js'
import { getSocket } from '@/socket/index.js'
import { useAuthStore } from './auth.js'

/**
 * Store текущего звонка. В каждый момент времени активный звонок один.
 *
 * Жизненный цикл:
 *   idle → outgoing (я позвонил, жду accept) → active → idle
 *   idle → incoming (мне звонят) → active → idle  (или decline → idle)
 *
 * Сами медиа-объекты (RTCPeerConnection, MediaStream) живут в WebRTCManager —
 * чтобы не пытаться засунуть их в reactive (Vue 3 ломает Proxy для DOM-like
 * объектов). В сторе только id, флаги и Map<userId, {fio, stream, audio, video}>.
 */
export const useCallStore = defineStore('call', {
  state: () => ({
    /** 'idle' | 'incoming' | 'outgoing' | 'active' */
    phase: 'idle',
    /** Метаданные текущего звонка с бэка (CallSchema). */
    call: null,
    /** Удалённые потоки и их состояние. Не reactive внутри (stream — raw). */
    remoteStreams: {}, // userId -> { stream, fio, avatar_path, audio, video }
    /** Локальный поток (raw MediaStream). Не реактивен — Vue с ним не дружит. */
    localStream: null,
    /** Локальные настройки. */
    audioEnabled: true,
    videoEnabled: true,
    /** Стартовый медиа-режим звонка ('audio' | 'video'). */
    media: 'video',
    /** UI-флаги. */
    isMinimized: false,
    error: null,
  }),

  getters: {
    isInCall: (s) => s.phase === 'active' || s.phase === 'outgoing',
    isIncoming: (s) => s.phase === 'incoming',
    initiatorId: (s) => s.call?.initiator_id,
    callId: (s) => s.call?.id,
    /** Все участники из метаданных (кроме меня). */
    otherParticipants() {
      const auth = useAuthStore()
      return (this.call?.participants || []).filter(p => p.user_id !== auth.user?.id)
    },
  },

  actions: {
    async _initWebRTC() {
      if (this._rtc) return this._rtc
      let cfg
      try {
        const data = await getIceServers()
        cfg = data?.iceServers
      } catch { /* падать здесь не критично, есть дефолтные STUN */ }
      const rtc = new WebRTCManager({ iceServers: cfg })
      const auth = useAuthStore()
      rtc.setMyUserId(auth.user?.id)

      rtc.addEventListener('local-stream', (e) => {
        this.localStream = e.detail
      })
      rtc.addEventListener('remote-stream', (e) => {
        const { userId, stream } = e.detail
        const existing = this.remoteStreams[userId] || {}
        const part = (this.call?.participants || []).find(p => p.user_id === userId)
        this.remoteStreams = {
          ...this.remoteStreams,
          [userId]: {
            ...existing,
            stream,
            fio: existing.fio || part?.fio || 'Участник',
            avatar_path: existing.avatar_path ?? part?.avatar_path ?? null,
            audio: existing.audio ?? true,
            video: existing.video ?? true,
          },
        }
      })
      rtc.addEventListener('local-signal', (e) => {
        const { toUserId, kind, payload } = e.detail
        const socket = getSocket()
        if (!socket) return
        socket.emit('webrtc:signal', {
          call_id: this.call?.id,
          to_user_id: toUserId,
          kind,
          payload,
        })
      })

      this._rtc = rtc
      return rtc
    },

    /** Я звоню кому-то — отправляем call:start, ждём accepted/declined/ended. */
    async startCall({ userIds, media = 'video', conversationId = null }) {
      if (this.phase !== 'idle') return
      this.media = media
      this.audioEnabled = true
      this.videoEnabled = media === 'video'
      this.phase = 'outgoing'
      this.error = null

      try {
        const rtc = await this._initWebRTC()
        await rtc.start(media)
      } catch (e) {
        this.error = 'Не удалось получить доступ к камере или микрофону'
        this.phase = 'idle'
        throw e
      }

      const socket = getSocket()
      if (!socket) {
        this.error = 'Нет соединения с сервером'
        this.phase = 'idle'
        return
      }
      socket.emit('call:start', {
        user_ids: userIds,
        media,
        conversation_id: conversationId,
      })
    },

    /** Сервер подтвердил, что звонок зарегистрирован (после call:start). */
    handleStarted(callPayload) {
      this.call = callPayload
      // Заполняем remoteStreams placeholder'ами, чтобы UI сразу показал плитки.
      const auth = useAuthStore()
      const others = (callPayload.participants || []).filter(p => p.user_id !== auth.user?.id)
      const next = {}
      for (const p of others) {
        next[p.user_id] = {
          stream: null, fio: p.fio, avatar_path: p.avatar_path, audio: true, video: true,
        }
      }
      this.remoteStreams = next
    },

    /** Мне позвонили. Не запускаем камеру — только пока пользователь не accept'нет. */
    handleIncoming(callPayload) {
      if (this.phase !== 'idle') {
        // Уже в звонке — автоматически отклоняем (бэк не блокирует, но мы не примем)
        return
      }
      this.call = callPayload
      this.media = callPayload.media || 'video'
      this.phase = 'incoming'
    },

    /** Я принимаю входящий. Запускаем камеру и сообщаем серверу. */
    async accept() {
      if (this.phase !== 'incoming') return
      try {
        const rtc = await this._initWebRTC()
        await rtc.start(this.media)
        this.audioEnabled = true
        this.videoEnabled = this.media === 'video'
      } catch (e) {
        this.error = 'Не удалось получить доступ к камере или микрофону'
        this.decline()
        return
      }
      const socket = getSocket()
      if (!socket) return
      socket.emit('call:accept', { call_id: this.call.id })
      // Сервер ответит call:accepted со списком existing_participants —
      // тогда мы начнём offer'ы.
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
      const isInitiator = this.call.initiator_id === useAuthStore().user?.id
      const socket = getSocket()
      if (socket) {
        if (isInitiator && this.phase === 'outgoing') {
          socket.emit('call:end', { call_id: this.call.id })
        } else {
          socket.emit('call:leave', { call_id: this.call.id })
        }
      }
      this.reset()
    },

    /** Серверный accept мой: подключаемся к существующим участникам через WebRTC offer. */
    async handleAccepted({ call_id, existing_participants, call }) {
      if (!this.call || this.call.id !== call_id) return
      this.call = call || this.call
      this.phase = 'active'
      const rtc = await this._initWebRTC()
      // Мы — новый участник; шлём offer всем, кто уже в звонке.
      for (const uid of existing_participants) {
        rtc.createOfferTo(uid).catch(() => {})
      }
    },

    /** К нам кто-то присоединился (не мы accept'нули — кто-то другой). Ждём offer от него. */
    handleParticipantJoined({ user_id }) {
      if (this.phase === 'outgoing') {
        this.phase = 'active'
      }
      // Добавим плитку с placeholder'ом
      if (!this.remoteStreams[user_id]) {
        const part = (this.call?.participants || []).find(p => p.user_id === user_id)
        this.remoteStreams = {
          ...this.remoteStreams,
          [user_id]: {
            stream: null,
            fio: part?.fio || 'Участник',
            avatar_path: part?.avatar_path || null,
            audio: true, video: true,
          },
        }
      }
    },

    handleParticipantLeft({ user_id }) {
      if (this._rtc) this._rtc.removePeer(user_id)
      const next = { ...this.remoteStreams }
      delete next[user_id]
      this.remoteStreams = next
    },

    handleParticipantDeclined({ user_id }) {
      // В p2p звонке это значит, что собеседник нажал «отклонить» — звонок сразу завершится
      // отдельным call:ended. В групповом — просто убираем плитку.
      this.handleParticipantLeft({ user_id })
    },

    handleEnded() {
      this.reset()
    },

    /** WebRTC offer/answer/ice от другого участника. */
    async handleSignal({ from_user_id, kind, payload }) {
      const rtc = await this._initWebRTC()
      if (kind === 'offer') await rtc.handleOffer(from_user_id, payload)
      else if (kind === 'answer') await rtc.handleAnswer(from_user_id, payload)
      else if (kind === 'ice') await rtc.handleRemoteIce(from_user_id, payload)
    },

    handleMediaState({ user_id, audio, video }) {
      const entry = this.remoteStreams[user_id]
      if (!entry) return
      this.remoteStreams = {
        ...this.remoteStreams,
        [user_id]: { ...entry, audio, video },
      }
    },

    toggleMic() {
      this.audioEnabled = !this.audioEnabled
      this._rtc?.setAudioEnabled(this.audioEnabled)
      this._emitMediaState()
    },

    toggleCam() {
      this.videoEnabled = !this.videoEnabled
      this._rtc?.setVideoEnabled(this.videoEnabled)
      this._emitMediaState()
    },

    _emitMediaState() {
      if (!this.call) return
      const socket = getSocket()
      socket?.emit('call:media-state', {
        call_id: this.call.id,
        audio: this.audioEnabled,
        video: this.videoEnabled,
      })
    },

    minimize() { this.isMinimized = true },
    expand() { this.isMinimized = false },

    reset() {
      try { this._rtc?.stop() } catch {}
      this._rtc = null
      this.localStream = null
      this.remoteStreams = {}
      this.phase = 'idle'
      this.call = null
      this.audioEnabled = true
      this.videoEnabled = true
      this.media = 'video'
      this.isMinimized = false
      this.error = null
    },

    handleError({ code, message }) {
      this.error = message || 'Ошибка звонка'
      // Если ошибка случилась до active — сразу выходим
      if (this.phase === 'outgoing' || this.phase === 'incoming') {
        this.reset()
      }
    },
  },
})
