<template>
  <div
    class="tile"
    :class="{
      local: isLocal,
      no_video: !hasVideo,
      audio_off: !audio,
      speaking: speaking && !isLocal,
      screen: source === 'screen',
    }"
  >
    <!-- Видео всегда muted: звук всех удалённых участников воспроизводит
         CallAudioSink — он не зависит от того, какие плитки отрисованы. -->
    <video
      v-show="hasVideo"
      ref="videoEl"
      class="tile-video"
      autoplay
      playsinline
      muted
    />

    <div v-show="!hasVideo" class="tile-placeholder">
      <div class="tile-avatar">
        <img v-if="avatar" :src="avatar" :alt="name" />
        <span v-else class="material-symbols-outlined">person</span>
      </div>
      <div v-if="pending" class="tile-status">
        <span class="material-symbols-outlined spin">progress_activity</span>
        Ждём ответа…
      </div>
    </div>

    <div class="tile-footer">
      <span class="tile-name">{{ isLocal ? `${name} (Вы)` : name }}</span>
      <span v-if="guest" class="tile-guest">гость</span>
      <span v-if="!audio && !pending" class="material-symbols-outlined tile-icon">mic_off</span>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed, onMounted, onBeforeUnmount } from 'vue'
import { callRoom } from '@/services/livekit.js'

const props = defineProps({
  /** Identity участника в комнате LiveKit (для локального — своя). */
  identity: { type: String, default: null },
  name: { type: String, required: true },
  /** 'camera' | 'screen' — какой трек показывает плитка. */
  source: { type: String, default: 'camera' },
  audio: { type: Boolean, default: true },
  video: { type: Boolean, default: true },
  isLocal: { type: Boolean, default: false },
  avatar: { type: String, default: null },
  /** Приглашён, но ещё не вошёл в комнату. */
  pending: { type: Boolean, default: false },
  speaking: { type: Boolean, default: false },
  guest: { type: Boolean, default: false },
  /** Меняется при каждом изменении треков — триггер пере-attach. */
  tick: { type: Number, default: 0 },
})

const videoEl = ref(null)

const hasVideo = computed(() => props.video && !props.pending)

let attachedVideo = null

function attach() {
  const identity = props.isLocal ? callRoom.localIdentity : props.identity
  if (!identity) return

  const videoTrack = callRoom.getTrack(identity, props.source === 'screen' ? 'screen' : 'camera')
  if (attachedVideo && attachedVideo !== videoTrack && videoEl.value) {
    attachedVideo.detach(videoEl.value)
    attachedVideo = null
  }
  if (videoTrack && videoEl.value && attachedVideo !== videoTrack) {
    videoTrack.attach(videoEl.value)
    attachedVideo = videoTrack
  }
}

watch(() => [props.identity, props.tick, props.video], attach, { flush: 'post' })
onMounted(attach)
onBeforeUnmount(() => {
  if (attachedVideo && videoEl.value) attachedVideo.detach(videoEl.value)
})
</script>

<style scoped>
.tile {
  position: relative;
  border-radius: 20px;
  overflow: hidden;
  background: var(--color-surface-highest);
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 140px;
}

/* Активный спикер — подсветка рамкой в тон primary. */
.tile.speaking {
  box-shadow: inset 0 0 0 3px var(--color-primary);
}

.tile-video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  background: var(--color-surface-highest);
}

/* Демонстрация экрана — без обрезки, текст должен читаться. */
.tile.screen .tile-video {
  object-fit: contain;
  background: var(--color-scrim);
}

/* Локальное видео — зеркалим, как Zoom/Meet (но не демонстрацию экрана). */
.tile.local:not(.screen) .tile-video {
  transform: scaleX(-1);
}

.tile-placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: var(--color-surface-highest);
}

.tile-avatar {
  width: 96px;
  height: 96px;
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid;
  place-items: center;
  overflow: hidden;
  border: 2px solid var(--color-surface);
}

.tile-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.tile-avatar .material-symbols-outlined { font-size: 48px; }

.tile-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--color-text-dim);
  font-weight: 500;
}

.spin {
  animation: spinIcon 1.2s linear infinite;
  font-size: 18px;
}

@keyframes spinIcon {
  from { transform: rotate(0); }
  to { transform: rotate(360deg); }
}

.tile-footer {
  position: absolute;
  left: 12px;
  bottom: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: color-mix(in oklch, var(--color-scrim) 56%, transparent);
  color: oklch(1 0 0);
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  max-width: calc(100% - 24px);
  backdrop-filter: blur(8px);
}

.tile-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tile-guest {
  flex-shrink: 0;
  padding: 1px 8px;
  border-radius: 999px;
  background: color-mix(in oklch, var(--color-tertiary) 40%, transparent);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.tile-icon { font-size: 16px; }

.tile.audio_off .tile-icon { color: var(--color-error); }
</style>
