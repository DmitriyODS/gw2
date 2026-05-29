<template>
  <div class="tile" :class="{ local: isLocal, no_video: !videoEnabled || !stream, audio_off: !audioEnabled }">
    <!-- Видео всегда muted: звук удалённого участника воспроизводит отдельный
         <audio> ниже. Так аудио не пропадает, когда собеседник выключает
         камеру (тогда <video> прячется через v-show, но звук продолжает идти). -->
    <video
      v-show="videoEnabled && stream"
      ref="videoEl"
      class="tile-video"
      autoplay
      playsinline
      muted
    />
    <!-- Аудио удалённого участника на отдельном элементе (для своего потока не
         нужно — себя не слушаем). -->
    <audio v-if="!isLocal" ref="audioEl" autoplay playsinline />

    <div v-show="!videoEnabled || !stream" class="tile-placeholder">
      <div class="tile-avatar">
        <img v-if="avatar" :src="avatar" :alt="name" />
        <span v-else class="material-symbols-outlined">person</span>
      </div>
      <div v-if="pending" class="tile-status">
        <span class="material-symbols-outlined spin">progress_activity</span>
        {{ connLabel }}
      </div>
    </div>

    <div class="tile-footer">
      <span class="tile-name">{{ isLocal ? `${name} (Вы)` : name }}</span>
      <span v-if="!audioEnabled" class="material-symbols-outlined tile-icon">mic_off</span>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, computed } from 'vue'

const props = defineProps({
  name: { type: String, required: true },
  stream: { type: Object, default: null },
  /* Меняется при каждом ontrack — поднимает watch, даже когда сам объект
     MediaStream остался прежним (его наполняют новыми треками по мере того,
     как WebRTC «прорастает»). Без этого watch на stream срабатывает только
     один раз и второй трек (видео) тихо не подхватится. */
  streamTick: { type: Number, default: 0 },
  audioEnabled: { type: Boolean, default: true },
  videoEnabled: { type: Boolean, default: true },
  isLocal: { type: Boolean, default: false },
  avatar: { type: String, default: null },
  pending: { type: Boolean, default: false },
  /* Состояние RTCPeerConnection.connectionState — для диагностики прямо на
     плитке: видно, встало ли соединение или зависло на подключении (тогда,
     скорее всего, нужен TURN). */
  connState: { type: String, default: null },
})

const videoEl = ref(null)
const audioEl = ref(null)

const connLabel = computed(() => {
  switch (props.connState) {
    case 'connected': return 'Соединение есть, ждём видео…'
    case 'failed': return 'Не удалось соединиться (нужен TURN)'
    case 'disconnected': return 'Связь прервалась…'
    default: return 'Подключается…'
  }
})

function bindEl(el, stream, force) {
  if (!el || !stream) return
  // Принудительное переподключение: когда трек добавляется в уже привязанный
  // MediaStream (тот же объект), некоторые браузеры не начинают его
  // воспроизводить без повторного присвоения srcObject. На смену streamTick
  // (force) обнуляем и заново ставим источник — гарантированный re-render.
  if (force && el.srcObject) el.srcObject = null
  if (el.srcObject !== stream) el.srcObject = stream
  // play() возвращает Promise, безопасно игнорируем отказ (autoplay-политика).
  el.play?.().catch(() => {})
}

let lastTick = 0
function attach() {
  const force = props.streamTick !== lastTick
  lastTick = props.streamTick
  // Видео — без звука (см. шаблон).
  bindEl(videoEl.value, props.stream, force)
  // Аудио удалённого участника — на отдельном элементе, чтобы звук шёл даже
  // при выключенной камере. play() уже разрешён жестом accept/«позвонить».
  if (!props.isLocal) bindEl(audioEl.value, props.stream, force)
}

// streamTick меняется при каждом ontrack — пере-attach подхватывает поздно
// прибывший трек (видео/аудио приходят независимо).
watch(() => [props.stream, props.streamTick], attach, { flush: 'post' })
onMounted(attach)
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

.tile-video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  background: var(--color-surface-highest);
}

/* Локальное видео — зеркалим, как Zoom/Meet */
.tile.local .tile-video {
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

.tile-icon { font-size: 16px; }

.tile.audio_off .tile-icon { color: var(--color-error); }
</style>
