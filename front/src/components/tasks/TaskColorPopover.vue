<template>
  <Teleport to="body">
    <div
      v-if="modelValue && anchorRect"
      ref="popoverRef"
      class="task-color-popover"
      :style="positionStyle"
      @click.stop
    >
      <TaskColorPicker
        :model-value="value"
        @select="onSelect"
      />
    </div>
  </Teleport>
</template>

<script setup>
import { computed, ref, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import TaskColorPicker from '@/components/tasks/TaskColorPicker.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  anchor: { type: [Object, null], default: null },
  value: { type: String, default: null },
})

const emit = defineEmits(['update:modelValue', 'select'])

const popoverRef = ref(null)
const anchorRect = ref(null)
const positionStyle = ref({})

const POPOVER_WIDTH = 220
const SCREEN_PADDING = 12
const GAP = 8

function recompute() {
  if (!props.anchor) {
    anchorRect.value = null
    return
  }
  const rect = props.anchor.getBoundingClientRect()
  anchorRect.value = rect
  const vw = window.innerWidth
  const vh = window.innerHeight

  // По умолчанию выравниваем правый край попапа с правым краем кнопки
  let left = Math.min(rect.right - POPOVER_WIDTH, vw - POPOVER_WIDTH - SCREEN_PADDING)
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
    width: `${POPOVER_WIDTH}px`,
  }
}

watch(() => props.modelValue, async (v) => {
  if (!v) return
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
</script>

<style scoped>
.task-color-popover {
  position: fixed;
  z-index: 1200;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: 12px;
  box-sizing: border-box;
}
</style>
