<template>
  <div class="tv-wt-wrap">
    <div class="tv-stage-eyebrow">
      <span class="material-symbols-outlined" style="color: var(--color-secondary)">category</span>
      <span>{{ slide.heroEyebrow }}</span>
    </div>
    <div v-if="typeList.length === 0" class="tv-stage-empty">За период работа не велась</div>
    <div v-else class="tv-wt">
      <div
        v-for="(t, i) in typeList"
        :key="t.type_id || t.name"
        class="tv-wt-row"
        :style="{ '--row-delay': i * 90 + 'ms' }"
      >
        <div class="tv-wt-label">{{ t.name }}</div>
        <div class="tv-wt-track">
          <div class="tv-wt-fill" :style="{ '--bar-width': barPercent(t.total_hours, typeMax) + '%' }"></div>
        </div>
        <div class="tv-wt-value">
          <TvCount :value="t.total_hours" format="hours" />
          <span class="tv-wt-tasks">{{ t.tasks_count }} задач</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
// «Структура работ»: топ-5 типов юнитов по часам за период.
import { computed } from 'vue'
import TvCount from '../TvCount.vue'
import { num, barPercent } from '../tvFormat.js'

const props = defineProps({
  slide: { type: Object, required: true },
  unitTypes: { type: Array, default: () => [] }, // extended.by_unit_types
})

const typeList = computed(() =>
  [...props.unitTypes].sort((a, b) => num(b.total_hours) - num(a.total_hours)).slice(0, 5))
const typeMax = computed(() => Math.max(1, ...typeList.value.map(t => num(t.total_hours))))
</script>

<style scoped>
.tv-wt-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 1.6vmin, 20px);
  min-height: 0;
}

.tv-wt {
  flex: 1;
  display: grid;
  grid-template-rows: repeat(auto-fit, minmax(0, 1fr));
  gap: clamp(8px, 1vmin, 14px);
  min-height: 0;
}

.tv-wt-row {
  display: grid;
  grid-template-columns: minmax(140px, 28%) 1fr auto;
  gap: clamp(12px, 1.6vmin, 20px);
  align-items: center;
  animation: tv-row-in 0.6s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
  animation-delay: var(--row-delay, 0ms);
}

.tv-wt-label {
  font-size: clamp(15px, 1.9vmin, 22px);
  font-weight: 700;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tv-wt-track {
  height: clamp(22px, 3vmin, 40px);
  background: color-mix(in oklch, var(--color-surface-high) 70%, transparent);
  border-radius: 999px;
  overflow: hidden;
}

.tv-wt-fill {
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg,
    color-mix(in oklch, var(--color-secondary) 60%, transparent),
    var(--color-secondary));
  width: 0;
  animation: tv-bar-fill 0.9s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
  animation-delay: calc(var(--row-delay, 0ms) + 200ms);
}

.tv-wt-value {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  font-size: clamp(15px, 2vmin, 24px);
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  color: var(--color-text);
  white-space: nowrap;
}

.tv-wt-tasks {
  font-size: clamp(10px, 1.3vmin, 15px);
  font-weight: 600;
  color: var(--color-text-dim);
}
</style>
