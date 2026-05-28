<template>
  <Teleport to="body">
    <Transition name="incoming">
      <div v-if="show" class="incoming-overlay" @click.self="$emit('decline')">
        <div class="incoming-card" :class="{ pulse: callStore.isIncoming }">
          <div class="incoming-tag">
            <span class="material-symbols-outlined">{{ mediaIcon }}</span>
            {{ tagLabel }}
          </div>

          <div class="incoming-avatar-wrap">
            <img v-if="primaryAvatar" :src="primaryAvatar" class="incoming-avatar" :alt="primaryName" />
            <div v-else class="incoming-avatar avatar-fallback">
              <span class="material-symbols-outlined">person</span>
            </div>
            <div class="ring ring-1"></div>
            <div class="ring ring-2"></div>
          </div>

          <h2 class="incoming-name">{{ primaryName }}</h2>
          <p class="incoming-sub">{{ subtitle }}</p>

          <div class="incoming-actions">
            <button class="round-btn decline" @click="$emit('decline')" title="Отклонить">
              <span class="material-symbols-outlined">call_end</span>
            </button>
            <button class="round-btn accept" @click="$emit('accept')" title="Принять">
              <span class="material-symbols-outlined">call</span>
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, watch, onBeforeUnmount } from 'vue'
import { useCallStore } from '@/stores/call.js'

const callStore = useCallStore()

defineEmits(['accept', 'decline'])

const show = computed(() => callStore.isIncoming)

const initiator = computed(() => (callStore.call?.participants || []).find(p => p.role === 'initiator'))
const isGroup = computed(() => callStore.call?.kind === 'group')

const primaryName = computed(() => initiator.value?.fio || 'Сотрудник')

const primaryAvatar = computed(() => {
  const path = initiator.value?.avatar_path
  if (path) return `/uploads/${path}`
  if (initiator.value?.user_id) return `/api/users/${initiator.value.user_id}/identicon`
  return null
})

const subtitle = computed(() => {
  if (isGroup.value) {
    const n = (callStore.call?.participants || []).length
    return `Групповой звонок · ${n} участников`
  }
  return callStore.media === 'audio' ? 'Аудиозвонок' : 'Видеозвонок'
})

const tagLabel = computed(() => callStore.media === 'audio' ? 'Входящий аудиозвонок' : 'Входящий видеозвонок')
const mediaIcon = computed(() => callStore.media === 'audio' ? 'call' : 'videocam')

/* Рингтон — двухтональный loop, пока показывается overlay.
   AudioContext'ы созданные ДО первого жеста пользователя оказываются в
   состоянии 'suspended' и звук молчит. Поэтому делаем 2 страховки:
   1) перед playOneRing() пытаемся resume();
   2) если контекст так и не разогрелся, при следующем pointerdown/keydown
      повторим попытку (одноразовый слушатель). */
let ringCtx = null
let ringTimer = null
let pendingGesture = null

function playOneRing() {
  if (!ringCtx) return
  try {
    if (ringCtx.state === 'suspended') ringCtx.resume()
    const now = ringCtx.currentTime
    const tones = [
      { freq: 520, start: 0,    dur: 0.35 },
      { freq: 660, start: 0.40, dur: 0.55 },
    ]
    tones.forEach(({ freq, start, dur }) => {
      const osc = ringCtx.createOscillator()
      const gain = ringCtx.createGain()
      osc.type = 'sine'
      osc.frequency.value = freq
      gain.gain.setValueAtTime(0, now + start)
      gain.gain.linearRampToValueAtTime(0.22, now + start + 0.05)
      gain.gain.exponentialRampToValueAtTime(0.0001, now + start + dur)
      osc.connect(gain).connect(ringCtx.destination)
      osc.start(now + start)
      osc.stop(now + start + dur + 0.02)
    })
  } catch {}
}

function installGestureRetry() {
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

function startRing() {
  if (ringTimer) return
  try {
    const Ctx = window.AudioContext || window.webkitAudioContext
    if (!Ctx) return
    ringCtx = new Ctx()
    // Если AudioContext «застрял» в suspended — попытаемся разбудить, а
    // также подвесим одноразовый retry на первый пользовательский жест.
    if (ringCtx.state === 'suspended') {
      ringCtx.resume().catch(() => {})
      installGestureRetry()
    }
  } catch { return }
  playOneRing()
  ringTimer = setInterval(() => playOneRing(), 1700)
}

function stopRing() {
  if (ringTimer) { clearInterval(ringTimer); ringTimer = null }
  if (ringCtx) { try { ringCtx.close() } catch {}; ringCtx = null }
  if (pendingGesture) {
    window.removeEventListener('pointerdown', pendingGesture, true)
    window.removeEventListener('keydown', pendingGesture, true)
    pendingGesture = null
  }
}

watch(show, (v) => {
  if (v) startRing()
  else stopRing()
}, { immediate: true })

onBeforeUnmount(stopRing)
</script>

<style scoped>
.incoming-overlay {
  position: fixed;
  inset: 0;
  background: color-mix(in oklch, var(--color-scrim) 100%, transparent);
  backdrop-filter: blur(12px);
  z-index: 12000;
  display: grid;
  place-items: center;
  padding: 16px;
}

.incoming-card {
  position: relative;
  max-width: 360px;
  width: 100%;
  padding: 32px 24px 24px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: 28px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  box-shadow: 0 24px 72px color-mix(in oklch, var(--color-scrim) 80%, transparent);
}

.incoming-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border-radius: 999px;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.incoming-tag .material-symbols-outlined { font-size: 16px; }

.incoming-avatar-wrap {
  position: relative;
  width: 120px;
  height: 120px;
  margin: 18px auto 12px;
  display: grid;
  place-items: center;
}

.incoming-avatar {
  position: relative;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  object-fit: cover;
  border: 3px solid var(--color-surface);
  box-shadow: 0 0 0 3px var(--color-primary-container);
  z-index: 1;
  background: var(--color-surface-high);
  display: grid;
  place-items: center;
}

.avatar-fallback .material-symbols-outlined {
  font-size: 56px;
  color: var(--color-on-primary-container);
}

.ring {
  position: absolute;
  border-radius: 50%;
  border: 2px solid var(--color-primary);
  inset: 0;
  opacity: 0;
  animation: ringExpand 2s ease-out infinite;
  pointer-events: none;
}

.ring-2 { animation-delay: 1s; }

@keyframes ringExpand {
  0%   { transform: scale(1);    opacity: 0.5; }
  100% { transform: scale(1.55); opacity: 0;   }
}

.incoming-name {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text);
  letter-spacing: -0.01em;
}

.incoming-sub {
  margin: 6px 0 24px;
  font-size: 14px;
  color: var(--color-text-dim);
}

.incoming-actions {
  display: flex;
  justify-content: space-around;
  gap: 32px;
  width: 100%;
  margin-top: 8px;
}

.round-btn {
  width: 66px;
  height: 66px;
  border-radius: 50%;
  border: 0;
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--color-on-primary);
  transition: transform 0.15s, box-shadow 0.15s;
  box-shadow: 0 8px 24px color-mix(in oklch, var(--color-scrim) 32%, transparent);
}

.round-btn:hover { transform: translateY(-2px); box-shadow: 0 12px 30px color-mix(in oklch, var(--color-scrim) 42%, transparent); }
.round-btn:active { transform: translateY(0); }

.round-btn.accept {
  background: var(--color-success);
  color: var(--color-on-success);
  animation: pickupPulse 1.6s ease-in-out infinite;
}

.round-btn.decline {
  background: var(--color-error);
  color: var(--color-on-error);
  transform: rotate(135deg);
}

.round-btn.decline:hover { transform: rotate(135deg) translateY(-2px); }

.round-btn .material-symbols-outlined { font-size: 28px; }

@keyframes pickupPulse {
  0%, 100% { box-shadow: 0 0 0 0 color-mix(in oklch, var(--color-success) 50%, transparent); }
  50%      { box-shadow: 0 0 0 12px color-mix(in oklch, var(--color-success) 0%, transparent); }
}

/* Анимация появления карточки */
.incoming-enter-active, .incoming-leave-active { transition: opacity 0.2s; }
.incoming-enter-from, .incoming-leave-to { opacity: 0; }

.incoming-enter-active .incoming-card,
.incoming-leave-active .incoming-card {
  transition: transform 0.25s cubic-bezier(0.22, 1, 0.36, 1);
}
.incoming-enter-from .incoming-card { transform: scale(0.92); }
.incoming-leave-to .incoming-card { transform: scale(0.95); }

@media (max-width: 480px) {
  .incoming-card { padding: 28px 18px 22px; }
  .incoming-avatar-wrap, .incoming-avatar { width: 100px; height: 100px; }
  .round-btn { width: 60px; height: 60px; }
}
</style>
