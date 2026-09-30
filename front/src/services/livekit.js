/**
 * Обёртка над livekit-client для звонков.
 *
 * Весь медиа-транспорт (SFU, ICE, reconnect, simulcast) — на LiveKit.
 * Менеджер держит «сырые» объекты Room/Track ВНЕ Vue-реактивности (Proxy
 * ломает RTCPeerConnection/MediaStream) и транслирует события Room в простые
 * CustomEvent'ы, которые слушает стор. Плитки участников берут треки напрямую
 * через getTrack()/getLocalTrack() и сами вызывают track.attach(el).
 */
/* Пакет livekit-client — самая тяжёлая зависимость фронта (~0,5 МБ после
   минификации). Стор звонков нужен App.vue всегда (ринг-фаза приходит по
   сокету), поэтому при статическом импорте пакет оказывался в инициальном
   чанке и качался на КАЖДОМ заходе — включая лендинг и экран входа. Грузим
   его при первом подключении к комнате; к этому моменту всё равно идёт сеть.

   Модуль кладём в `lk`: синхронные методы (mediaState/getTrack) обращаются к
   Track.Source, но вызывают их только при живой комнате — то есть уже после
   загрузки. */
let lk = null
let lkLoading = null

function loadLivekit() {
  if (lk) return Promise.resolve(lk)
  // Single-flight: параллельные connect не должны тянуть чанк дважды.
  if (!lkLoading) lkLoading = import('livekit-client').then((m) => { lk = m; return m })
  return lkLoading
}

/** Топик data-канала для чата звонка. */
const CHAT_TOPIC = 'chat'

/** '/livekit' → wss://host/livekit (по схеме страницы); абсолютные оставляем. */
export function resolveLivekitUrl(url) {
  if (!url) return null
  if (url.startsWith('ws://') || url.startsWith('wss://')) return url
  if (url.startsWith('http://')) return 'ws://' + url.slice(7)
  if (url.startsWith('https://')) return 'wss://' + url.slice(8)
  const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const path = url.startsWith('/') ? url : `/${url}`
  return `${scheme}://${window.location.host}${path}`
}

export function parseParticipantMetadata(participant) {
  try {
    return participant?.metadata ? JSON.parse(participant.metadata) : {}
  } catch {
    return {}
  }
}

/** Демонстрация экрана: чёткий текст вместо плавности, без «зеркала» своей
 *  вкладки, с возможностью отдать звук вкладки/системы (галочка в диалоге
 *  выбора браузера). */
const SCREEN_SHARE_OPTIONS = {
  audio: true,
  contentHint: 'detail',
  selfBrowserSurface: 'exclude',
  surfaceSwitching: 'include',
  systemAudio: 'include',
}

/** Подключение отменено (повесили трубку, пока шёл вход) — не ошибка. */
function abortError() {
  const e = new Error('Подключение отменено')
  e.name = 'AbortError'
  return e
}

export class CallRoomManager extends EventTarget {
  constructor() {
    super()
    this.room = null
    // Поколение подключения: disconnect() во время connect() делает текущую
    // попытку устаревшей, и она сворачивается, не публикуя треки в чужую
    // (уже закрытую) комнату — иначе микрофон и камера оставались включены.
    this._gen = 0
  }

  get connected() {
    return !!this.room && this.room.state === 'connected'
  }

  get localIdentity() {
    return this.room?.localParticipant?.identity || null
  }

  /** Браузер разрешил воспроизводить звук (иначе нужен жест — startAudio). */
  get canPlaybackAudio() {
    return this.room ? this.room.canPlaybackAudio : true
  }

  /** Подтянуть SDK заранее (входящий, дозвон): к «Принять» чанк уже в кеше. */
  preload() {
    return loadLivekit().then(() => {}, () => {})
  }

  /**
   * Подключиться к комнате. audio/video — стартовое состояние локальных
   * устройств (выключенная камера НЕ запрашивает разрешение на неё).
   */
  async connect({ url, token, audio = true, video = true }) {
    await this.disconnect()
    const gen = ++this._gen

    const {
      Room, RoomEvent, DisconnectReason, createLocalAudioTrack, createLocalVideoTrack,
    } = await loadLivekit()
    if (gen !== this._gen) throw abortError()

    const room = new Room({
      // SFU сам подбирает слои simulcast под размер плитки у получателя.
      adaptiveStream: true,
      dynacast: true,
    })
    this.room = room

    const emitLocal = () => this._emit('track-changed', { identity: this.localIdentity, local: true })
    room
      .on(RoomEvent.ParticipantConnected, (p) => this._emit('participant-joined', { identity: p.identity }))
      .on(RoomEvent.ParticipantDisconnected, (p) => this._emit('participant-left', { identity: p.identity }))
      .on(RoomEvent.TrackSubscribed, (_t, _pub, p) => this._emit('track-changed', { identity: p.identity }))
      .on(RoomEvent.TrackUnsubscribed, (_t, _pub, p) => this._emit('track-changed', { identity: p.identity }))
      .on(RoomEvent.TrackMuted, (_pub, p) => this._emit('track-changed', { identity: p.identity }))
      .on(RoomEvent.TrackUnmuted, (_pub, p) => this._emit('track-changed', { identity: p.identity }))
      .on(RoomEvent.LocalTrackPublished, emitLocal)
      .on(RoomEvent.LocalTrackUnpublished, emitLocal)
      .on(RoomEvent.ActiveSpeakersChanged, (speakers) => {
        this._emit('speakers', { identities: speakers.map(s => s.identity) })
      })
      .on(RoomEvent.ConnectionQualityChanged, (quality, p) => {
        this._emit('quality', { identity: p.identity, local: p === room.localParticipant, quality })
      })
      // Полное переподключение (сменилась сеть, долгий обрыв) — медиа стоит,
      // пользователю нужно это видеть. Лёгкое восстановление сигнального
      // канала медиа не прерывает и в интерфейсе не отражается.
      .on(RoomEvent.Reconnecting, () => this._emit('reconnecting', {}))
      .on(RoomEvent.Reconnected, () => this._emit('reconnected', {}))
      // Автовоспроизведение звука запрещено (вход без жеста — например,
      // авто-возврат в звонок после перезагрузки): нужен клик и startAudio().
      .on(RoomEvent.AudioPlaybackStatusChanged, () => {
        this._emit('audio-playback', { canPlay: room.canPlaybackAudio })
      })
      .on(RoomEvent.MediaDevicesChanged, () => this._emit('devices-changed', {}))
      .on(RoomEvent.ActiveDeviceChanged, () => this._emit('devices-changed', {}))
      .on(RoomEvent.Disconnected, (reason) => {
        this._emit('disconnected', {
          // Комнату закрыл сервер (звонок завершён) или нас выкинули — стору
          // важно отличать это от обрыва связи, после которого возвращаются.
          byServer: reason === DisconnectReason.ROOM_DELETED
            || reason === DisconnectReason.ROOM_CLOSED
            || reason === DisconnectReason.PARTICIPANT_REMOVED
            || reason === DisconnectReason.SERVER_SHUTDOWN,
          // Этим же identity вошли в другом месте (вторая вкладка/устройство,
          // ссылка-приглашение): LiveKit вышиб это соединение.
          duplicate: reason === DisconnectReason.DUPLICATE_IDENTITY,
        })
      })

    room.registerTextStreamHandler(CHAT_TOPIC, async (reader, { identity }) => {
      try {
        const text = await reader.readAll()
        this._emit('chat', {
          identity,
          name: room.remoteParticipants.get(identity)?.name || 'Участник',
          text: String(text || '').slice(0, 2000),
          ts: reader.info?.timestamp || Date.now(),
        })
      } catch { /* поток оборвался — сообщение потеряно вместе с ним */ }
    })

    // Разрешения и захват устройств — параллельно с сигнальным подключением:
    // это самые долгие шаги входа, последовательно они складывались.
    const captured = Promise.allSettled([
      audio ? createLocalAudioTrack() : null,
      video ? createLocalVideoTrack() : null,
    ])

    try {
      await room.connect(resolveLivekitUrl(url), token)
    } catch (e) {
      stopCaptured(await captured)
      if (this.room === room) this.room = null
      throw gen !== this._gen ? abortError() : e
    }

    const [mic, cam] = await captured
    if (gen !== this._gen) {
      stopCaptured([mic, cam])
      throw abortError()
    }
    // Ошибка устройства не рвёт соединение — можно сидеть «слушателем»
    // (например, гость без камеры).
    await this._publish(room, mic, 'audio')
    await this._publish(room, cam, 'video')
    if (gen !== this._gen) throw abortError()

    this._emit('connected', {})
    this._emit('audio-playback', { canPlay: room.canPlaybackAudio })
    return room
  }

  async _publish(room, result, kind) {
    if (result.status === 'rejected') {
      this._emit('media-error', { kind, error: result.reason })
      return
    }
    const track = result.value
    if (!track) return
    try {
      await room.localParticipant.publishTrack(track)
    } catch (e) {
      track.stop()
      if (this.room === room) this._emit('media-error', { kind, error: e })
    }
  }

  async disconnect() {
    this._gen++
    const room = this.room
    this.room = null
    if (room) {
      room.removeAllListeners()
      try { await room.disconnect() } catch { /* уже отключены */ }
    }
  }

  /** Включить звук после запрета автовоспроизведения (зовётся из клика). */
  async startAudio() {
    await this.room?.startAudio()
    return this.canPlaybackAudio
  }

  async setMicEnabled(v) {
    await this.room?.localParticipant.setMicrophoneEnabled(v)
  }

  async setCamEnabled(v) {
    await this.room?.localParticipant.setCameraEnabled(v)
  }

  async setScreenShareEnabled(v) {
    await this.room?.localParticipant.setScreenShareEnabled(v, v ? SCREEN_SHARE_OPTIONS : undefined)
  }

  /** Сообщение в чат звонка; отказ доставки — исключением. */
  async sendChat(text) {
    if (!this.room) throw new Error('Нет соединения со звонком')
    await this.room.localParticipant.sendText(text, { topic: CHAT_TOPIC })
  }

  /** Устройства: микрофоны, камеры и (где браузер умеет) выходы звука. */
  async listDevices() {
    const { Room } = await loadLivekit()
    const kinds = ['audioinput', 'videoinput', 'audiooutput']
    const lists = await Promise.all(kinds.map(k => Room.getLocalDevices(k, false).catch(() => [])))
    return Object.fromEntries(kinds.map((k, i) => [k, lists[i].filter(d => d.deviceId)]))
  }

  activeDevice(kind) {
    return this.room?.getActiveDevice(kind) || 'default'
  }

  async switchDevice(kind, deviceId) {
    if (!this.room) return false
    return this.room.switchActiveDevice(kind, deviceId)
  }

  /** Снимок удалённых участников (LiveKit Participant[]). */
  remoteParticipants() {
    return this.room ? Array.from(this.room.remoteParticipants.values()) : []
  }

  _participant(identity) {
    if (!this.room) return null
    if (identity === this.localIdentity) return this.room.localParticipant
    return this.room.remoteParticipants.get(identity) || null
  }

  /** Состояние треков участника для UI (микрофон/камера/демонстрация). */
  mediaState(identity) {
    const p = this._participant(identity)
    if (!p) return { audio: false, video: false, screen: false }
    const { Track } = lk
    const mic = p.getTrackPublication(Track.Source.Microphone)
    const cam = p.getTrackPublication(Track.Source.Camera)
    const screen = p.getTrackPublication(Track.Source.ScreenShare)
    const has = (pub) => !!pub && !pub.isMuted && !!pub.track
    return { audio: has(mic), video: has(cam), screen: has(screen) }
  }

  /**
   * Трек участника для attach. source: 'camera' | 'screen' | 'audio' |
   * 'screen-audio' (звук демонстрации). Возвращает livekit Track или null.
   */
  getTrack(identity, source) {
    const p = this._participant(identity)
    if (!p) return null
    const { Track } = lk
    const src = {
      screen: Track.Source.ScreenShare,
      audio: Track.Source.Microphone,
      'screen-audio': Track.Source.ScreenShareAudio,
    }[source] || Track.Source.Camera
    const pub = p.getTrackPublication(src)
    if (!pub || !pub.track || pub.isMuted) return null
    return pub.track
  }

  _emit(type, detail) {
    this.dispatchEvent(new CustomEvent(type, { detail }))
  }
}

function stopCaptured(results) {
  for (const r of results) {
    if (r.status === 'fulfilled' && r.value) r.value.stop()
  }
}

/** Singleton: в один момент времени активен максимум один звонок. */
export const callRoom = new CallRoomManager()
