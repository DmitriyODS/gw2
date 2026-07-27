<template>
  <Teleport to="body">
    <Transition name="rpick">
      <div v-if="visible" class="rpick-backdrop" @pointerdown.self="emit('close')" @contextmenu.prevent>
        <div ref="panelEl" class="rpick" :style="style" role="menu" aria-label="Выбор реакции">
          <button
            v-for="e in QUICK_REACTIONS"
            :key="e"
            class="rpick-item"
            :class="{ active: mine.includes(e) }"
            type="button"
            :aria-label="`Реакция ${e}`"
            @click="emit('pick', e)"
          >{{ e }}</button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { QUICK_REACTIONS } from '@/utils/reactions.js'

const props = defineProps({
  visible: { type: Boolean, default: false },
  // Точка привязки — обычно центр кнопки, открывшей панель.
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
  // Уже поставленные мной реакции — подсвечиваем.
  mine: { type: Array, default: () => [] },
})
const emit = defineEmits(['pick', 'close'])

const panelEl = ref(null)
const pos = ref({ x: 0, y: 0 })

const style = computed(() => ({ left: `${pos.value.x}px`, top: `${pos.value.y}px` }))

/* Панель раскрывается над кнопкой и прижимается к экрану, если у края не
   помещается (замер после отрисовки — размер зависит от ширины окна). */
watch(() => props.visible, async (v) => {
  if (!v) return
  pos.value = { x: props.x, y: props.y }
  await nextTick()
  const el = panelEl.value
  if (!el) return
  const r = el.getBoundingClientRect()
  const pad = 8
  const x = Math.min(Math.max(pad, props.x - r.width / 2), window.innerWidth - r.width - pad)
  const above = props.y - r.height - 10
  pos.value = { x, y: above >= pad ? above : Math.min(props.y + 10, window.innerHeight - r.height - pad) }
})

function onKey(e) {
  if (e.key === 'Escape') emit('close')
}

watch(() => props.visible, (v) => {
  if (v) window.addEventListener('keydown', onKey)
  else window.removeEventListener('keydown', onKey)
})

onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<style scoped>
.rpick-backdrop {
  position: fixed;
  inset: 0;
  z-index: 11000;
}

.rpick {
  position: fixed;
  display: grid;
  grid-template-columns: repeat(10, 1fr);
  gap: 2px;
  max-width: min(420px, calc(100vw - 16px));
  padding: 6px;
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
}

@media (max-width: 560px) {
  .rpick { grid-template-columns: repeat(5, 1fr); }
}

.rpick-item {
  width: 36px;
  min-width: 36px;
  max-width: 36px;
  height: 36px;
  min-height: 36px;
  max-height: 36px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  font-size: 20px;
  line-height: 1;
  cursor: pointer;
  transition: background 0.12s, scale 0.12s;
}

.rpick-item:hover { background: color-mix(in oklch, var(--color-primary) 12%, transparent); scale: 1.15; }
.rpick-item.active { background: var(--color-primary-container); }

.rpick-enter-active,
.rpick-leave-active { transition: opacity 0.14s ease; }
.rpick-enter-from,
.rpick-leave-to { opacity: 0; }
.rpick-enter-active .rpick,
.rpick-leave-active .rpick { transition: scale 0.16s cubic-bezier(0.2, 0, 0, 1); }
.rpick-enter-from .rpick,
.rpick-leave-to .rpick { scale: 0.9; }
</style>
