<template>
  <div class="avatar-cropper">

    <!-- Зона загрузки -->
    <div v-if="!loaded" class="upload-zone">
      <label class="upload-btn">
        <span class="material-symbols-outlined">photo_camera</span>
        Выбрать фото
        <input type="file" accept="image/jpeg,image/png" @change="onFileSelect" style="display:none" />
      </label>
      <p class="upload-hint">JPG или PNG, не более 10 МБ</p>
    </div>

    <!-- Редактор кропа -->
    <div v-else class="crop-zone">

      <div class="crop-toolbar">
        <button type="button" class="tool-btn" title="Повернуть влево" @click="rotate(-90)">
          <span class="material-symbols-outlined">rotate_left</span>
        </button>
        <button type="button" class="tool-btn" title="Повернуть вправо" @click="rotate(90)">
          <span class="material-symbols-outlined">rotate_right</span>
        </button>
        <button type="button" class="tool-btn" title="Отразить по горизонтали" @click="flip">
          <span class="material-symbols-outlined">swap_horiz</span>
        </button>
        <span class="tool-hint">Перемещайте рамку и тяните за углы</span>
        <button type="button" class="tool-btn tool-reset" title="Сбросить" @click="resetEdits">
          <span class="material-symbols-outlined">restart_alt</span>
        </button>
      </div>

      <!-- Сцена: canvas + затемнение + квадратная рамка 1:1 -->
      <div ref="stageEl" class="crop-stage">
        <canvas ref="viewEl" class="crop-canvas" />
        <div class="ov" :style="shadeStyle('top')" />
        <div class="ov" :style="shadeStyle('bottom')" />
        <div class="ov" :style="shadeStyle('left')" />
        <div class="ov" :style="shadeStyle('right')" />
        <div class="crop-frame" :style="frameStyle" @pointerdown.prevent="startDrag('move', $event)">
          <div class="crop-circle" />
          <span
            v-for="h in ['nw', 'ne', 'sw', 'se']"
            :key="h"
            class="handle"
            :class="h"
            @pointerdown.stop.prevent="startDrag(h, $event)"
          />
        </div>
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
// Редактор аватарки: квадратный кроп 1:1 (перемещение рамки + угловые ручки),
// повороты на 90° и отражение «запекаются» в offscreen-канву. Драг — на
// pointer-событиях с touch-action:none, поэтому на тач-экранах двигается
// рамка, а не страница за диалогом.
import { computed, nextTick, ref } from 'vue'

const MIN_CROP = 48       // в базовых (натуральных) пикселях
const TARGET_SIZE = 400

const emit = defineEmits(['cropped', 'cancel'])

const stageEl = ref(null)
const viewEl = ref(null)
const loaded = ref(false)
const confirming = ref(false)

let base = null           // offscreen-canvas с текущим (повёрнутым/отражённым) изображением
let original = null       // исходное изображение для сброса
const scale = ref(1)      // базовые координаты → экранные
const crop = ref({ x: 0, y: 0, size: 0 }) // квадрат в базовых координатах

function onFileSelect(e) {
  const file = e.target.files[0]
  if (!file) return
  const url = URL.createObjectURL(file)
  const img = new Image()
  img.onload = async () => {
    URL.revokeObjectURL(url)
    original = img
    rebuildBase()
    loaded.value = true
    await nextTick()
    redraw()
  }
  img.onerror = () => URL.revokeObjectURL(url)
  img.src = url
}

function rebuildBase() {
  base = document.createElement('canvas')
  base.width = original.naturalWidth
  base.height = original.naturalHeight
  base.getContext('2d').drawImage(original, 0, 0)
  centerCrop()
}

function centerCrop() {
  const size = Math.round(Math.min(base.width, base.height) * 0.8)
  crop.value = {
    x: Math.round((base.width - size) / 2),
    y: Math.round((base.height - size) / 2),
    size,
  }
}

function redraw() {
  const view = viewEl.value
  const stage = stageEl.value
  if (!view || !stage || !base) return
  const maxW = stage.clientWidth || 480
  const maxH = Math.round(window.innerHeight * 0.45)
  const k = Math.min(maxW / base.width, maxH / base.height, 1)
  scale.value = k
  view.width = Math.max(1, Math.round(base.width * k))
  view.height = Math.max(1, Math.round(base.height * k))
  view.getContext('2d').drawImage(base, 0, 0, view.width, view.height)
}

function rotate(deg) {
  if (!base) return
  const next = document.createElement('canvas')
  next.width = base.height
  next.height = base.width
  const ctx = next.getContext('2d')
  ctx.translate(next.width / 2, next.height / 2)
  ctx.rotate((deg * Math.PI) / 180)
  ctx.drawImage(base, -base.width / 2, -base.height / 2)
  base = next
  centerCrop()
  redraw()
}

function flip() {
  if (!base) return
  const next = document.createElement('canvas')
  next.width = base.width
  next.height = base.height
  const ctx = next.getContext('2d')
  ctx.translate(next.width, 0)
  ctx.scale(-1, 1)
  ctx.drawImage(base, 0, 0)
  base = next
  redraw()
}

function resetEdits() {
  if (!original) return
  rebuildBase()
  redraw()
}

function reset() {
  loaded.value = false
  base = null
  original = null
  crop.value = { x: 0, y: 0, size: 0 }
}

// ── Рамка и затемнение (экранные координаты от базовых через scale) ──
const frameStyle = computed(() => {
  const k = scale.value
  const c = crop.value
  return {
    left: c.x * k + 'px',
    top: c.y * k + 'px',
    width: c.size * k + 'px',
    height: c.size * k + 'px',
  }
})

function shadeStyle(side) {
  const k = scale.value
  const c = crop.value
  const W = (base?.width || 0) * k
  const H = (base?.height || 0) * k
  const x = c.x * k, y = c.y * k, s = c.size * k
  switch (side) {
    case 'top': return { left: 0, top: 0, width: W + 'px', height: y + 'px' }
    case 'bottom': return { left: 0, top: y + s + 'px', width: W + 'px', height: Math.max(0, H - y - s) + 'px' }
    case 'left': return { left: 0, top: y + 'px', width: x + 'px', height: s + 'px' }
    default: return { left: x + s + 'px', top: y + 'px', width: Math.max(0, W - x - s) + 'px', height: s + 'px' }
  }
}

// ── Драг: перемещение рамки и угловые ручки с сохранением 1:1 ──
let drag = null

function startDrag(mode, e) {
  drag = { mode, startX: e.clientX, startY: e.clientY, start: { ...crop.value } }
  window.addEventListener('pointermove', onDrag)
  window.addEventListener('pointerup', endDrag, { once: true })
}

function onDrag(e) {
  if (!drag || !base) return
  const k = scale.value
  const dx = (e.clientX - drag.startX) / k
  const dy = (e.clientY - drag.startY) / k
  const s = drag.start

  if (drag.mode === 'move') {
    crop.value = {
      x: Math.min(Math.max(s.x + dx, 0), base.width - s.size),
      y: Math.min(Math.max(s.y + dy, 0), base.height - s.size),
      size: s.size,
    }
    return
  }

  // Угловая ручка: противоположный угол зафиксирован, квадрат растёт по
  // среднему смещению вдоль обеих осей.
  const sx = drag.mode.includes('e') ? 1 : -1
  const sy = drag.mode.includes('s') ? 1 : -1
  const anchorX = sx === 1 ? s.x : s.x + s.size
  const anchorY = sy === 1 ? s.y : s.y + s.size
  const maxSize = Math.min(
    sx === 1 ? base.width - anchorX : anchorX,
    sy === 1 ? base.height - anchorY : anchorY,
  )
  const size = Math.round(Math.min(Math.max(s.size + (sx * dx + sy * dy) / 2, MIN_CROP), maxSize))
  crop.value = {
    x: sx === 1 ? anchorX : anchorX - size,
    y: sy === 1 ? anchorY : anchorY - size,
    size,
  }
}

function endDrag() {
  drag = null
  window.removeEventListener('pointermove', onDrag)
}

// ── Результат ──
async function confirmCrop() {
  if (!base) return
  confirming.value = true
  try {
    const c = crop.value
    const canvas = document.createElement('canvas')
    canvas.width = TARGET_SIZE
    canvas.height = TARGET_SIZE
    canvas.getContext('2d').drawImage(base, c.x, c.y, c.size, c.size, 0, 0, TARGET_SIZE, TARGET_SIZE)

    let blob = await new Promise(r => canvas.toBlob(r, 'image/jpeg', 0.92))
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
  border: 2px dashed var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
}

.upload-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 24px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  border-radius: 10px;
  cursor: pointer;
  font-size: 15px;
  font-weight: 600;
  transition: background 0.15s;
}
.upload-btn:hover { background: var(--color-primary-hover); }
.upload-btn .material-symbols-outlined { font-size: 22px; }

.upload-hint {
  font-size: 13px;
  color: var(--color-text-dim);
  margin: 0;
}

/* ── Crop zone ── */
.crop-zone {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.crop-toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.tool-btn {
  height: 34px;
  min-width: 34px;
  padding: 0 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-surface-low);
  color: var(--color-text);
  cursor: pointer;
}
.tool-btn:hover { background: var(--color-surface-high); }
.tool-btn .material-symbols-outlined { font-size: 19px; }
.tool-reset { margin-left: auto; color: var(--color-text-dim); }

.tool-hint {
  font-size: 12px;
  color: var(--color-text-dim);
  margin-left: 6px;
}

.crop-stage {
  position: relative;
  display: inline-block;
  max-width: 100%;
  line-height: 0;
  /* палец двигает рамку, а не страницу за диалогом */
  touch-action: none;
  user-select: none;
  -webkit-user-select: none;
  align-self: center;
  margin: 0 auto;
}

.crop-canvas {
  display: block;
  max-width: 100%;
  border-radius: var(--radius-md);
}

/* ── Overlay ── */
.ov {
  position: absolute;
  background: color-mix(in oklch, var(--color-surface) 60%, transparent);
  pointer-events: none;
}

/* ── Frame ── */
.crop-frame {
  position: absolute;
  border: 2px solid var(--color-primary);
  cursor: move;
  box-sizing: border-box;
}

.crop-circle {
  position: absolute;
  inset: 0;
  border: 1.5px dashed color-mix(in oklch, var(--color-on-primary, white) 70%, transparent);
  border-radius: 50%;
  pointer-events: none;
}

.handle {
  position: absolute;
  width: 18px;
  height: 18px;
  background: var(--color-primary);
  border: 2px solid var(--color-on-primary, var(--color-surface));
  border-radius: 50%;
  box-sizing: border-box;
}
.handle.nw { left: -9px; top: -9px; cursor: nwse-resize; }
.handle.ne { right: -9px; top: -9px; cursor: nesw-resize; }
.handle.sw { left: -9px; bottom: -9px; cursor: nesw-resize; }
.handle.se { right: -9px; bottom: -9px; cursor: nwse-resize; }

/* ── Actions ── */
.crop-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  width: 100%;
  flex-wrap: wrap;
}

.btn-secondary {
  display: flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  color: var(--color-text-dim);
  border: 1px solid var(--color-outline-dim);
  border-radius: 8px;
  padding: 8px 16px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.btn-secondary:hover { background: var(--color-surface-low); color: var(--color-text); }
.btn-secondary .material-symbols-outlined { font-size: 16px; }

.btn-primary {
  background: var(--color-primary);
  color: var(--color-on-primary);
  border: none;
  border-radius: 8px;
  padding: 8px 20px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}
.btn-primary:hover:not(:disabled) { background: var(--color-primary-hover); }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
