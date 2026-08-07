<template>
  <div class="stack" :class="{ row }" :style="{ gap: gapPx }"><slot /></div>
</template>

<script setup>
/* Стопка блоков с единым промежутком — тело раздела, список строк, ряд кнопок.
   Мелочь, но именно она копировалась в каждый второй файл как
   `display:flex; flex-direction:column; gap:14px`. */
import { computed } from 'vue'

const props = defineProps({
  gap: { type: [Number, String], default: 14 },
  /** Горизонтальная стопка с переносом (ряд чипов, ряд кнопок). */
  row: { type: Boolean, default: false },
})

const gapPx = computed(() => (typeof props.gap === 'number' ? `${props.gap}px` : props.gap))
</script>

<style scoped>
.stack {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.stack.row {
  flex-direction: row;
  align-items: center;
  flex-wrap: wrap;
}
</style>
