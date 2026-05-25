<template>
  <div class="color-picker">
    <button
      class="color-swatch none"
      :class="{ selected: !modelValue }"
      title="Без цвета"
      @click.stop="select(null)"
    >
      <span class="material-symbols-outlined">format_color_reset</span>
    </button>
    <button
      v-for="c in colors"
      :key="c.id"
      class="color-swatch"
      :class="{ selected: modelValue === c.id }"
      :style="{ '--swatch': `var(--tag-${c.id}-accent)` }"
      :title="c.label"
      @click.stop="select(c.id)"
    >
      <span v-if="modelValue === c.id" class="material-symbols-outlined">check</span>
    </button>
  </div>
</template>

<script setup>
import { TASK_COLORS } from '@/utils/taskColors.js'

defineProps({
  modelValue: { type: String, default: null },
})

const emit = defineEmits(['update:modelValue', 'select'])

const colors = TASK_COLORS

function select(id) {
  emit('update:modelValue', id)
  emit('select', id)
}
</script>

<style scoped>
.color-picker {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.color-swatch {
  width: 28px;
  height: 28px;
  min-height: 28px;
  border-radius: 50%;
  border: 2px solid transparent;
  background: var(--swatch);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  flex-shrink: 0;
  flex-grow: 0;
  box-sizing: border-box;
  aspect-ratio: 1 / 1;
  transition: transform 0.12s, border-color 0.12s;
  box-shadow: var(--shadow-sm);
}

/* На мобильных глобально включён min-height: 36px для тап-таргетов —
   синхронизируем ширину/высоту, чтобы свотчи оставались круглыми. */
@media (max-width: 768px) {
  .color-swatch {
    width: 36px;
    height: 36px;
    min-height: 36px;
  }
}

.color-swatch:hover {
  transform: scale(1.12);
}

.color-swatch.selected {
  border-color: var(--color-text);
}

.color-swatch .material-symbols-outlined {
  font-size: 16px;
  color: oklch(1 0 0);
  font-variation-settings: 'FILL' 1, 'wght' 600;
}

.color-swatch.none {
  background: var(--color-surface);
  border: 2px solid var(--gw-border);
  color: var(--gw-text-secondary);
}

.color-swatch.none.selected {
  border-color: var(--color-text);
}

.color-swatch.none .material-symbols-outlined {
  font-size: 16px;
  color: var(--gw-text-secondary);
}
</style>
