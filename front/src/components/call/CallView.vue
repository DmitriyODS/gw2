<template>
  <Teleport to="body">
    <Transition name="callview">
      <div
        v-if="visible"
        class="callview"
        :class="{ mini: callStore.isMinimized, audio: callStore.media === 'audio' }"
        :style="miniStyle"
      >
        <!-- Шапка. В свёрнутом окне служит «ручкой» для перетаскивания. -->
        <header
          class="callview-header"
          :class="{ 'mini-handle': callStore.isMinimized }"
          @pointerdown="onMiniDragStart"
        >
          <div class="header-left">
            <span class="status-dot" :class="callStore.phase"></span>
            <span class="status-text">{{ statusText }}</span>
            <span v-if="callStore.phase === 'active'" class="status-time">{{ elapsed }}</span>
          </div>
          <div class="header-right">
            <button
              v-if="!callStore.isMinimized && callStore.inviteLink"
              class="header-btn link-btn"
              :class="{ copied: linkCopied }"
              :title="linkCopied ? 'Ссылка скопирована' : 'Скопировать ссылку на звонок'"
              @click="copyInviteLink"
            >
              <span class="material-symbols-outlined">{{ linkCopied ? 'check' : 'link' }}</span>
              <span class="link-label">{{ linkCopied ? 'Скопировано' : 'Ссылка' }}</span>
            </button>
            <button class="header-btn" :title="callStore.isMinimized ? 'Развернуть' : 'Свернуть'" @click="toggleMin">
              <span class="material-symbols-outlined">{{ callStore.isMinimized ? 'open_in_full' : 'close_fullscreen' }}</span>
            </button>
          </div>
        </header>

        <div class="callview-body">
          <!-- Сцена: либо демонстрация экрана + лента камер, либо сетка камер -->
          <div class="callview-stage">
            <template v-if="focusShare && !callStore.isMinimized">
              <div class="stage-focus">
                <ParticipantTile
                  :key="`screen-${focusShare.identity}`"
                  :identity="focusShare.isLocal ? null : focusShare.identity"
                  :name="focusShare.name"
                  source="screen"
                  :is-local="focusShare.isLocal"
                  :audio="true"
                  :video="true"
                  :tick="focusShare.tick"
                />
              </div>
              <div class="stage-strip">
                <ParticipantTile
                  :name="myName"
                  :is-local="true"
                  :audio="callStore.audioEnabled"
                  :video="callStore.videoEnabled"
                  :avatar="myAvatar"
                  :tick="callStore.localTick"
                />
                <ParticipantTile
                  v-for="p in callStore.participantList"
                  :key="p.identity"
                  :identity="p.identity"
                  :name="p.name"
                  :audio="p.audio"
                  :video="p.video"
                  :avatar="avatarOf(p)"
                  :pending="p.pending"
                  :speaking="p.speaking"
                  :guest="p.guest"
                  :tick="p.tick"
                />
              </div>
            </template>

            <div v-else class="callview-grid" :class="gridClass">
              <ParticipantTile
                v-if="!callStore.isMinimized || !primaryRemote"
                :name="myName"
                :is-local="true"
                :audio="callStore.audioEnabled"
                :video="callStore.videoEnabled"
                :avatar="myAvatar"
                :tick="callStore.localTick"
              />
              <ParticipantTile
                v-for="p in visibleRemotes"
                :key="p.identity"
                :identity="p.identity"
                :name="p.name"
                :audio="p.audio"
                :video="p.video"
                :avatar="avatarOf(p)"
                :pending="p.pending"
                :speaking="p.speaking"
                :guest="p.guest"
                :tick="p.tick"
              />
            </div>
          </div>

          <!-- Боковая панель: участники / чат -->
          <aside v-if="callStore.sidePanel && !callStore.isMinimized" class="callview-aside">
            <CallParticipantsPanel
              v-if="callStore.sidePanel === 'participants'"
              @invite="inviteOpen = true"
              @copy-link="copyInviteLink"
            />
            <CallChatPanel v-else-if="callStore.sidePanel === 'chat'" />
          </aside>
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
            class="ctrl-btn"
            :class="{ off: !callStore.videoEnabled }"
            :title="callStore.videoEnabled ? 'Выключить камеру' : 'Включить камеру'"
            @click="callStore.toggleCam()"
          >
            <span class="material-symbols-outlined">{{ callStore.videoEnabled ? 'videocam' : 'videocam_off' }}</span>
          </button>
          <button
            v-if="canScreenShare && !callStore.isMinimized"
            class="ctrl-btn"
            :class="{ on: callStore.screenEnabled }"
            :title="callStore.screenEnabled ? 'Остановить демонстрацию' : 'Демонстрация экрана'"
            @click="callStore.toggleScreenShare()"
          >
            <span class="material-symbols-outlined">{{ callStore.screenEnabled ? 'stop_screen_share' : 'screen_share' }}</span>
          </button>
          <button
            v-if="!callStore.isMinimized"
            class="ctrl-btn"
            :class="{ on: callStore.sidePanel === 'participants' }"
            title="Участники"
            @click="callStore.togglePanel('participants')"
          >
            <span class="material-symbols-outlined">group</span>
            <span class="ctrl-badge">{{ callStore.participantCount }}</span>
          </button>
          <button
            v-if="!callStore.isMinimized"
            class="ctrl-btn"
            :class="{ on: callStore.sidePanel === 'chat' }"
            title="Чат звонка"
            @click="callStore.togglePanel('chat')"
          >
            <span class="material-symbols-outlined">forum</span>
            <span v-if="callStore.chatUnread" class="ctrl-badge unread">{{ callStore.chatUnread }}</span>
          </button>
          <button
            v-if="!callStore.guest && callStore.phase === 'active' && !callStore.isMinimized"
            class="ctrl-btn"
            title="Пригласить участника"
            @click="inviteOpen = true"
          >
            <span class="material-symbols-outlined">person_add</span>
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

        <CallAudioSink />
      </div>
    </Transition>

    <InviteToCallDialog
      v-model="inviteOpen"
      :exclude-ids="participantIds"
      @confirm="onInviteConfirm"
    />
  </Teleport>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useCallStore } from '@/stores/call.js'
import { useAuthStore } from '@/stores/auth.js'
import ParticipantTile from './ParticipantTile.vue'
import InviteToCallDialog from './InviteToCallDialog.vue'
import CallParticipantsPanel from './CallParticipantsPanel.vue'
import CallChatPanel from './CallChatPanel.vue'
import CallAudioSink from './CallAudioSink.vue'

const callStore = useCallStore()
const authStore = useAuthStore()

const inviteOpen = ref(false)
const participantIds = computed(() =>
  callStore.participantList.map(p => p.userId).filter(Boolean))

function onInviteConfirm({ userIds }) {
  callStore.inviteToCall(userIds)
}

const visible = computed(() => callStore.phase === 'active' || callStore.phase === 'outgoing')
const isRinging = computed(() => callStore.phase === 'outgoing')

const statusText = computed(() => {
  if (callStore.phase === 'outgoing') return 'Звоним…'
  if (callStore.phase === 'active') {
    return callStore.participantCount > 2 ? 'Групповой звонок' : 'В разговоре'
  }
  return ''
})

const myName = computed(() => callStore.guest
  ? (callStore.guestName || 'Вы')
  : (authStore.user?.fio || 'Вы'))

const myAvatar = computed(() => {
  if (callStore.guest) return null
  const u = authStore.user
  if (!u) return null
  if (u.avatar_path) return `/uploads/${u.avatar_path}`
  return `/api/users/${u.id}/identicon`
})

function avatarOf(p) {
  if (p?.avatarPath) return `/uploads/${p.avatarPath}`
  if (p?.userId) return `/api/users/${p.userId}/identicon`
  return null
}

const canScreenShare = computed(() =>
  !!navigator.mediaDevices?.getDisplayMedia)

/* Демонстрация экрана: первый, кто шарит (локальный — приоритетно),
   становится «сценой», камеры уезжают в ленту снизу. */
const focusShare = computed(() => {
  if (callStore.screenEnabled) {
    return { identity: 'local', isLocal: true, name: myName.value, tick: callStore.localTick }
  }
  const p = callStore.participantList.find(x => x.screen)
  return p ? { identity: p.identity, isLocal: false, name: p.name, tick: p.tick } : null
})

const numTiles = computed(() => 1 + callStore.participantList.length)
const gridClass = computed(() => {
  if (callStore.isMinimized) return 'g-mini'
  if (numTiles.value <= 1) return 'g-1'
  if (numTiles.value === 2) return 'g-2'
  if (numTiles.value <= 4) return 'g-4'
  return 'g-many'
})

/* В свёрнутом окне показываем собеседника, а не себя: первого активного
   (не pending), иначе — первого в списке. */
const primaryRemote = computed(() => {
  const list = callStore.participantList
  if (!list.length) return null
  return list.find(p => !p.pending) || list[0]
})

const visibleRemotes = computed(() => {
  if (!callStore.isMinimized) return callStore.participantList
  return primaryRemote.value ? [primaryRemote.value] : []
})

/* Копирование ссылки-приглашения */
const linkCopied = ref(false)
let copiedTimer = null

async function copyInviteLink() {
  const link = callStore.inviteLink
  if (!link) return
  try {
    await navigator.clipboard.writeText(link)
  } catch {
    // Старый браузер/не-HTTPS: textarea-фолбэк.
    const ta = document.createElement('textarea')
    ta.value = link
    document.body.appendChild(ta)
    ta.select()
    try { document.execCommand('copy') } catch {}
    ta.remove()
  }
  linkCopied.value = true
  clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => { linkCopied.value = false }, 2000)
}

function toggleMin() {
  if (callStore.isMinimized) callStore.expand()
  else callStore.minimize()
}

/* ── Перетаскивание свёрнутого окна ─────────────────────────────── */
const miniPos = ref(null) // { left, top } в px, либо null = угол по CSS
let dragging = false
let dragOffset = { x: 0, y: 0 }

const miniStyle = computed(() => {
  if (!callStore.isMinimized || !miniPos.value) return null
  return {
    left: `${miniPos.value.left}px`,
    top: `${miniPos.value.top}px`,
    right: 'auto',
    bottom: 'auto',
  }
})

function onMiniDragStart(e) {
  if (!callStore.isMinimized) return
  if (e.target.closest('button')) return
  const el = e.currentTarget.closest('.callview')
  if (!el) return
  const rect = el.getBoundingClientRect()
  dragOffset = { x: e.clientX - rect.left, y: e.clientY - rect.top }
  miniPos.value = { left: rect.left, top: rect.top }
  dragging = true
  window.addEventListener('pointermove', onMiniDragMove)
  window.addEventListener('pointerup', onMiniDragEnd)
  e.preventDefault()
}

function onMiniDragMove(e) {
  if (!dragging) return
  const el = document.querySelector('.callview.mini')
  const w = el?.offsetWidth || 320
  const h = el?.offsetHeight || 240
  const left = Math.max(8, Math.min(e.clientX - dragOffset.x, window.innerWidth - w - 8))
  const top = Math.max(8, Math.min(e.clientY - dragOffset.y, window.innerHeight - h - 8))
  miniPos.value = { left, top }
}

function onMiniDragEnd() {
  dragging = false
  window.removeEventListener('pointermove', onMiniDragMove)
  window.removeEventListener('pointerup', onMiniDragEnd)
}

watch(visible, (v) => { if (!v) miniPos.value = null })

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
  onMiniDragEnd()
  clearTimeout(copiedTimer)
})

/* Ringback tone — гудки «туу...туу...» пока звоним и никто не вошёл.
   Чистый тон 425 Гц (российская АТС), 1с звук + 4с тишина. Защита от
   suspended AudioContext — первый жест пользователя «разогревает» звук. */
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

.callview.mini .callview-header {
  padding: 8px 12px;
  font-size: 12px;
}

.callview-header.mini-handle {
  cursor: grab;
  touch-action: none;
  user-select: none;
}

.callview-header.mini-handle:active {
  cursor: grabbing;
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

.header-right { display: flex; gap: 4px; align-items: center; }

.header-btn {
  height: 36px;
  min-width: 36px;
  border-radius: 999px;
  border: 0;
  background: transparent;
  color: var(--color-text);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  cursor: pointer;
  transition: background 0.15s;
  font-family: inherit;
}

.header-btn:hover { background: var(--color-surface-high); }
.header-btn .material-symbols-outlined { font-size: 20px; }

.link-btn {
  padding: 0 14px;
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  font-size: 13px;
  font-weight: 600;
}

.link-btn.copied {
  background: var(--color-success-container, var(--color-primary-container));
  color: var(--color-on-success-container, var(--color-on-primary-container));
}

.link-label { white-space: nowrap; }

.callview.mini .header-btn { width: 28px; height: 28px; min-width: 28px; }
.callview.mini .header-btn .material-symbols-outlined { font-size: 16px; }

/* Тело: сцена + боковая панель */
.callview-body {
  flex: 1;
  min-height: 0;
  display: flex;
}

.callview-stage {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--color-surface-low);
}

/* Сетка камер */
.callview-grid {
  flex: 1;
  min-height: 0;
  display: grid;
  gap: 8px;
  padding: 16px;
}

.callview.mini .callview-grid { padding: 6px; gap: 4px; }

.g-1 { grid-template-columns: 1fr; }
.g-2 { grid-template-columns: 1fr 1fr; }
.g-4 { grid-template-columns: 1fr 1fr; grid-auto-rows: 1fr; }
.g-many { grid-template-columns: repeat(3, 1fr); grid-auto-rows: 1fr; }

/* Демонстрация экрана: сцена + лента камер */
.stage-focus {
  flex: 1;
  min-height: 0;
  padding: 16px 16px 8px;
  display: flex;
}

.stage-focus > * { flex: 1; }

.stage-strip {
  display: flex;
  gap: 8px;
  padding: 0 16px 12px;
  overflow-x: auto;
  flex-shrink: 0;
}

.stage-strip > * {
  width: 168px;
  min-width: 168px;
  min-height: 100px;
  height: 100px;
}

/* Боковая панель */
.callview-aside {
  width: 320px;
  flex-shrink: 0;
  border-left: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  min-height: 0;
}

@media (max-width: 900px) {
  .callview-aside {
    position: absolute;
    inset: 0;
    width: auto;
    z-index: 5;
    border-left: 0;
  }
}

@media (max-width: 720px) {
  .g-2, .g-4 { grid-template-columns: 1fr; }
  .g-many { grid-template-columns: 1fr 1fr; }
}

/* На мобильном свёрнутый режим поднимаем над нижней навигацией. */
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
  gap: 14px;
  padding: 18px 16px calc(18px + env(safe-area-inset-bottom, 0px));
  background: color-mix(in oklch, var(--color-surface) 88%, transparent);
  border-top: 1px solid var(--color-outline-dim);
  flex-shrink: 0;
}

.callview.mini .callview-controls {
  padding: 8px;
  gap: 8px;
}

.ctrl-btn {
  position: relative;
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

.ctrl-btn.on {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.ctrl-btn.hangup {
  background: var(--color-error);
  color: var(--color-on-error);
  width: 64px;
  height: 64px;
}

.ctrl-badge {
  position: absolute;
  top: -2px;
  right: -2px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  font-size: 11px;
  font-weight: 700;
  display: grid;
  place-items: center;
}

.ctrl-badge.unread {
  background: var(--color-error);
  color: var(--color-on-error);
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
  z-index: 6;
}

/* Анимация */
.callview-enter-active, .callview-leave-active { transition: opacity 0.22s; }
.callview-enter-from, .callview-leave-to { opacity: 0; }
</style>
