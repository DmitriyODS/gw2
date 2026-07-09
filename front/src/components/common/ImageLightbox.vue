<template>
  <!-- Полноэкранный просмотр картинки: зум (колесо/кнопки), поворот, панорама
       перетаскиванием. Teleport в body, выше всех слоёв. -->
  <teleport to="body">
    <transition name="ilb">
      <div v-if="modelValue" class="ilb" @click.self="close" @wheel.prevent="onWheel">
        <div class="ilb-toolbar" @click.stop>
          <button class="ilb-btn" title="Уменьшить" @click="zoomBy(-0.25)">
            <span class="material-symbols-outlined">zoom_out</span>
          </button>
          <button class="ilb-btn" title="Увеличить" @click="zoomBy(0.25)">
            <span class="material-symbols-outlined">zoom_in</span>
          </button>
          <button class="ilb-btn" title="Повернуть" @click="rotate">
            <span class="material-symbols-outlined">rotate_right</span>
          </button>
          <button class="ilb-btn" title="Сбросить" @click="reset">
            <span class="material-symbols-outlined">restart_alt</span>
          </button>
          <a class="ilb-btn" :href="currentSrc" :download="currentCaption || ''" title="Скачать" @click.stop>
            <span class="material-symbols-outlined">download</span>
          </a>
          <button class="ilb-btn" title="Закрыть" @click="close">
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>

        <!-- Галерея: стрелки пролистывания при нескольких картинках -->
        <button v-if="hasGallery" class="ilb-btn ilb-nav prev" title="Предыдущая" @click.stop="step(-1)">
          <span class="material-symbols-outlined">chevron_left</span>
        </button>
        <button v-if="hasGallery" class="ilb-btn ilb-nav next" title="Следующая" @click.stop="step(1)">
          <span class="material-symbols-outlined">chevron_right</span>
        </button>

        <img
          :src="currentSrc"
          :alt="currentCaption || 'Изображение'"
          class="ilb-img"
          :class="{ grabbing: dragging }"
          :style="imgStyle"
          @mousedown.prevent="startDrag"
          @click.stop
        />
        <div v-if="currentCaption || hasGallery" class="ilb-caption" @click.stop>
          <template v-if="hasGallery">{{ index + 1 }} / {{ items.length }}<template v-if="currentCaption"> · </template></template>{{ currentCaption }}
        </div>
      </div>
    </transition>
  </teleport>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, required: true },
  src: { type: String, default: '' },
  caption: { type: String, default: '' },
  // Галерея: [{src, caption}] + стартовый индекс — тогда src/caption не нужны.
  items: { type: Array, default: () => [] },
  startIndex: { type: Number, default: 0 },
})
const emit = defineEmits(['update:modelValue'])

const index = ref(0)
const hasGallery = computed(() => props.items.length > 1)
const currentSrc = computed(() => (props.items.length ? props.items[index.value]?.src : props.src))
const currentCaption = computed(() => (props.items.length ? props.items[index.value]?.caption || '' : props.caption))

function step(d) {
  if (!props.items.length) return
  index.value = (index.value + d + props.items.length) % props.items.length
  reset()
}

const scale = ref(1)
const angle = ref(0)
const offset = reactive({ x: 0, y: 0 })
const dragging = ref(false)

const imgStyle = computed(() => ({
  transform: `translate(${offset.x}px, ${offset.y}px) scale(${scale.value}) rotate(${angle.value}deg)`,
}))

function close() { emit('update:modelValue', false) }
function reset() { scale.value = 1; angle.value = 0; offset.x = 0; offset.y = 0 }
function zoomBy(d) { scale.value = Math.min(8, Math.max(0.2, +(scale.value + d).toFixed(2))) }
function rotate() { angle.value = (angle.value + 90) % 360 }
function onWheel(e) { zoomBy(e.deltaY < 0 ? 0.2 : -0.2) }

let start = null
function startDrag(e) {
  dragging.value = true
  start = { x: e.clientX - offset.x, y: e.clientY - offset.y }
  window.addEventListener('mousemove', onDrag)
  window.addEventListener('mouseup', endDrag, { once: true })
}
function onDrag(e) {
  if (!start) return
  offset.x = e.clientX - start.x
  offset.y = e.clientY - start.y
}
function endDrag() {
  dragging.value = false
  start = null
  window.removeEventListener('mousemove', onDrag)
}

function onKey(e) {
  if (!props.modelValue) return
  if (e.key === 'Escape') close()
  else if (e.key === 'ArrowLeft') step(-1)
  else if (e.key === 'ArrowRight') step(1)
}
watch(() => props.modelValue, (v) => {
  if (v) {
    index.value = Math.min(Math.max(props.startIndex, 0), Math.max(props.items.length - 1, 0))
    reset()
    window.addEventListener('keydown', onKey)
  } else window.removeEventListener('keydown', onKey)
})
</script>

<style scoped>
.ilb {
  position: fixed;
  inset: 0;
  z-index: 10200;
  background: color-mix(in oklch, var(--color-scrim, #000) 86%, transparent);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}
.ilb-toolbar {
  position: absolute;
  top: 16px;
  right: 16px;
  display: flex;
  gap: 8px;
  z-index: 2;
}
.ilb-btn {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-full);
  background: color-mix(in oklch, var(--color-surface) 22%, transparent);
  color: #fff;
  cursor: pointer;
  transition: background 0.15s;
}
.ilb-btn:hover { background: color-mix(in oklch, var(--color-surface) 42%, transparent); }
.ilb-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  z-index: 2;
  width: 48px;
  height: 48px;
}
.ilb-nav.prev { left: 16px; }
.ilb-nav.next { right: 16px; }
.ilb-nav .material-symbols-outlined { font-size: 28px; }
.ilb-img {
  max-width: 88vw;
  max-height: 84vh;
  user-select: none;
  cursor: grab;
  transition: transform 0.08s linear;
  border-radius: var(--radius-sm);
  will-change: transform;
}
.ilb-img.grabbing { cursor: grabbing; transition: none; }
.ilb-caption {
  position: absolute;
  bottom: 18px;
  left: 50%;
  transform: translateX(-50%);
  max-width: 80vw;
  padding: 6px 14px;
  border-radius: var(--radius-full);
  background: color-mix(in oklch, var(--color-surface) 22%, transparent);
  color: #fff;
  font-size: 13px;
  text-align: center;
}
.ilb-enter-active, .ilb-leave-active { transition: opacity 0.18s ease; }
.ilb-enter-from, .ilb-leave-to { opacity: 0; }
</style>
