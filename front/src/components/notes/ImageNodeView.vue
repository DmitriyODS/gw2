<template>
  <NodeViewWrapper
    as="div"
    class="rimg-block"
    :style="{ textAlign: node.attrs.align || 'left' }"
  >
    <span
      ref="holder"
      class="rimg-holder"
      :class="{ selected: selected && editable }"
      :style="{ width: displayWidth }"
      contenteditable="false"
    >
      <img ref="img" :src="node.attrs.src" :alt="node.attrs.alt || ''" class="rimg" draggable="false" />

      <!-- Угловая ручка ресайза (курсор/палец через Pointer Events) -->
      <span
        v-if="selected && editable"
        class="rimg-grip"
        title="Потяните, чтобы изменить размер"
        @pointerdown="startResize"
      />
    </span>

    <!-- Панель управления: на уровне блока (его ширина не меняется при
         ресайзе), поэтому не дёргается вслед за уменьшающейся картинкой. -->
    <span v-if="selected && editable" class="rimg-bar" contenteditable="false">
      <button type="button" class="rimg-btn" :class="{ on: align === 'left' }" title="Слева" @mousedown.prevent @click="setAlign('left')">
        <span class="material-symbols-outlined">format_align_left</span>
      </button>
      <button type="button" class="rimg-btn" :class="{ on: align === 'center' }" title="По центру" @mousedown.prevent @click="setAlign('center')">
        <span class="material-symbols-outlined">format_align_center</span>
      </button>
      <button type="button" class="rimg-btn" :class="{ on: align === 'right' }" title="Справа" @mousedown.prevent @click="setAlign('right')">
        <span class="material-symbols-outlined">format_align_right</span>
      </button>
      <span class="rimg-sep" />
      <input
        class="rimg-range"
        type="range" min="15" max="100" step="1"
        :value="widthPct"
        title="Размер"
        @mousedown.stop @touchstart.stop @pointerdown.stop
        @input="onSlider"
      />
      <span class="rimg-sep" />
      <button type="button" class="rimg-btn" title="Удалить" @mousedown.prevent @click="deleteNode()">
        <span class="material-symbols-outlined">delete</span>
      </button>
    </span>
  </NodeViewWrapper>
</template>

<script setup>
import { computed, ref } from 'vue'
import { NodeViewWrapper } from '@tiptap/vue-3'

// Пропсы, которые прокидывает VueNodeViewRenderer.
const props = defineProps({
  editor: { type: Object, required: true },
  node: { type: Object, required: true },
  updateAttributes: { type: Function, required: true },
  deleteNode: { type: Function, required: true },
  selected: { type: Boolean, default: false },
  getPos: { type: Function, default: null },
  extension: { type: Object, default: null },
  decorations: { type: Array, default: () => [] },
})

// Ширина хранится как процент ширины блока ('50%'); null — натуральная.
const live = ref('') // текущая ширина во время перетаскивания
const holder = ref(null)
const img = ref(null)

const editable = computed(() => props.editor?.isEditable)
const align = computed(() => props.node.attrs.align || 'left')
const displayWidth = computed(() => live.value || props.node.attrs.width || 'auto')
const widthPct = computed(() => {
  const w = live.value || props.node.attrs.width
  const m = w && /^(\d+)%$/.exec(w)
  return m ? Number(m[1]) : 100
})

function blockWidth() {
  // Ширина блочной обёртки (в неё вписан контент редактора) — база для %.
  return holder.value?.parentElement?.offsetWidth || holder.value?.offsetWidth || 1
}

function setAlign(a) {
  props.updateAttributes({ align: a })
}

function onSlider(e) {
  props.updateAttributes({ width: `${e.target.value}%` })
}

// ── Ресайз угловой ручкой (мышь и тач унифицированы Pointer Events) ──
let startX = 0
let startPx = 0
let baseW = 1

function startResize(e) {
  e.preventDefault()
  e.stopPropagation()
  startX = e.clientX
  startPx = holder.value?.offsetWidth || 0
  baseW = blockWidth()
  e.target.setPointerCapture?.(e.pointerId)
  window.addEventListener('pointermove', onResize)
  window.addEventListener('pointerup', endResize)
  window.addEventListener('pointercancel', endResize)
}

function onResize(e) {
  const px = startPx + (e.clientX - startX)
  const pct = Math.max(15, Math.min(100, Math.round((px / baseW) * 100)))
  live.value = `${pct}%`
}

function endResize() {
  window.removeEventListener('pointermove', onResize)
  window.removeEventListener('pointerup', endResize)
  window.removeEventListener('pointercancel', endResize)
  if (live.value) props.updateAttributes({ width: live.value })
  live.value = ''
}
</script>

<style scoped>
.rimg-block { position: relative; margin: 8px 0; }
.rimg-holder {
  position: relative;
  display: inline-block;
  max-width: 100%;
  line-height: 0;
  border-radius: var(--radius-md);
}
.rimg { width: 100%; height: auto; border-radius: var(--radius-md); display: block; }
.rimg-holder.selected { outline: 2px solid var(--color-primary); outline-offset: 2px; }

/* Панель — по центру блока (ширина блока постоянна), поэтому не смещается
   при изменении размера картинки. */
.rimg-bar {
  position: absolute;
  top: 0;
  left: 50%;
  transform: translate(-50%, -50%);
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 6px;
  border-radius: var(--radius-full);
  background: var(--color-surface);
  border: 1px solid var(--acrylic-border);
  box-shadow: var(--shadow-md);
  z-index: 3;
  line-height: 1;
  white-space: nowrap;
}
.rimg-btn {
  display: grid;
  place-items: center;
  /* min/max по обеим осям равны размеру — иначе глобальный мобильный
     button{min-height:36px} растягивает круг в овал (см. память). */
  width: 32px; height: 32px;
  min-width: 32px; max-width: 32px;
  min-height: 32px; max-height: 32px;
  padding: 0;
  border: none; background: transparent; border-radius: var(--radius-full);
  color: var(--color-text-dim); cursor: pointer;
}
.rimg-btn:hover { background: var(--color-surface-low); color: var(--color-text); }
.rimg-btn.on { background: var(--color-primary); color: var(--color-on-primary); }
.rimg-btn .material-symbols-outlined { font-size: 18px; }
.rimg-sep { width: 1px; height: 20px; background: var(--color-outline-dim); }
.rimg-range { width: 96px; accent-color: var(--color-primary); cursor: pointer; }

/* Угловая ручка (правый нижний угол). Крупнее на тач-устройствах. */
.rimg-grip {
  position: absolute;
  right: -6px; bottom: -6px;
  width: 16px; height: 16px;
  border-radius: 50%;
  background: var(--color-primary);
  border: 2px solid var(--color-surface);
  cursor: nwse-resize;
  z-index: 3;
  touch-action: none;
}
@media (hover: none) and (pointer: coarse) {
  .rimg-grip { width: 26px; height: 26px; right: -10px; bottom: -10px; }
  .rimg-range { width: 84px; }
}
</style>
