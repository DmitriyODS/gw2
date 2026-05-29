<template>
  <Teleport to="body">
    <Transition name="callview">
      <div
        v-if="visible"
        class="callview"
        :class="{ mini: callStore.isMinimized, audio: callStore.media === 'audio' }"
      >
        <!-- Шапка -->
        <header class="callview-header">
          <div class="header-left">
            <span class="status-dot" :class="callStore.phase"></span>
            <span class="status-text">{{ statusText }}</span>
            <span v-if="callStore.phase === 'active'" class="status-time">{{ elapsed }}</span>
          </div>
          <div class="header-right">
            <button class="header-btn" :title="callStore.isMinimized ? 'Развернуть' : 'Свернуть'" @click="toggleMin">
              <span class="material-symbols-outlined">{{ callStore.isMinimized ? 'open_in_full' : 'close_fullscreen' }}</span>
            </button>
          </div>
        </header>

        <!-- Сетка участников -->
        <div class="callview-grid" :class="gridClass">
          <!-- Локальное превью -->
          <ParticipantTile
            :name="'Вы'"
            :stream="callStore.localStream"
            :audio-enabled="callStore.audioEnabled"
            :video-enabled="callStore.videoEnabled"
            :is-local="true"
            :avatar="myAvatar"
          />
          <!-- Удалённые участники -->
          <ParticipantTile
            v-for="(p, uid) in callStore.remoteStreams"
            :key="uid"
            :name="p.fio"
            :stream="p.stream"
            :stream-tick="p.streamTick"
            :audio-enabled="p.audio"
            :video-enabled="p.video"
            :avatar="avatarOf(p)"
            :pending="!p.stream"
            :conn-state="p.conn"
          />
        </div>

        <!-- Контролы -->
        <div class="callview-controls">
          <button
            class="ctrl-btn"
            :class="{ off: !callStore.audioEnabled }"
            :title="callStore.audioEnabled ? 'Выключить микрофон' : 'Включить микрофон'"
            @click="callStore.toggleMic()"
          >
            <span class="material-symbols-outlined">{{ callStore.audioEnabled ? 'mic' : 'mic_off' }}</span>
          </button>
          <button
            v-if="callStore.media === 'video'"
            class="ctrl-btn"
            :class="{ off: !callStore.videoEnabled }"
            :title="callStore.videoEnabled ? 'Выключить камеру' : 'Включить камеру'"
            @click="callStore.toggleCam()"
          >
            <span class="material-symbols-outlined">{{ callStore.videoEnabled ? 'videocam' : 'videocam_off' }}</span>
          </button>
          <button
            class="ctrl-btn hangup"
            title="Завершить звонок"
            @click="callStore.hangup()"
          >
            <span class="material-symbols-outlined">call_end</span>
          </button>
        </div>

        <div v-if="callStore.error" class="callview-error">{{ callStore.error }}</div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useCallStore } from '@/stores/call.js'
import { useAuthStore } from '@/stores/auth.js'
import ParticipantTile from './ParticipantTile.vue'

const callStore = useCallStore()
const authStore = useAuthStore()

const visible = computed(() => callStore.phase === 'active' || callStore.phase === 'outgoing')
const isRinging = computed(() => callStore.phase === 'outgoing')

const statusText = computed(() => {
  if (callStore.phase === 'outgoing') return 'Звоним…'
  if (callStore.phase === 'active') return callStore.call?.kind === 'group' ? 'Групповой звонок' : 'В разговоре'
  return ''
})

const myAvatar = computed(() => {
  const u = authStore.user
  if (!u) return null
  if (u.avatar_path) return `/uploads/${u.avatar_path}`
  return `/api/users/${u.id}/identicon`
})

function avatarOf(p) {
  if (p?.avatar_path) return `/uploads/${p.avatar_path}`
  return null
}

const numTiles = computed(() => 1 + Object.keys(callStore.remoteStreams).length)
const gridClass = computed(() => {
  if (callStore.isMinimized) return 'g-mini'
  if (numTiles.value <= 1) return 'g-1'
  if (numTiles.value === 2) return 'g-2'
  if (numTiles.value <= 4) return 'g-4'
  return 'g-many'
})

function toggleMin() {
  if (callStore.isMinimized) callStore.expand()
  else callStore.minimize()
}

/* Таймер длительности звонка */
const elapsedSec = ref(0)
let timer = null

function startTimer() {
  if (timer) return
  const startedAt = callStore.call?.started_at ? new Date(callStore.call.started_at).getTime() : Date.now()
  timer = setInterval(() => {
    elapsedSec.value = Math.max(0, Math.floor((Date.now() - startedAt) / 1000))
  }, 1000)
}

function stopTimer() {
  if (timer) { clearInterval(timer); timer = null }
  elapsedSec.value = 0
}

const elapsed = computed(() => {
  const s = elapsedSec.value
  const m = Math.floor(s / 60)
  const sec = (s % 60).toString().padStart(2, '0')
  return m >= 60
    ? `${Math.floor(m / 60)}:${(m % 60).toString().padStart(2, '0')}:${sec}`
    : `${m}:${sec}`
})

onMounted(startTimer)
onBeforeUnmount(() => {
  stopTimer()
  stopRingback()
})

/* Ringback tone — гудки «туу...туу...» пока звоним и собеседник не ответил.
   Стандарт: 425 Гц, 1с звук + 4с тишина (российский тип) либо 440Гц 2с/4с
   (US). Берём что-то в духе российской АТС: чистый тон 425Гц, длительность
   1с, период 5с. Останавливаем как только phase != outgoing (accepted или
   hangup). Защита от suspended AudioContext — как в IncomingCallOverlay:
   первый жест разогревает звук. */
let ringCtx = null
let ringTimer = null
let pendingGesture = null

function playOneBeep() {
  if (!ringCtx) return
  try {
    if (ringCtx.state === 'suspended') ringCtx.resume()
    const now = ringCtx.currentTime
    const osc = ringCtx.createOscillator()
    const gain = ringCtx.createGain()
    osc.type = 'sine'
    osc.frequency.value = 425
    gain.gain.setValueAtTime(0, now)
    gain.gain.linearRampToValueAtTime(0.18, now + 0.05)
    gain.gain.setValueAtTime(0.18, now + 0.95)
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 1.0)
    osc.connect(gain).connect(ringCtx.destination)
    osc.start(now)
    osc.stop(now + 1.02)
  } catch {}
}

function installRingbackGestureRetry() {
  if (pendingGesture) return
  pendingGesture = () => {
    pendingGesture = null
    if (ringCtx && ringCtx.state === 'suspended') {
      ringCtx.resume().catch(() => {})
    }
    window.removeEventListener('pointerdown', pendingGesture, true)
    window.removeEventListener('keydown', pendingGesture, true)
  }
  window.addEventListener('pointerdown', pendingGesture, true)
  window.addEventListener('keydown', pendingGesture, true)
}

function startRingback() {
  if (ringTimer) return
  try {
    const Ctx = window.AudioContext || window.webkitAudioContext
    if (!Ctx) return
    ringCtx = new Ctx()
    if (ringCtx.state === 'suspended') {
      ringCtx.resume().catch(() => {})
      installRingbackGestureRetry()
    }
  } catch { return }
  playOneBeep()
  ringTimer = setInterval(() => playOneBeep(), 5000)
}

function stopRingback() {
  if (ringTimer) { clearInterval(ringTimer); ringTimer = null }
  if (ringCtx) { try { ringCtx.close() } catch {}; ringCtx = null }
  if (pendingGesture) {
    window.removeEventListener('pointerdown', pendingGesture, true)
    window.removeEventListener('keydown', pendingGesture, true)
    pendingGesture = null
  }
}

watch(isRinging, (v) => {
  if (v) startRingback()
  else stopRingback()
}, { immediate: true })
</script>

<style scoped>
.callview {
  position: fixed;
  inset: 0;
  z-index: 11500;
  background: var(--color-bg);
  display: flex;
  flex-direction: column;
  color: var(--color-text);
}

/* Свёрнутый режим — плавающая мини-панель в углу */
.callview.mini {
  inset: auto 16px 90px auto;
  width: 320px;
  height: 240px;
  border-radius: 24px;
  overflow: hidden;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  box-shadow: 0 12px 36px color-mix(in oklch, var(--color-scrim) 35%, transparent);
}

/* Шапка */
.callview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 22px;
  padding-top: calc(14px + env(safe-area-inset-top, 0px));
  background: color-mix(in oklch, var(--color-surface) 80%, transparent);
  border-bottom: 1px solid var(--color-outline-dim);
  flex-shrink: 0;
}

.callview.mini .callview-header { padding-top: 8px; }

.callview.mini .callview-header {
  padding: 8px 12px;
  font-size: 12px;
}

.header-left { display: flex; align-items: center; gap: 10px; min-width: 0; }

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--color-text-dim);
  flex-shrink: 0;
}

.status-dot.active { background: var(--color-success); box-shadow: 0 0 0 4px color-mix(in oklch, var(--color-success) 25%, transparent); }
.status-dot.outgoing { background: var(--color-warning); animation: blink 1.2s ease-in-out infinite; }

@keyframes blink { 0%,100% { opacity: 1 } 50% { opacity: 0.35 } }

.status-text {
  font-weight: 600;
  font-size: 14px;
  color: var(--color-text);
}

.status-time {
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: var(--color-text-dim);
}

.header-right { display: flex; gap: 4px; }

.header-btn {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 0;
  background: transparent;
  color: var(--color-text);
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: background 0.15s;
}

.header-btn:hover { background: var(--color-surface-high); }
.header-btn .material-symbols-outlined { font-size: 20px; }

.callview.mini .header-btn { width: 28px; height: 28px; }
.callview.mini .header-btn .material-symbols-outlined { font-size: 16px; }

/* Сетка */
.callview-grid {
  flex: 1;
  min-height: 0;
  display: grid;
  gap: 8px;
  padding: 16px;
  background: var(--color-surface-low);
}

.callview.mini .callview-grid { padding: 6px; gap: 4px; }

.g-1 { grid-template-columns: 1fr; }
.g-2 { grid-template-columns: 1fr 1fr; }
.g-4 { grid-template-columns: 1fr 1fr; grid-auto-rows: 1fr; }
.g-many { grid-template-columns: repeat(3, 1fr); grid-auto-rows: 1fr; }

@media (max-width: 720px) {
  .g-2, .g-4 { grid-template-columns: 1fr; }
  .g-many { grid-template-columns: 1fr 1fr; }
}

/* На мобильном свёрнутый режим поднимаем над нижней навигацией и сужаем,
   чтобы окошко не перекрывало панель навигации и safe-area. */
@media (max-width: 600px) {
  .callview.mini {
    inset: auto 12px calc(76px + env(safe-area-inset-bottom, 0px)) 12px;
    width: auto;
    height: 200px;
  }
}

.g-mini { grid-template-columns: 1fr; }

/* Контролы */
.callview-controls {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 18px;
  padding: 20px 16px calc(20px + env(safe-area-inset-bottom, 0px));
  background: color-mix(in oklch, var(--color-surface) 88%, transparent);
  border-top: 1px solid var(--color-outline-dim);
  flex-shrink: 0;
}

.callview.mini .callview-controls {
  padding: 8px;
  gap: 8px;
}

.ctrl-btn {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  border: 0;
  background: var(--color-surface-high);
  color: var(--color-text);
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: background 0.15s, transform 0.15s;
}

.ctrl-btn:hover { transform: translateY(-2px); }
.ctrl-btn:active { transform: translateY(0); }

.ctrl-btn.off {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.ctrl-btn.hangup {
  background: var(--color-error);
  color: var(--color-on-error);
  width: 64px;
  height: 64px;
}

.ctrl-btn .material-symbols-outlined { font-size: 24px; }
.ctrl-btn.hangup .material-symbols-outlined { font-size: 26px; }

.callview.mini .ctrl-btn { width: 36px; height: 36px; }
.callview.mini .ctrl-btn.hangup { width: 40px; height: 40px; }
.callview.mini .ctrl-btn .material-symbols-outlined { font-size: 18px; }

.callview-error {
  position: absolute;
  top: 60px;
  left: 50%;
  transform: translateX(-50%);
  padding: 8px 14px;
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  border-radius: 999px;
  font-size: 13px;
  font-weight: 600;
}

/* Анимация */
.callview-enter-active, .callview-leave-active { transition: opacity 0.22s; }
.callview-enter-from, .callview-leave-to { opacity: 0; }
</style>
