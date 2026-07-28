<template>
  <div class="hp-backdrop" @pointerdown.self="close">
    <section class="hp" :style="style" role="dialog" aria-label="Hola ассистент">
      <header class="hp-bar">
        <HolaIcon :size="20" class="hp-mark" />
        <h2 class="hp-title">Hola ассистент</h2>
      </header>

      <HolaPanel ref="panelRef" class="hp-body" @navigate="close" />
    </section>
  </div>
</template>

<script setup>
/**
 * Всплывающая панель Hola поверх рабочего стола — наследница строки Spotlight,
 * а не окно раздела: ни кнопки в панели задач, ни своего адреса, ни кнопок
 * управления окном у неё нет. Появляется всегда по центру экрана, закрывается
 * клавишей Esc, кликом мимо панели и уходом в найденный раздел.
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { TASKBAR_RESERVE } from '@/desktop/layout.js'
import HolaPanel from '@/components/hola/HolaPanel.vue'
import HolaIcon from '@/components/common/HolaIcon.vue'

const SIZE = { w: 720, h: 720 }
const MARGIN = 24

const emit = defineEmits(['close'])

const panelRef = ref(null)
const rect = ref(centerRect())

/* Центр экрана с поправкой на панель задач: на невысоком экране панель не
   должна наполовину уезжать под неё. */
function centerRect() {
  const vw = window.innerWidth
  const vh = window.innerHeight - TASKBAR_RESERVE
  const w = Math.min(SIZE.w, vw - MARGIN * 2)
  const h = Math.min(SIZE.h, vh - MARGIN * 2)
  return {
    x: Math.max(MARGIN, Math.round((vw - w) / 2)),
    y: Math.max(MARGIN, Math.round((vh - h) / 2)),
    w,
    h,
  }
}

const style = computed(() => ({
  transform: `translate3d(${rect.value.x}px, ${rect.value.y}px, 0)`,
  width: `${rect.value.w}px`,
  height: `${rect.value.h}px`,
}))

function close() {
  emit('close')
}

function onKeydown(e) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    close()
  }
}

function onResize() {
  rect.value = centerRect()
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('resize', onResize, { passive: true })
  nextTick(() => panelRef.value?.focus())
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('resize', onResize)
})
</script>

<style scoped>
.hp-backdrop {
  position: fixed;
  inset: 0;
  z-index: 960;
  background: color-mix(in oklch, var(--color-text) 18%, transparent);
  -webkit-backdrop-filter: blur(2px);
  backdrop-filter: blur(2px);
  transition: opacity 0.18s ease;
}

/* Плавающий слой — единственное место, где размытие реально работает: внутри
   акриловой панели родитель становится backdrop root, и вложенное стекло уже
   ничего не размывает. */
.hp {
  position: absolute;
  top: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  box-shadow: 0 24px 64px color-mix(in oklch, var(--color-text) 18%, transparent);
  transition: opacity 0.18s ease, translate 0.2s cubic-bezier(0.2, 0, 0, 1),
    scale 0.2s cubic-bezier(0.2, 0, 0, 1);
}

.hp-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  height: 46px;
  padding: 0 16px;
  border-bottom: 1px solid color-mix(in oklch, var(--acrylic-border) 70%, transparent);
  user-select: none;
}

.hp-mark { color: var(--color-primary); }

.hp-title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hp-body { flex: 1; min-height: 0; }

/* Появление — как у прежней строки поиска: панель подаётся сверху и тает. */
.hp-enter-from,
.hp-leave-to { opacity: 0; }

.hp-enter-from .hp,
.hp-leave-to .hp {
  opacity: 0;
  translate: 0 -14px;
  scale: 0.97;
}
</style>
