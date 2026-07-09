<template>
  <Teleport to="body">
    <!-- На мобильном — центрированный bottom-sheet поверх диалога задачи.
         На десктопе — popover, привязанный к anchor-кнопке. -->
    <template v-if="modelValue">
      <div v-if="isMobile" class="task-color-backdrop" @click="close">
        <div class="task-color-sheet" @click.stop>
          <div class="sheet-header">
            <span class="sheet-title">Цвет задачи</span>
            <button class="sheet-close" @click="close" aria-label="Закрыть">
              <span class="material-symbols-outlined">close</span>
            </button>
          </div>
          <TaskColorPicker :model-value="value" @select="onSelect" />
        </div>
      </div>
      <div
        v-else-if="anchorRect"
        ref="popoverRef"
        class="task-color-popover"
        :style="positionStyle"
        @click.stop
      >
        <TaskColorPicker :model-value="value" @select="onSelect" />
      </div>
    </template>
  </Teleport>
</template>

<script setup>
import { computed, ref, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import TaskColorPicker from '@/components/tasks/TaskColorPicker.vue'
import { useBreakpoint } from '@/composables/useBreakpoint.js'

const { isMobile } = useBreakpoint()

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  anchor: { type: [Object, null], default: null },
  value: { type: String, default: null },
})

const emit = defineEmits(['update:modelValue', 'select'])

const popoverRef = ref(null)
const anchorRect = ref(null)
const positionStyle = ref({})

const SCREEN_PADDING = 12
const GAP = 8

// Подбираем ширину так, чтобы 9 свотчей (28 или 36 пкс + 8px gap) разложились
// аккуратной сеткой. 3 ряда по 3 на десктопе, 3 ряда по 3 на мобильном.
function pickPopoverWidth(vw) {
  const mobile = vw <= 768
  const swatch = mobile ? 36 : 28
  const cols = 3
  const padding = 12 * 2
  return Math.min(
    vw - SCREEN_PADDING * 2,
    swatch * cols + GAP * (cols - 1) + padding,
  )
}

function recompute() {
  if (!props.anchor) {
    anchorRect.value = null
    return
  }
  const rect = props.anchor.getBoundingClientRect()
  anchorRect.value = rect
  const vw = window.innerWidth
  const vh = window.innerHeight
  const width = pickPopoverWidth(vw)

  // По умолчанию выравниваем правый край попапа с правым краем кнопки
  let left = Math.min(rect.right - width, vw - width - SCREEN_PADDING)
  left = Math.max(SCREEN_PADDING, left)

  // По вертикали — снизу под кнопкой, либо сверху если внизу не помещается
  const popH = popoverRef.value?.offsetHeight ?? 120
  let top = rect.bottom + GAP
  if (top + popH > vh - SCREEN_PADDING) {
    top = Math.max(SCREEN_PADDING, rect.top - popH - GAP)
  }
  positionStyle.value = {
    left: `${Math.round(left)}px`,
    top: `${Math.round(top)}px`,
    width: `${Math.round(width)}px`,
  }
}

watch(() => props.modelValue, async (v) => {
  if (!v) return
  // На мобильном — bottom-sheet без anchor-позиционирования.
  if (isMobile.value) { anchorRect.value = null; return }
  await nextTick()
  recompute()
  // Пересчёт после следующего тика — попап уже отрендерился, известна высота
  await nextTick()
  recompute()
})

function onWindowChange() {
  if (props.modelValue) recompute()
}

function onDocClick(e) {
  if (!props.modelValue) return
  // На мобильном закрытием управляет backdrop — onDocClick не нужен и
  // может срабатывать раньше монтирования sheet.
  if (isMobile.value) return
  const inPopover = popoverRef.value && popoverRef.value.contains(e.target)
  const inAnchor = props.anchor && props.anchor.contains(e.target)
  if (!inPopover && !inAnchor) {
    emit('update:modelValue', false)
  }
}

onMounted(() => {
  window.addEventListener('resize', onWindowChange)
  window.addEventListener('scroll', onWindowChange, true)
  document.addEventListener('click', onDocClick, true)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onWindowChange)
  window.removeEventListener('scroll', onWindowChange, true)
  document.removeEventListener('click', onDocClick, true)
})

function onSelect(color) {
  emit('select', color)
  emit('update:modelValue', false)
}

function close() {
  emit('update:modelValue', false)
}
</script>

<style scoped>
.task-color-popover {
  position: fixed;
  /* Выше PrimeVue Dialog (~1100) и мобильного меню действий внутри TaskModal
     (z-index 10001), чтобы попап не прятался под открытым диалогом. */
  z-index: 10200;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: 12px;
  box-sizing: border-box;
}

.task-color-backdrop {
  position: fixed;
  inset: 0;
  z-index: 10200;
  background: var(--color-scrim, color-mix(in oklch, black 50%, transparent));
  display: flex;
  align-items: flex-end;
  justify-content: center;
  padding: 16px;
  padding-bottom: max(16px, env(safe-area-inset-bottom, 0px));
}

.task-color-sheet {
  width: 100%;
  max-width: 420px;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border-radius: var(--radius-lg, 20px);
  box-shadow: var(--shadow-lg);
  padding: 16px 18px 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.sheet-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.sheet-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text, var(--gw-text));
}

.sheet-close {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: none;
  background: transparent;
  color: var(--color-text-dim, var(--gw-text-secondary));
  cursor: pointer;
  display: grid;
  place-items: center;
}

.sheet-close:active {
  background: color-mix(in oklch, var(--color-primary) 12%, transparent);
}

.sheet-close .material-symbols-outlined {
  font-size: 22px;
}
</style>
