<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="crop_rotate"
    size="lg"
    title="Редактировать изображение"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: 'Применить', icon: 'check' },
    ]"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="apply"
  >
    <div class="ied-toolbar">
      <button type="button" class="ied-btn" title="Повернуть влево" @click="rotate(-90)">
        <span class="material-symbols-outlined">rotate_left</span>
      </button>
      <button type="button" class="ied-btn" title="Повернуть вправо" @click="rotate(90)">
        <span class="material-symbols-outlined">rotate_right</span>
      </button>
      <button type="button" class="ied-btn" title="Отразить по горизонтали" @click="flip">
        <span class="material-symbols-outlined">swap_horiz</span>
      </button>
      <span class="ied-hint">Рамка — область обрезки: тяните углы или перемещайте её</span>
      <button type="button" class="ied-btn ied-reset" title="Сбросить всё" @click="resetAll">
        <span class="material-symbols-outlined">restart_alt</span>
        Сбросить
      </button>
    </div>

    <div ref="stageEl" class="ied-stage">
      <canvas ref="viewEl" class="ied-canvas" />
      <!-- Затемнение вокруг рамки + рамка с ручками -->
      <div class="ied-shade" :style="shadeStyle('top')" />
      <div class="ied-shade" :style="shadeStyle('bottom')" />
      <div class="ied-shade" :style="shadeStyle('left')" />
      <div class="ied-shade" :style="shadeStyle('right')" />
      <div
        class="ied-crop"
        :style="cropStyle"
        @pointerdown.prevent="startDrag('move', $event)"
      >
        <span
          v-for="h in ['nw', 'ne', 'sw', 'se']"
          :key="h"
          class="ied-handle"
          :class="h"
          @pointerdown.stop.prevent="startDrag(h, $event)"
        />
      </div>
    </div>
  </AppDialog>
</template>

<script setup>
// Лёгкий canvas-редактор картинки перед загрузкой: обрезка (рамка с
// угловыми ручками), повороты на 90°, отражение. Повороты/отражения
// «запекаются» в offscreen-канву сразу, обрезка применяется на «Применить».
import { computed, nextTick, ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  file: { type: File, default: null },
})
const emit = defineEmits(['update:modelValue', 'apply'])

const stageEl = ref(null)
const viewEl = ref(null)

let base = null            // offscreen-canvas с текущим (повёрнутым) изображением
let original = null        // исходный ImageBitmap/HTMLImageElement для сброса
const scale = ref(1)       // отображение → базовые координаты
const crop = ref({ x: 0, y: 0, w: 0, h: 0 }) // в базовых координатах

const MIN_CROP = 32

watch(() => props.modelValue, async (v) => {
  if (!v || !props.file) return
  await load()
})

async function load() {
  const url = URL.createObjectURL(props.file)
  try {
    original = await new Promise((resolve, reject) => {
      const img = new Image()
      img.onload = () => resolve(img)
      img.onerror = reject
      img.src = url
    })
  } finally {
    URL.revokeObjectURL(url)
  }
  base = document.createElement('canvas')
  base.width = original.naturalWidth
  base.height = original.naturalHeight
  base.getContext('2d').drawImage(original, 0, 0)
  crop.value = { x: 0, y: 0, w: base.width, h: base.height }
  await nextTick()
  redraw()
}

function redraw() {
  const view = viewEl.value
  const stage = stageEl.value
  if (!view || !stage || !base) return
  const maxW = stage.clientWidth || 600
  const maxH = Math.round(window.innerHeight * 0.5)
  const k = Math.min(maxW / base.width, maxH / base.height, 1)
  scale.value = k
  view.width = Math.round(base.width * k)
  view.height = Math.round(base.height * k)
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
  crop.value = { x: 0, y: 0, w: base.width, h: base.height }
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

function resetAll() {
  if (!original) return
  base = document.createElement('canvas')
  base.width = original.naturalWidth
  base.height = original.naturalHeight
  base.getContext('2d').drawImage(original, 0, 0)
  crop.value = { x: 0, y: 0, w: base.width, h: base.height }
  redraw()
}

// ── Рамка обрезки (стили от базовых координат через scale) ──
const cropStyle = computed(() => {
  const k = scale.value
  const c = crop.value
  return {
    left: c.x * k + 'px',
    top: c.y * k + 'px',
    width: c.w * k + 'px',
    height: c.h * k + 'px',
  }
})

function shadeStyle(side) {
  const k = scale.value
  const c = crop.value
  const W = (base?.width || 0) * k
  const H = (base?.height || 0) * k
  const x = c.x * k, y = c.y * k, w = c.w * k, h = c.h * k
  switch (side) {
    case 'top': return { left: 0, top: 0, width: W + 'px', height: y + 'px' }
    case 'bottom': return { left: 0, top: y + h + 'px', width: W + 'px', height: Math.max(0, H - y - h) + 'px' }
    case 'left': return { left: 0, top: y + 'px', width: x + 'px', height: h + 'px' }
    default: return { left: x + w + 'px', top: y + 'px', width: Math.max(0, W - x - w) + 'px', height: h + 'px' }
  }
}

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
  let { x, y, w, h } = s
  const clampX = (v) => Math.min(Math.max(v, 0), base.width)
  const clampY = (v) => Math.min(Math.max(v, 0), base.height)

  if (drag.mode === 'move') {
    x = Math.min(Math.max(s.x + dx, 0), base.width - s.w)
    y = Math.min(Math.max(s.y + dy, 0), base.height - s.h)
  } else {
    // Угловые ручки: противоположный угол зафиксирован.
    const x2 = s.x + s.w
    const y2 = s.y + s.h
    let nx1 = s.x, ny1 = s.y, nx2 = x2, ny2 = y2
    if (drag.mode.includes('w')) nx1 = clampX(s.x + dx)
    if (drag.mode.includes('e')) nx2 = clampX(x2 + dx)
    if (drag.mode.includes('n')) ny1 = clampY(s.y + dy)
    if (drag.mode.includes('s')) ny2 = clampY(y2 + dy)
    x = Math.min(nx1, nx2 - MIN_CROP)
    y = Math.min(ny1, ny2 - MIN_CROP)
    w = Math.max(MIN_CROP, nx2 - x)
    h = Math.max(MIN_CROP, ny2 - y)
    if (drag.mode.includes('w')) { x = Math.min(nx1, x2 - MIN_CROP); w = x2 - x }
    if (drag.mode.includes('n')) { y = Math.min(ny1, y2 - MIN_CROP); h = y2 - y }
  }
  crop.value = { x, y, w, h }
}

function endDrag() {
  drag = null
  window.removeEventListener('pointermove', onDrag)
}

async function apply() {
  if (!base) return
  const c = crop.value
  const out = document.createElement('canvas')
  out.width = Math.round(c.w)
  out.height = Math.round(c.h)
  out.getContext('2d').drawImage(base, c.x, c.y, c.w, c.h, 0, 0, out.width, out.height)
  const type = props.file.type === 'image/png' ? 'image/png' : 'image/jpeg'
  const blob = await new Promise((resolve) => out.toBlob(resolve, type, 0.92))
  if (!blob) return
  emit('apply', new File([blob], props.file.name, { type }))
  emit('update:modelValue', false)
}
</script>

<style scoped>
.ied-toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}

.ied-btn {
  height: 34px;
  min-width: 34px;
  padding: 0 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-surface-low);
  color: var(--color-text);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}
.ied-btn:hover { background: var(--color-surface-high); }
.ied-btn .material-symbols-outlined { font-size: 19px; }
.ied-reset { margin-left: auto; color: var(--color-text-dim); }

.ied-hint {
  font-size: 12px;
  color: var(--color-text-dim);
  margin-left: 6px;
}

.ied-stage {
  position: relative;
  display: inline-block;
  max-width: 100%;
  line-height: 0;
  touch-action: none;
  user-select: none;
  align-self: center;
  margin: 0 auto;
}

.ied-canvas {
  display: block;
  max-width: 100%;
  border-radius: var(--radius-md);
}

.ied-shade {
  position: absolute;
  background: color-mix(in oklch, var(--color-surface) 55%, transparent);
  pointer-events: none;
}

.ied-crop {
  position: absolute;
  border: 2px solid var(--color-primary);
  border-radius: 2px;
  cursor: move;
  box-sizing: border-box;
}

.ied-handle {
  position: absolute;
  width: 16px;
  height: 16px;
  background: var(--color-primary);
  border: 2px solid var(--color-on-primary, var(--color-surface));
  border-radius: 50%;
  box-sizing: border-box;
}
.ied-handle.nw { left: -8px; top: -8px; cursor: nwse-resize; }
.ied-handle.ne { right: -8px; top: -8px; cursor: nesw-resize; }
.ied-handle.sw { left: -8px; bottom: -8px; cursor: nesw-resize; }
.ied-handle.se { right: -8px; bottom: -8px; cursor: nwse-resize; }
</style>
