<template>
  <!-- Ряд свотчей — в едином стиле с палитрой контекстного меню задач/заметок. -->
  <div class="color-picker">
    <button
      v-for="c in colors"
      :key="c.id"
      class="cp-swatch"
      :class="{ active: modelValue === c.id }"
      :style="{ background: `var(--tag-${c.id}-surface)`, borderColor: `var(--tag-${c.id}-border)` }"
      :title="c.label"
      @click.stop="select(c.id)"
    />
    <button
      class="cp-swatch off"
      :class="{ active: !modelValue }"
      title="Без цвета"
      @click.stop="select(null)"
    >
      <span class="material-symbols-outlined">format_color_reset</span>
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
  align-items: center;
  gap: 6px;
}

.cp-swatch {
  width: 22px;
  height: 22px;
  min-height: 22px;
  border-radius: var(--radius-sm);
  border: 1px solid;
  cursor: pointer;
  padding: 0;
  flex-shrink: 0;
}

.cp-swatch.active {
  outline: 2px solid var(--color-primary);
  outline-offset: 1px;
}

.cp-swatch.off {
  display: grid;
  place-items: center;
  background: var(--color-surface);
  border-color: var(--color-outline-variant);
  color: var(--color-text-dim);
}

.cp-swatch.off .material-symbols-outlined { font-size: 15px; }

/* Тач-зоны на мобильных крупнее (глобальный минимум тап-таргета). */
@media (max-width: 768px) {
  .cp-swatch {
    width: 36px;
    height: 36px;
    min-height: 36px;
  }
  .cp-swatch.off .material-symbols-outlined { font-size: 18px; }
}
</style>
