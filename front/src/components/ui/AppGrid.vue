<template>
  <div class="grid" :style="style"><slot /></div>
</template>

<script setup>
/* Адаптивная сетка плиток и карточек. Колонок столько, сколько влезает в
   ШИРИНУ РОДИТЕЛЯ, ряд делится поровну — то же правило, что у стартового экрана
   мобильного каркаса. Заменила десятки копий `grid-template-columns` с
   собственными брейкпоинтами: раздел живёт окном, и его ширина не равна ширине
   экрана, поэтому media-запросы здесь врут, а auto-fill — нет. */
import { computed } from 'vue'

const props = defineProps({
  /** Минимальная ширина колонки, px. */
  min: { type: [Number, String], default: 220 },
  gap: { type: [Number, String], default: 14 },
  /** Ровно N колонок независимо от ширины (редкий случай — витрины 2×2). */
  columns: { type: Number, default: 0 },
})

const style = computed(() => ({
  gap: typeof props.gap === 'number' ? `${props.gap}px` : props.gap,
  gridTemplateColumns: props.columns
    ? `repeat(${props.columns}, minmax(0, 1fr))`
    : `repeat(auto-fill, minmax(min(${typeof props.min === 'number' ? `${props.min}px` : props.min}, 100%), 1fr))`,
}))
</script>

<style scoped>
.grid { display: grid; }
</style>
