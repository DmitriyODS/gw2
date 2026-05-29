/**
 * WebRTC manager: одна сущность на звонок, держит mesh RTCPeerConnection'ов
 * (по одному с каждым другим участником). Не зависит от Vue/Pinia — чистый
 * EventTarget с методами, которые дёргает call-store.
 *
 * Модель сигналинга — детерминированная, без glare:
 *
 *  - На каждую пару участников ровно ОДИН offerer. Им всегда выступает
 *    «новенький»: тот, кто только что вошёл в звонок (accept/rejoin), получает
 *    список уже подключённых и шлёт им offer — connectTo(uid, {offerer:true}).
 *    Уже находящиеся в звонке узнают о нём через participant-joined и лишь
 *    ОТВЕЧАЮТ — connectTo(uid) без offerer. Так на пару приходится один offer,
 *    нет коллизий и лишних renegotiation (именно из-за них раньше соединение
 *    вставало, но медиа не шло).
 *  - На случай редкой настоящей коллизии (двое вошли одновременно и видят друг
 *    друга в existing) оставлен страховочный perfect-negotiation: роль
 *    polite/impolite по сравнению user_id — «вежливый» при коллизии откатывает
 *    свой offer и принимает чужой, «невежливый» игнорирует чужой.
 *  - ICE-кандидаты, прилетевшие до setRemoteDescription, копим в очереди и
 *    применяем после (иначе addIceCandidate бросает InvalidStateError).
 *  - На `iceConnectionState === 'failed'` делаем restartIce() — соединение
 *    пере-устанавливается, а не висит вечно.
 *
 * Сигналинг: наружу через событие 'local-signal' летят два вида сообщений —
 * kind:'sdp' (offer/answer как RTCSessionDescription) и kind:'ice'. Сервер
 * маршрутизирует их «как есть», не разбирая содержимое.
 */

const PC_CONFIG_DEFAULT = {
  iceServers: [{ urls: ['stun:stun.l.google.com:19302', 'stun:stun1.l.google.com:19302'] }],
}

export class WebRTCManager extends EventTarget {
  constructor({ iceServers } = {}) {
    super()
    this.config = (iceServers && iceServers.length) ? { iceServers } : PC_CONFIG_DEFAULT
    this.localStream = null
    this.peers = new Map() // userId -> peer entry
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
      // Если запросили видео, но камеры нет — пробуем только аудио.
      if (media === 'video') {
        this.localStream = await navigator.mediaDevices.getUserMedia({ audio: true })
      } else {
        throw e
      }
    }
    // Если peers были созданы раньше localStream — добавим треки во все.
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
        try { pc.addTrack(track, this.localStream) } catch { /* уже добавлен */ }
      }
    }
  }

  /** Начать соединение с участником: создаём peer и вешаем локальные треки.
   *  Если offerer=true — мы инициируем offer (мы «новенький» в этой паре);
   *  иначе только готовимся отвечать на чужой offer. Идемпотентно. */
  connectTo(remoteUserId, { offerer = false } = {}) {
    const existed = this.peers.has(remoteUserId)
    const entry = this._ensurePeer(remoteUserId)
    if (offerer) {
      // Помечаем себя инициатором. Для СВЕЖЕГО peer offer уйдёт сам: добавление
      // локальных треков в _ensurePeer поднимет negotiationneeded, а тот увидит
      // флаг (он ставится синхронно, до асинхронного negotiationneeded). Если
      // peer уже существовал (треки добавлены раньше — negotiationneeded
      // больше не выстрелит) — инициируем offer вручную.
      entry.isOfferer = true
      if (existed && entry.pc.signalingState === 'stable') {
        this._makeOffer(remoteUserId, entry)
      }
    }
    return entry
  }

  _ensurePeer(remoteUserId) {
    let entry = this.peers.get(remoteUserId)
    if (entry) return entry

    const pc = new RTCPeerConnection(this.config)
    // Politeness детерминированно: у кого id меньше — polite (уступает при
    // коллизии). Обе стороны вычисляют одинаково. Нужно только как страховка
    // от редкого настоящего glare (двое вошли одновременно).
    const polite = Number(this.myUserId) < Number(remoteUserId)
    entry = {
      pc, remoteStream: new MediaStream(), polite,
      isOfferer: false, makingOffer: false, ignoreOffer: false,
      pendingIce: [], remoteSet: false,
    }
    this.peers.set(remoteUserId, entry)

    pc.ontrack = (event) => {
      try { entry.remoteStream.addTrack(event.track) } catch { /* дубликат */ }
      this._emitRemoteStream(remoteUserId, entry, event.track?.kind)
      event.track.onended = () => {
        try { entry.remoteStream.removeTrack(event.track) } catch { /* нет трека */ }
        this._emitRemoteStream(remoteUserId, entry, event.track?.kind, true)
      }
    }

    pc.onicecandidate = (event) => {
      if (event.candidate) {
        this.dispatchEvent(new CustomEvent('local-signal', {
          detail: { toUserId: remoteUserId, kind: 'ice', payload: event.candidate.toJSON() },
        }))
      }
    }

    // Offer шлёт ТОЛЬКО назначенный инициатор пары (offerer). Отвечающая
    // сторона ждёт чужой offer — так нет glare и лишних renegotiation.
    pc.onnegotiationneeded = () => {
      if (!entry.isOfferer) return
      this._makeOffer(remoteUserId, entry)
    }

    pc.oniceconnectionstatechange = () => {
      if (pc.iceConnectionState === 'failed') {
        // Сеть «провалилась» (часто — STUN не пробил NAT и нужен TURN).
        // Пробуем перезапустить ICE, а не висеть вечно.
        try { pc.restartIce() } catch { /* старый браузер */ }
      }
    }

    pc.onconnectionstatechange = () => {
      this.dispatchEvent(new CustomEvent('peer-state', {
        detail: { userId: remoteUserId, state: pc.connectionState },
      }))
    }

    this._attachLocalTracks(pc) // → negotiationneeded → offer
    return entry
  }

  async _makeOffer(remoteUserId, entry) {
    if (entry.makingOffer) return
    try {
      entry.makingOffer = true
      await entry.pc.setLocalDescription() // implicit-offer
      this.dispatchEvent(new CustomEvent('local-signal', {
        detail: { toUserId: remoteUserId, kind: 'sdp', payload: entry.pc.localDescription },
      }))
    } catch (e) {
      console.warn('[gw2 webrtc] makeOffer error', e)
    } finally {
      entry.makingOffer = false
    }
  }

  _emitRemoteStream(userId, entry, trackKind, ended = false) {
    this.dispatchEvent(new CustomEvent('remote-stream', {
      detail: { userId, stream: entry.remoteStream, trackKind, tick: Date.now(), ended },
    }))
  }

  /** Принять SDP (offer или answer) с учётом politeness. */
  async handleDescription(fromUserId, description) {
    const entry = this._ensurePeer(fromUserId)
    const pc = entry.pc
    const offerCollision = description.type === 'offer'
      && (entry.makingOffer || pc.signalingState !== 'stable')

    entry.ignoreOffer = !entry.polite && offerCollision
    if (entry.ignoreOffer) {
      // Невежливый peer при коллизии игнорирует чужой offer — продолжит свой.
      return
    }

    await pc.setRemoteDescription(description)
    entry.remoteSet = true
    await this._flushPendingIce(entry)

    if (description.type === 'offer') {
      await pc.setLocalDescription() // implicit-answer
      this.dispatchEvent(new CustomEvent('local-signal', {
        detail: { toUserId: fromUserId, kind: 'sdp', payload: pc.localDescription },
      }))
    }
  }

  async handleRemoteIce(fromUserId, candidate) {
    const entry = this._ensurePeer(fromUserId)
    if (!entry.remoteSet) {
      // remoteDescription ещё не установлен — копим кандидата.
      entry.pendingIce.push(candidate)
      return
    }
    try {
      await entry.pc.addIceCandidate(new RTCIceCandidate(candidate))
    } catch (e) {
      // Кандидат от offer'а, который мы проигнорировали (glare) — не критично.
      if (!entry.ignoreOffer) console.debug('[webrtc] addIceCandidate', e?.name)
    }
  }

  async _flushPendingIce(entry) {
    if (!entry.pendingIce.length) return
    const queue = entry.pendingIce.splice(0)
    for (const cand of queue) {
      try { await entry.pc.addIceCandidate(new RTCIceCandidate(cand)) } catch { /* устарел */ }
    }
  }

  removePeer(userId) {
    const entry = this.peers.get(userId)
    if (!entry) return
    try { entry.pc.close() } catch { /* уже закрыт */ }
    this.peers.delete(userId)
  }

  /** Локальный mute/unmute микрофона. */
  setAudioEnabled(enabled) {
    if (!this.localStream) return
    for (const track of this.localStream.getAudioTracks()) track.enabled = enabled
  }

  /** Локальное вкл/выкл камеры. */
  setVideoEnabled(enabled) {
    if (!this.localStream) return
    for (const track of this.localStream.getVideoTracks()) track.enabled = enabled
  }

  /** Завершаем звонок: закрываем peer'ы и отпускаем камеру/микрофон. */
  stop() {
    for (const { pc } of this.peers.values()) {
      try { pc.close() } catch { /* уже закрыт */ }
    }
    this.peers.clear()
    if (this.localStream) {
      for (const track of this.localStream.getTracks()) {
        try { track.stop() } catch { /* уже остановлен */ }
      }
      this.localStream = null
    }
  }
}
