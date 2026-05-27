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
 */

const PC_CONFIG_DEFAULT = {
  iceServers: [{ urls: ['stun:stun.l.google.com:19302', 'stun:stun1.l.google.com:19302'] }],
}

export class WebRTCManager extends EventTarget {
  constructor({ iceServers } = {}) {
    super()
    this.config = iceServers ? { iceServers } : PC_CONFIG_DEFAULT
    this.localStream = null
    this.peers = new Map() // userId -> { pc, remoteStream, audio, video }
    this.myUserId = null
  }

  /** Получаем доступ к камере/микрофону. media: 'audio' | 'video' */
  async start(media = 'video') {
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
    this.dispatchEvent(new CustomEvent('local-stream', { detail: this.localStream }))
    return this.localStream
  }

  setMyUserId(id) {
    this.myUserId = id
  }

  /** Получаем или создаём peer connection с указанным пользователем. */
  _ensurePeer(remoteUserId) {
    let entry = this.peers.get(remoteUserId)
    if (entry) return entry
    const pc = new RTCPeerConnection(this.config)

    if (this.localStream) {
      for (const track of this.localStream.getTracks()) {
        pc.addTrack(track, this.localStream)
      }
    }

    const remoteStream = new MediaStream()
    pc.ontrack = (event) => {
      // Берём track напрямую — если использовать event.streams[0], при
      // renegotiation возможны лишние «пустые» stream'ы.
      remoteStream.addTrack(event.track)
      this.dispatchEvent(new CustomEvent('remote-stream', {
        detail: { userId: remoteUserId, stream: remoteStream },
      }))
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
      if (pc.connectionState === 'failed' || pc.connectionState === 'disconnected') {
        this.dispatchEvent(new CustomEvent('peer-failed', { detail: { userId: remoteUserId } }))
      }
    }

    entry = { pc, remoteStream, audio: true, video: true }
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
    const { pc } = this._ensurePeer(fromUserId)
    await pc.setRemoteDescription(new RTCSessionDescription(offer))
    const answer = await pc.createAnswer()
    await pc.setLocalDescription(answer)
    this.dispatchEvent(new CustomEvent('local-signal', {
      detail: { toUserId: fromUserId, kind: 'answer', payload: answer },
    }))
  }

  async handleAnswer(fromUserId, answer) {
    const entry = this.peers.get(fromUserId)
    if (!entry) return
    await entry.pc.setRemoteDescription(new RTCSessionDescription(answer))
  }

  async handleRemoteIce(fromUserId, candidate) {
    const entry = this.peers.get(fromUserId)
    if (!entry) return
    try {
      await entry.pc.addIceCandidate(new RTCIceCandidate(candidate))
    } catch (e) {
      // На раннем этапе addIceCandidate может прилететь до setRemoteDescription —
      // игнорируем; «горячее» соединение восстановится после полной negotiation.
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
