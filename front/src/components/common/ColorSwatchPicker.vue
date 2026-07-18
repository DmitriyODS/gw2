<template>
  <div class="csp" role="radiogroup" :aria-label="ariaLabel">
    <button
      type="button"
      class="csp-swatch csp-none"
      :class="{ active: !modelValue }"
      role="radio"
      :aria-checked="!modelValue"
      title="Без цвета"
      @click="pick('')"
    >
      <span class="material-symbols-outlined">block</span>
    </button>
    <button
      v-for="c in colors"
      :key="c.id"
      type="button"
      class="csp-swatch"
      :class="{ active: modelValue === c.id }"
      :style="swatchStyle(c.id)"
      role="radio"
      :aria-checked="modelValue === c.id"
      :title="c.label"
      @click="pick(c.id)"
    >
      <span v-if="modelValue === c.id" class="material-symbols-outlined csp-check">check</span>
    </button>
  </div>
</template>

<script setup>
import { TASK_COLORS } from '@/utils/taskColors.js'

defineProps({
  modelValue: { type: String, default: '' },
  ariaLabel: { type: String, default: 'Выбор цвета' },
})
const emit = defineEmits(['update:modelValue'])
const colors = TASK_COLORS

function swatchStyle(id) {
  return {
    background: `var(--tag-${id}-surface)`,
    borderColor: `var(--tag-${id}-border)`,
    color: `var(--tag-${id}-accent)`,
  }
}
function pick(id) { emit('update:modelValue', id) }
</script>

<style scoped>
/* Одна компактная прокручиваемая строка — не раздувает контекстное меню/диалог.
   Внутренние отступы дают место рамке выделения (box-shadow), чтобы её не
   срезал скролл-контейнер. */
.csp {
  display: flex;
  flex-wrap: nowrap;
  gap: 10px;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
  padding: 4px 4px 6px;
}
.csp::-webkit-scrollbar { display: none; }
/* Компактные прямоугольники-чипы: занимают мало по высоте и ширине, лишние —
   уезжают в горизонтальный скролл (меню не раздувается). Явные min/max
   перебивают глобальный мобильный button { min-height: 36px }. */
.csp-swatch {
  flex: 0 0 auto;
  width: 30px;
  height: 22px;
  min-width: 30px;
  min-height: 22px;
  max-width: 30px;
  max-height: 22px;
  border-radius: 6px;
  border: 1.5px solid var(--color-outline-dim);
  display: grid;
  place-items: center;
  cursor: pointer;
  padding: 0;
  transition: transform 0.1s ease;
}
.csp-swatch:hover { transform: scale(1.08); }
/* Рамка выделения — ВНУТРЬ (inset): скролл-контейнер не срежет её, в отличие
   от внешней тени. */
.csp-swatch.active { box-shadow: inset 0 0 0 2px var(--color-primary); border-color: var(--color-primary); }
.csp-none {
  background: var(--color-surface);
  color: var(--color-text-dim);
}
.csp-none .material-symbols-outlined { font-size: 15px; }
.csp-check { font-size: 16px; font-variation-settings: 'wght' 700; }
</style>
