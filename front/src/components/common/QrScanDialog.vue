<template>
  <AppDialog
    :model-value="modelValue"
    icon="qr_code_scanner"
    size="sm"
    :title="title"
    :subtitle="subtitle"
    @update:modelValue="close"
  >
    <div class="scan-body">
      <div class="scan-view">
        <video ref="videoEl" class="scan-video" playsinline muted></video>
        <div class="scan-frame"></div>
      </div>
      <p v-if="error" class="scan-error">{{ error }}</p>
      <p v-else class="scan-hint">{{ hint }}</p>
      <slot name="footer-note" />
    </div>
  </AppDialog>
</template>

<script setup>
import { ref, watch, onBeforeUnmount } from 'vue'
import jsQR from 'jsqr'
import AppDialog from '@/components/common/AppDialog.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: 'Сканирование QR' },
  subtitle: { type: String, default: '' },
  hint: { type: String, default: 'Держите код в рамке — распознаётся автоматически.' },
  // Фильтр-преобразователь распознанного текста: вернул пусто — код не наш,
  // сканирование продолжается. По умолчанию годится любой непустой QR.
  decode: { type: Function, default: (raw) => raw },
  // Закрывать диалог после успешного распознавания.
  closeOnDecode: { type: Boolean, default: true },
})
const emit = defineEmits(['update:modelValue', 'decoded'])

const videoEl = ref(null)
const error = ref('')

let stream = null
let rafId = null
let canvas = null
let ctx = null

async function startCamera() {
  error.value = ''
  if (!navigator.mediaDevices?.getUserMedia) {
    error.value = 'Камера недоступна на этом устройстве.'
    return
  }
  try {
    stream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: 'environment' },
      audio: false,
    })
    const video = videoEl.value
    if (!video) return
    video.srcObject = stream
    await video.play()
    canvas = document.createElement('canvas')
    ctx = canvas.getContext('2d', { willReadFrequently: true })
    scanLoop()
  } catch {
    error.value = 'Не удалось получить доступ к камере. Разрешите доступ в настройках браузера.'
  }
}

function scanLoop() {
  const video = videoEl.value
  if (!video || video.readyState !== video.HAVE_ENOUGH_DATA) {
    rafId = requestAnimationFrame(scanLoop)
    return
  }
  canvas.width = video.videoWidth
  canvas.height = video.videoHeight
  ctx.drawImage(video, 0, 0, canvas.width, canvas.height)
  const img = ctx.getImageData(0, 0, canvas.width, canvas.height)
  const found = jsQR(img.data, img.width, img.height, { inversionAttempts: 'dontInvert' })
  if (found?.data) {
    const value = props.decode(found.data)
    if (value) {
      emit('decoded', value)
      if (props.closeOnDecode) { close(); return }
    }
  }
  rafId = requestAnimationFrame(scanLoop)
}

function stopCamera() {
  if (rafId) { cancelAnimationFrame(rafId); rafId = null }
  if (stream) {
    stream.getTracks().forEach((t) => t.stop())
    stream = null
  }
}

function close() {
  stopCamera()
  emit('update:modelValue', false)
}

watch(
  () => props.modelValue,
  (open) => { if (open) startCamera(); else stopCamera() },
)

onBeforeUnmount(stopCamera)
</script>

<style scoped>
.scan-body { display: flex; flex-direction: column; gap: 12px; align-items: center; }
.scan-view {
  position: relative;
  width: 100%;
  max-width: 300px;
  aspect-ratio: 1;
  border-radius: var(--radius-lg, 16px);
  overflow: hidden;
  background: #000;
}
.scan-video { width: 100%; height: 100%; object-fit: cover; }
.scan-frame {
  position: absolute;
  inset: 14%;
  border: 3px solid rgba(255, 255, 255, 0.85);
  border-radius: var(--radius-md, 12px);
  box-shadow: 0 0 0 100vmax rgba(0, 0, 0, 0.25);
}
.scan-hint { font-size: 0.85rem; color: var(--color-text-secondary); text-align: center; }
.scan-error { font-size: 0.85rem; color: var(--color-error); text-align: center; }
</style>
