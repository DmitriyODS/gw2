// Смоук-тест сигналинга WebRTCManager с моками WebRTC API.
//
// Проверяет два сценария:
//  1) Нормальный (детерминированный, без glare): «новенький» — единственный
//     offerer (connectTo с offerer:true), существующий только отвечает
//     (connectTo без offerer). Должен пройти ровно один offer→answer, оба peer
//     приходят в 'stable'.
//  2) Страховочный perfect-negotiation: если из-за гонки обе стороны стали
//     offerer одновременно (glare), обмен всё равно сходится в 'stable', а роли
//     polite/impolite противоположны (детерминированы по user_id).
//
// Запуск:  node front/tests/webrtc.test.mjs
import { WebRTCManager } from '../src/services/webrtc.js'

class MockTrack { constructor(kind){ this.kind = kind; this.enabled = true } stop(){} }
class MockStream {
  constructor(tracks=[]){ this._tracks = tracks }
  getTracks(){ return this._tracks }
  getAudioTracks(){ return this._tracks.filter(t=>t.kind==='audio') }
  getVideoTracks(){ return this._tracks.filter(t=>t.kind==='video') }
  addTrack(t){ this._tracks.push(t) }
  removeTrack(t){ this._tracks = this._tracks.filter(x=>x!==t) }
}
globalThis.MediaStream = MockStream
globalThis.RTCIceCandidate = class { constructor(c){ Object.assign(this, c) } }
globalThis.RTCSessionDescription = class { constructor(d){ Object.assign(this, d) } }

let sdpSeq = 0
class MockPC {
  constructor(){ this.signalingState='stable'; this.connectionState='new'; this.iceConnectionState='new'
    this.localDescription=null; this.remoteDescription=null; this._senders=[]
    this.onicecandidate=null; this.ontrack=null; this.onnegotiationneeded=null
    this.onconnectionstatechange=null; this.oniceconnectionstatechange=null }
  getSenders(){ return this._senders }
  addTrack(track){ this._senders.push({ track })
    // Как в реальном браузере: negotiationneeded дебаунсится и стреляет один
    // раз и только когда signalingState === 'stable'.
    if (!this._negScheduled) { this._negScheduled = true
      queueMicrotask(()=>{ this._negScheduled=false
        if (this.signalingState==='stable'){ this.onnegotiationneeded && this.onnegotiationneeded() } }) } }
  async setLocalDescription(desc){
    if (!desc) {
      if (this.signalingState==='stable' || this.signalingState==='have-local-offer') {
        desc = { type:'offer', sdp:`offer-${++sdpSeq}` }; this.signalingState='have-local-offer'
      } else { desc = { type:'answer', sdp:`answer-${++sdpSeq}` }; this.signalingState='stable' }
    }
    this.localDescription = desc
  }
  async setRemoteDescription(desc){
    this.remoteDescription = desc
    // setRemoteDescription(offer) в have-local-offer = implicit rollback (как в спеке).
    if (desc.type==='offer') this.signalingState='have-remote-offer'
    else this.signalingState='stable'
  }
  async addIceCandidate(){}
  restartIce(){}
  close(){ this.connectionState='closed' }
}
globalThis.RTCPeerConnection = MockPC
Object.defineProperty(globalThis, 'navigator', {
  value: { mediaDevices: { getUserMedia: async () => new MockStream([new MockTrack('audio'), new MockTrack('video')]) } },
  configurable: true,
})

let offers = 0, answers = 0
function wire(a, b) {
  a.addEventListener('local-signal', (e) => {
    const { kind, payload } = e.detail
    if (kind === 'sdp' && payload?.type === 'offer') offers++
    if (kind === 'sdp' && payload?.type === 'answer') answers++
    queueMicrotask(async () => {
      try {
        if (kind === 'sdp') await b.handleDescription(a.myUserId, payload)
        else if (kind === 'ice') await b.handleRemoteIce(a.myUserId, payload)
      } catch (err) { console.error('SIGNAL THROW', kind, err); process.exitCode = 1 }
    })
  })
}

const tick = () => new Promise(r => setTimeout(r, 30))
let ok = true

// ── Сценарий 1: детерминированный, один offerer ──────────────────
{
  const m1 = new WebRTCManager({}); m1.setMyUserId(1) // существующий
  const m2 = new WebRTCManager({}); m2.setMyUserId(2) // новенький
  wire(m1, m2); wire(m2, m1)
  await m1.start('video'); await m2.start('video')

  offers = 0; answers = 0
  m2.connectTo(1, { offerer: true }) // новенький инициирует offer
  m1.connectTo(2)                    // существующий только отвечает
  await tick(); await tick(); await tick()

  const s1 = m1.peers.get(2).pc.signalingState
  const s2 = m2.peers.get(1).pc.signalingState
  console.log('[1] offers:', offers, '| answers:', answers, '| s1:', s1, '| s2:', s2)
  if (s1 !== 'stable' || s2 !== 'stable') { console.error('FAIL[1]: не сошлось в stable'); ok = false }
  if (offers !== 1 || answers !== 1) { console.error('FAIL[1]: ожидался ровно один offer и answer'); ok = false }
}

// ── Сценарий 2: glare (страховка perfect-negotiation) ────────────
{
  const m1 = new WebRTCManager({}); m1.setMyUserId(1)
  const m2 = new WebRTCManager({}); m2.setMyUserId(2)
  wire(m1, m2); wire(m2, m1)
  await m1.start('video'); await m2.start('video')

  m1.connectTo(2, { offerer: true }) // оба считают себя offerer (гонка)
  m2.connectTo(1, { offerer: true })
  await tick(); await tick(); await tick()

  const s1 = m1.peers.get(2).pc.signalingState
  const s2 = m2.peers.get(1).pc.signalingState
  console.log('[2] s1:', s1, '| s2:', s2, '| polite1:', m1.peers.get(2).polite, '| polite2:', m2.peers.get(1).polite)
  if (s1 !== 'stable' || s2 !== 'stable') { console.error('FAIL[2]: glare не сошёлся в stable'); ok = false }
  if (m1.peers.get(2).polite === m2.peers.get(1).polite) { console.error('FAIL[2]: politeness не противоположен'); ok = false }
}

if (process.exitCode === 1) ok = false
console.log(ok ? '\nOK: сигналинг сошёлся в обоих сценариях' : '\nFAILED')
process.exit(ok ? 0 : 1)
