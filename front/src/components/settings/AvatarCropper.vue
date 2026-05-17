<template>
  <div class="avatar-cropper">

    <!-- Зона загрузки -->
    <div v-if="!imageSrc" class="upload-zone">
      <label class="upload-btn">
        <span class="material-symbols-outlined">photo_camera</span>
        Выбрать фото
        <input type="file" accept="image/jpeg,image/png" @change="onFileSelect" style="display:none" />
      </label>
      <p class="upload-hint">JPG или PNG, не более 10 МБ</p>
    </div>

    <!-- Редактор кропа -->
    <div v-else class="crop-zone">

      <!-- Холст с изображением -->
      <div
        class="crop-area"
        :style="{ width: displayW + 'px', height: displayH + 'px' }"
        @mousemove.prevent="onMouseMove"
        @mouseup="stopDrag"
        @mouseleave="stopDrag"
      >
        <img
          :src="imageSrc"
          :style="{ width: displayW + 'px', height: displayH + 'px' }"
          draggable="false"
          @load="onImageLoad"
        />

        <!-- Затемнение вокруг рамки (4 блока) -->
        <div class="ov ov-top"    :style="{ height: cropY + 'px' }" />
        <div class="ov ov-bottom" :style="{ top: cropY + cropSize + 'px' }" />
        <div class="ov ov-left"   :style="{ top: cropY + 'px', width: cropX + 'px', height: cropSize + 'px' }" />
        <div class="ov ov-right"  :style="{ top: cropY + 'px', left: cropX + cropSize + 'px', height: cropSize + 'px' }" />

        <!-- Рамка кропа -->
        <div
          class="crop-frame"
          :style="{ left: cropX + 'px', top: cropY + 'px', width: cropSize + 'px', height: cropSize + 'px' }"
          @mousedown.prevent="startDrag"
        >
          <div class="corner tl" /><div class="corner tr" />
          <div class="corner bl" /><div class="corner br" />
          <!-- Линии сетки -->
          <div class="grid-h" style="top:33.3%" /><div class="grid-h" style="top:66.6%" />
          <div class="grid-v" style="left:33.3%" /><div class="grid-v" style="left:66.6%" />
        </div>
      </div>

      <!-- Слайдер размера -->
      <div class="crop-controls">
        <span class="material-symbols-outlined ctrl-icon">crop</span>
        <input type="range" :min="MIN_CROP" :max="maxCropSize" v-model.number="cropSize" class="size-range" />
        <span class="ctrl-value">{{ cropSize }}px</span>
      </div>

      <!-- Кнопки -->
      <div class="crop-actions">
        <button class="btn-secondary" @click="reset">
          <span class="material-symbols-outlined">restart_alt</span>
          Другое фото
        </button>
        <button class="btn-secondary" @click="$emit('cancel')">Отмена</button>
        <button class="btn-primary" @click="confirmCrop" :disabled="confirming">
          {{ confirming ? 'Сохранение…' : 'Подтвердить' }}
        </button>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const MAX_W = 520
const MAX_H = 380
const MIN_CROP = 50
const TARGET_SIZE = 400

const emit = defineEmits(['cropped', 'cancel'])

const imageSrc = ref(null)
const naturalW  = ref(0)
const naturalH  = ref(0)
const displayW  = ref(0)
const displayH  = ref(0)
const cropX     = ref(0)
const cropY     = ref(0)
const cropSize  = ref(150)
const confirming = ref(false)

const maxCropSize = computed(() => Math.min(displayW.value, displayH.value))

// ─── drag ───────────────────────────────────────────
let dragging = false
let dragStartClientX = 0
let dragStartClientY = 0
let dragStartCropX   = 0
let dragStartCropY   = 0

function startDrag(e) {
  dragging = true
  dragStartClientX = e.clientX
  dragStartClientY = e.clientY
  dragStartCropX   = cropX.value
  dragStartCropY   = cropY.value
}

function onMouseMove(e) {
  if (!dragging) return
  const s = cropSize.value
  cropX.value = Math.max(0, Math.min(dragStartCropX + e.clientX - dragStartClientX, displayW.value - s))
  cropY.value = Math.max(0, Math.min(dragStartCropY + e.clientY - dragStartClientY, displayH.value - s))
}

function stopDrag() { dragging = false }

// ─── size change ─────────────────────────────────────
watch(cropSize, (s) => {
  cropX.value = Math.max(0, Math.min(cropX.value, displayW.value - s))
  cropY.value = Math.max(0, Math.min(cropY.value, displayH.value - s))
})

// ─── load ────────────────────────────────────────────
function onFileSelect(e) {
  const file = e.target.files[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = (ev) => { imageSrc.value = ev.target.result }
  reader.readAsDataURL(file)
}

function onImageLoad(e) {
  const img = e.target
  naturalW.value = img.naturalWidth
  naturalH.value = img.naturalHeight

  const scale = Math.min(MAX_W / naturalW.value, MAX_H / naturalH.value)
  displayW.value = Math.round(naturalW.value * scale)
  displayH.value = Math.round(naturalH.value * scale)

  const size = Math.max(MIN_CROP, Math.min(Math.round(Math.min(displayW.value, displayH.value) * 0.65), 220))
  cropSize.value = size
  cropX.value = Math.round((displayW.value - size) / 2)
  cropY.value = Math.round((displayH.value - size) / 2)
}

function reset() {
  imageSrc.value = null
  naturalW.value = naturalH.value = displayW.value = displayH.value = 0
  cropX.value = cropY.value = 0
  cropSize.value = 150
}

// ─── crop ────────────────────────────────────────────
async function confirmCrop() {
  confirming.value = true
  try {
    const img = new Image()
    img.src = imageSrc.value
    await new Promise(r => { img.onload = r })

    // scale: сколько натуральных пикселей на один отображаемый
    const scale = naturalW.value / displayW.value
    const srcX    = Math.round(cropX.value    * scale)
    const srcY    = Math.round(cropY.value    * scale)
    const srcSize = Math.round(cropSize.value * scale)

    const canvas = document.createElement('canvas')
    canvas.width  = TARGET_SIZE
    canvas.height = TARGET_SIZE
    canvas.getContext('2d').drawImage(img, srcX, srcY, srcSize, srcSize, 0, 0, TARGET_SIZE, TARGET_SIZE)

    let blob = await new Promise(r => canvas.toBlob(r, 'image/jpeg', 0.92))

    // Сжатие если превышает 2 МБ
    if (blob.size > 2 * 1024 * 1024) {
      let q = 0.75
      while (blob.size > 2 * 1024 * 1024 && q > 0.1) {
        blob = await new Promise(r => canvas.toBlob(r, 'image/jpeg', q))
        q -= 0.15
      }
    }

    emit('cropped', blob)
  } catch (err) {
    console.error('Crop error:', err)
  } finally {
    confirming.value = false
  }
}
</script>

<style scoped>
.avatar-cropper {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ── Upload ── */
.upload-zone {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 24px;
  border: 2px dashed var(--gw-border);
  border-radius: var(--gw-radius);
  background: var(--gw-bg);
}

.upload-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 24px;
  background: var(--gw-primary);
  color: var(--color-on-primary);
  border-radius: 10px;
  cursor: pointer;
  font-size: 15px;
  font-weight: 600;
  transition: background 0.15s;
}
.upload-btn:hover { background: var(--gw-primary-hover); }
.upload-btn .material-symbols-outlined { font-size: 22px; }

.upload-hint {
  font-size: 13px;
  color: var(--gw-text-secondary);
  margin: 0;
}

/* ── Crop zone ── */
.crop-zone {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
}

.crop-area {
  position: relative;
  overflow: hidden;
  border-radius: 8px;
  background: #0a0a0a;
  user-select: none;
  cursor: crosshair;
  flex-shrink: 0;
}

.crop-area img {
  display: block;
  pointer-events: none;
}

/* ── Overlay ── */
.ov {
  position: absolute;
  background: rgba(0, 0, 0, 0.58);
  pointer-events: none;
}
.ov-top    { top: 0; left: 0; right: 0; }
.ov-bottom { bottom: 0; left: 0; right: 0; }
.ov-left   { left: 0; }
.ov-right  { right: 0; }

/* ── Frame ── */
.crop-frame {
  position: absolute;
  border: 2px solid #fff;
  cursor: move;
  box-sizing: border-box;
}

.corner {
  position: absolute;
  width: 12px;
  height: 12px;
  background: #fff;
}
.corner.tl { top: -2px; left: -2px; }
.corner.tr { top: -2px; right: -2px; }
.corner.bl { bottom: -2px; left: -2px; }
.corner.br { bottom: -2px; right: -2px; }

.grid-h, .grid-v {
  position: absolute;
  background: rgba(255, 255, 255, 0.25);
  pointer-events: none;
}
.grid-h { left: 0; right: 0; height: 1px; }
.grid-v  { top: 0; bottom: 0; width: 1px; }

/* ── Controls ── */
.crop-controls {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  max-width: 520px;
}

.ctrl-icon {
  font-size: 20px;
  color: var(--gw-text-secondary);
  flex-shrink: 0;
}

.size-range {
  flex: 1;
  accent-color: var(--gw-primary);
}

.ctrl-value {
  font-size: 13px;
  color: var(--gw-text-secondary);
  width: 48px;
  text-align: right;
}

/* ── Actions ── */
.crop-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  width: 100%;
  max-width: 520px;
}

.btn-secondary {
  display: flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  color: var(--gw-text-secondary);
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  padding: 8px 16px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.btn-secondary:hover { background: var(--gw-bg); color: var(--gw-text); }
.btn-secondary .material-symbols-outlined { font-size: 16px; }

.btn-primary {
  background: var(--gw-primary);
  color: var(--color-on-primary);
  border: none;
  border-radius: 8px;
  padding: 8px 20px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}
.btn-primary:hover:not(:disabled) { background: var(--gw-primary-hover); }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
