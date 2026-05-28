/**
 * WebRTC manager: одна сущность на звонок, держит mesh RTCPeerConnection'ов
 * (по одному с каждым другим участником). Не зависит от Vue/Pinia — чистый
 * EventEmitter с методами, которые дёргает call-store.
 *
 * Архитектура:
 *  - localStream  — наш getUserMedia (audio + видео-трек, который можно gate'ить через .enabled)
 *  - peers Map<userId, { pc, remoteStream }> — соединения с каждым участником
 *  - на каждом pc: ontrack → emit('remote-stream'), onicecandidate → emit('local-ice')
 *
 * Кто кому шлёт offer:
 *  - При accept'е новый участник получает existing_participants и сам инициирует
 *    offer к каждому. Существующие узлы получают call:participant-joined и ждут offer.
 *  - Это симметрично решает «glare» (offer collision): у каждой пары один заведомый
 *    инициатор — тот, кто пришёл позже.
 *
 * ICE candidate queue:
 *  - addIceCandidate() требует уже установленного remoteDescription. Если ICE
 *    прилетел раньше (что бывает, когда offer/answer и первые кандидаты идут
 *    в одной пачке по сокету), мы складываем его в очередь и применяем после
 *    setRemoteDescription. Без очереди ICE теряются → соединение не поднимается.
 */

const PC_CONFIG_DEFAULT = {
  iceServers: [{ urls: ['stun:stun.l.google.com:19302', 'stun:stun1.l.google.com:19302'] }],
}

export class WebRTCManager extends EventTarget {
  constructor({ iceServers } = {}) {
    super()
    this.config = iceServers ? { iceServers } : PC_CONFIG_DEFAULT
    this.localStream = null
    this.peers = new Map() // userId -> { pc, remoteStream, audio, video, pendingIce[], remoteSet }
    this.myUserId = null
  }

  /** Получаем доступ к камере/микрофону. media: 'audio' | 'video' */
  async start(media = 'video') {
    if (this.localStream) return this.localStream
    const constraints = media === 'audio'
      ? { audio: true, video: false }
      : { audio: true, video: { width: { ideal: 1280 }, height: { ideal: 720 } } }
    try {
      this.localStream = await navigator.mediaDevices.getUserMedia(constraints)
    } catch (e) {
      // Если запросили видео, но камеры нет — пробуем только аудио
      if (media === 'video') {
        try {
          this.localStream = await navigator.mediaDevices.getUserMedia({ audio: true })
        } catch (e2) {
          throw e2
        }
      } else {
        throw e
      }
    }
    // Если peers были созданы раньше localStream (теоретический race) —
    // добавим треки во все существующие соединения.
    for (const entry of this.peers.values()) {
      this._attachLocalTracks(entry.pc)
    }
    this.dispatchEvent(new CustomEvent('local-stream', { detail: this.localStream }))
    return this.localStream
  }

  setMyUserId(id) {
    this.myUserId = id
  }

  _attachLocalTracks(pc) {
    if (!this.localStream) return
    const senders = pc.getSenders()
    for (const track of this.localStream.getTracks()) {
      const already = senders.some(s => s.track === track)
      if (!already) {
        try { pc.addTrack(track, this.localStream) } catch {}
      }
    }
  }

  /** Получаем или создаём peer connection с указанным пользователем. */
  _ensurePeer(remoteUserId) {
    let entry = this.peers.get(remoteUserId)
    if (entry) return entry
    const pc = new RTCPeerConnection(this.config)

    this._attachLocalTracks(pc)

    const remoteStream = new MediaStream()
    pc.ontrack = (event) => {
      // Берём track напрямую — если использовать event.streams[0], при
      // renegotiation возможны лишние «пустые» stream'ы.
      try { remoteStream.addTrack(event.track) } catch {}
      // Каждый ontrack — отдельный track (audio/video приходят независимо).
      // Чтобы Vue гарантированно среагировал на «доехал второй трек» (тот же
      // объект stream — без подмены ссылки watch не сработает), эмитим event
      // с уникальным маркером в detail.
      this.dispatchEvent(new CustomEvent('remote-stream', {
        detail: {
          userId: remoteUserId,
          stream: remoteStream,
          trackKind: event.track?.kind,
          tick: Date.now(),
        },
      }))
      // Track сам сигналит, когда «закончился» — это значит peer выключил
      // дорожку (не renegotiate-вытащил). Чистим, чтобы placeholder показался.
      event.track.onended = () => {
        try { remoteStream.removeTrack(event.track) } catch {}
        this.dispatchEvent(new CustomEvent('remote-stream', {
          detail: {
            userId: remoteUserId,
            stream: remoteStream,
            trackKind: event.track?.kind,
            tick: Date.now(),
            ended: true,
          },
        }))
      }
    }

    pc.onicecandidate = (event) => {
      if (event.candidate) {
        this.dispatchEvent(new CustomEvent('local-signal', {
          detail: {
            toUserId: remoteUserId,
            kind: 'ice',
            payload: event.candidate.toJSON(),
          },
        }))
      }
    }

    pc.onconnectionstatechange = () => {
      const st = pc.connectionState
      if (st === 'failed' || st === 'disconnected' || st === 'closed') {
        this.dispatchEvent(new CustomEvent('peer-state', {
          detail: { userId: remoteUserId, state: st },
        }))
      }
      if (st === 'connected') {
        this.dispatchEvent(new CustomEvent('peer-state', {
          detail: { userId: remoteUserId, state: 'connected' },
        }))
      }
    }

    entry = { pc, remoteStream, audio: true, video: true, pendingIce: [], remoteSet: false }
    this.peers.set(remoteUserId, entry)
    return entry
  }

  /** Инициируем offer к существующему участнику звонка. */
  async createOfferTo(remoteUserId) {
    const { pc } = this._ensurePeer(remoteUserId)
    const offer = await pc.createOffer()
    await pc.setLocalDescription(offer)
    this.dispatchEvent(new CustomEvent('local-signal', {
      detail: { toUserId: remoteUserId, kind: 'offer', payload: offer },
    }))
  }

  /** Принимаем offer и отдаём answer. */
  async handleOffer(fromUserId, offer) {
    const entry = this._ensurePeer(fromUserId)
    await entry.pc.setRemoteDescription(new RTCSessionDescription(offer))
    entry.remoteSet = true
    await this._flushPendingIce(entry)
    const answer = await entry.pc.createAnswer()
    await entry.pc.setLocalDescription(answer)
    this.dispatchEvent(new CustomEvent('local-signal', {
      detail: { toUserId: fromUserId, kind: 'answer', payload: answer },
    }))
  }

  async handleAnswer(fromUserId, answer) {
    const entry = this.peers.get(fromUserId)
    if (!entry) return
    await entry.pc.setRemoteDescription(new RTCSessionDescription(answer))
    entry.remoteSet = true
    await this._flushPendingIce(entry)
  }

  async handleRemoteIce(fromUserId, candidate) {
    const entry = this._ensurePeer(fromUserId)
    // Пока не установили remoteDescription — кандидаты копим. addIceCandidate
    // до setRemoteDescription выкидывает InvalidStateError и кандидат теряется.
    if (!entry.remoteSet) {
      entry.pendingIce.push(candidate)
      return
    }
    try {
      await entry.pc.addIceCandidate(new RTCIceCandidate(candidate))
    } catch {
      // Дубликаты/уже неактуальные кандидаты — не критично.
    }
  }

  async _flushPendingIce(entry) {
    if (!entry.pendingIce.length) return
    const queue = entry.pendingIce.splice(0)
    for (const cand of queue) {
      try {
        await entry.pc.addIceCandidate(new RTCIceCandidate(cand))
      } catch {}
    }
  }

  removePeer(userId) {
    const entry = this.peers.get(userId)
    if (!entry) return
    try { entry.pc.close() } catch {}
    this.peers.delete(userId)
  }

  /** Локальный mute/unmute микрофона. */
  setAudioEnabled(enabled) {
    if (!this.localStream) return
    for (const track of this.localStream.getAudioTracks()) {
      track.enabled = enabled
    }
  }

  /** Локальное вкл/выкл камеры. */
  setVideoEnabled(enabled) {
    if (!this.localStream) return
    for (const track of this.localStream.getVideoTracks()) {
      track.enabled = enabled
    }
  }

  /** Завершаем звонок: закрываем peer'ы и отпускаем камеру/микрофон. */
  stop() {
    for (const { pc } of this.peers.values()) {
      try { pc.close() } catch {}
    }
    this.peers.clear()
    if (this.localStream) {
      for (const track of this.localStream.getTracks()) {
        try { track.stop() } catch {}
      }
      this.localStream = null
    }
  }
}
