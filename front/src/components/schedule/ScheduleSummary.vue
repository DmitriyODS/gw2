<template>
  <div v-if="items.length" class="ssum">
    <span class="ssum-day">{{ label }}</span>
    <span>{{ duration(busy) }} занято</span>
    <span v-if="gap">{{ duration(gap) }} в окнах</span>
    <span v-if="clashes" class="ssum-clash">накладок: {{ clashes }}</span>
    <span class="ssum-range">{{ hhmm(first) }} — {{ hhmm(last) }}</span>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { busyOf, clusters, gapMinutes } from '@/utils/scheduleLayout.js'
import { duration, hhmm } from '@/utils/scheduleCycle.js'

/* Сводка дня — та же цифра, что показывает живая плитка: занятость и окна
   считаются общими правилами (utils/scheduleLayout.js ≡ domain/layout.go). */
const props = defineProps({
  items: { type: Array, default: () => [] },
  gapMin: { type: Number, default: 20 },
  label: { type: String, default: '' },
})

const busy = computed(() => busyOf(props.items))
const gap = computed(() => gapMinutes(props.items, props.gapMin))
const first = computed(() => Math.min(...props.items.map((i) => i.start_min)))
const last = computed(() => Math.max(...props.items.map((i) => i.end_min)))
// Накладка — группа больше одного занятия: она не удваивает занятость, но о
// ней стоит сказать вслух.
const clashes = computed(() => clusters(props.items).filter((g) => g.length > 1).length)
</script>

<style scoped>
.ssum {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 14px;
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-low);
  font-size: 12px;
  color: var(--color-text-dim);
}
.ssum-day { font-weight: 600; color: var(--color-text); }
.ssum-clash { color: var(--color-error); }
.ssum-range { margin-left: auto; font-variant-numeric: tabular-nums; }
</style>
