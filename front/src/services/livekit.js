/**
 * Обёртка над livekit-client для звонков.
 *
 * Весь медиа-транспорт (SFU, ICE, reconnect, simulcast) — на LiveKit.
 * Менеджер держит «сырые» объекты Room/Track ВНЕ Vue-реактивности (Proxy
 * ломает RTCPeerConnection/MediaStream) и транслирует события Room в простые
 * CustomEvent'ы, которые слушает стор. Плитки участников берут треки напрямую
 * через getTrack()/getLocalTrack() и сами вызывают track.attach(el).
 */
import {
  Room,
  RoomEvent,
  Track,
  DisconnectReason,
} from 'livekit-client'

/** Топик data-канала для чата звонка. */
const CHAT_TOPIC = 'chat'

const encoder = new TextEncoder()
const decoder = new TextDecoder()

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

export class CallRoomManager extends EventTarget {
  constructor() {
    super()
    this.room = null
  }

  get connected() {
    return !!this.room && this.room.state === 'connected'
  }

  get localIdentity() {
    return this.room?.localParticipant?.identity || null
  }

  /**
   * Подключиться к комнате. audio/video — стартовое состояние локальных
   * устройств (выключенная камера НЕ запрашивает разрешение на неё).
   */
  async connect({ url, token, audio = true, video = true }) {
    await this.disconnect()

    const room = new Room({
      // SFU сам подбирает слои simulcast под размер плитки у получателя.
      adaptiveStream: true,
      dynacast: true,
    })
    this.room = room

    room
      .on(RoomEvent.ParticipantConnected, (p) => this._emit('participant-joined', { identity: p.identity }))
      .on(RoomEvent.ParticipantDisconnected, (p) => this._emit('participant-left', { identity: p.identity }))
      .on(RoomEvent.TrackSubscribed, (_t, _pub, p) => this._emit('track-changed', { identity: p.identity }))
      .on(RoomEvent.TrackUnsubscribed, (_t, _pub, p) => this._emit('track-changed', { identity: p.identity }))
      .on(RoomEvent.TrackMuted, (_pub, p) => this._emit('track-changed', { identity: p.identity }))
      .on(RoomEvent.TrackUnmuted, (_pub, p) => this._emit('track-changed', { identity: p.identity }))
      .on(RoomEvent.LocalTrackPublished, () => this._emit('track-changed', { identity: this.localIdentity, local: true }))
      .on(RoomEvent.LocalTrackUnpublished, () => this._emit('track-changed', { identity: this.localIdentity, local: true }))
      .on(RoomEvent.ActiveSpeakersChanged, (speakers) => {
        this._emit('speakers', { identities: speakers.map(s => s.identity) })
      })
      .on(RoomEvent.DataReceived, (payload, participant, _kind, topic) => {
        if (topic !== CHAT_TOPIC) return
        try {
          const msg = JSON.parse(decoder.decode(payload))
          this._emit('chat', {
            identity: participant?.identity || null,
            name: participant?.name || msg.name || 'Участник',
            text: String(msg.text || '').slice(0, 2000),
            ts: msg.ts || Date.now(),
          })
        } catch { /* мусор в data-канале игнорируем */ }
      })
      .on(RoomEvent.ConnectionStateChanged, (state) => this._emit('connection-state', { state }))
      .on(RoomEvent.Disconnected, (reason) => {
        this._emit('disconnected', {
          // Комнату закрыл сервер (звонок завершён) или нас выкинули — стору
          // важно отличать это от нашего собственного disconnect().
          byServer: reason === DisconnectReason.ROOM_DELETED
            || reason === DisconnectReason.PARTICIPANT_REMOVED
            || reason === DisconnectReason.SERVER_SHUTDOWN,
        })
      })

    await room.connect(resolveLivekitUrl(url), token)

    // Микрофон/камера — после connect: до него setMicrophoneEnabled не
    // публикует трек. Ошибка устройства не рвёт соединение — можно сидеть
    // «слушателем» (например, гость без камеры).
    try {
      await room.localParticipant.setMicrophoneEnabled(audio)
    } catch (e) {
      this._emit('media-error', { kind: 'audio', error: e })
    }
    if (video) {
      try {
        await room.localParticipant.setCameraEnabled(true)
      } catch (e) {
        this._emit('media-error', { kind: 'video', error: e })
      }
    }
    this._emit('connected', {})
    return room
  }

  async disconnect() {
    const room = this.room
    this.room = null
    if (room) {
      room.removeAllListeners()
      try { await room.disconnect() } catch { /* уже отключены */ }
    }
  }

  async setMicEnabled(v) {
    await this.room?.localParticipant.setMicrophoneEnabled(v)
  }

  async setCamEnabled(v) {
    await this.room?.localParticipant.setCameraEnabled(v)
  }

  async setScreenShareEnabled(v) {
    await this.room?.localParticipant.setScreenShareEnabled(v)
  }

  sendChat(text) {
    if (!this.room) return
    const payload = encoder.encode(JSON.stringify({ text, ts: Date.now() }))
    this.room.localParticipant.publishData(payload, { reliable: true, topic: CHAT_TOPIC })
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
    const mic = p.getTrackPublication(Track.Source.Microphone)
    const cam = p.getTrackPublication(Track.Source.Camera)
    const screen = p.getTrackPublication(Track.Source.ScreenShare)
    const has = (pub) => !!pub && !pub.isMuted && !!pub.track
    return { audio: has(mic), video: has(cam), screen: has(screen) }
  }

  /**
   * Трек участника для attach в плитке. source: 'camera'|'screen'|'audio'.
   * Возвращает livekit Track или null.
   */
  getTrack(identity, source) {
    const p = this._participant(identity)
    if (!p) return null
    const src = source === 'screen' ? Track.Source.ScreenShare
      : source === 'audio' ? Track.Source.Microphone
        : Track.Source.Camera
    const pub = p.getTrackPublication(src)
    if (!pub || !pub.track || pub.isMuted) return null
    return pub.track
  }

  _emit(type, detail) {
    this.dispatchEvent(new CustomEvent(type, { detail }))
  }
}

/** Singleton: в один момент времени активен максимум один звонок. */
export const callRoom = new CallRoomManager()
