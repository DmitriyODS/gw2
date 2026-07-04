<template>
  <div class="tv-deps-wrap">
    <div class="tv-stage-eyebrow">
      <span class="material-symbols-outlined" style="color: var(--color-tertiary)">apartment</span>
      <span>{{ slide.heroEyebrow }}</span>
    </div>
    <div v-if="deptList.length === 0" class="tv-stage-empty">Нет данных по отделам</div>
    <div v-else class="tv-deps">
      <div
        v-for="(d, i) in deptList"
        :key="d.dept_id || d.name"
        class="tv-dep-bar"
        :style="{ '--row-delay': i * 100 + 'ms' }"
      >
        <div class="tv-dep-bar-label">{{ d.name }}</div>
        <div class="tv-dep-bar-track">
          <div
            class="tv-dep-bar-fill"
            :style="{ '--bar-width': barPercent(d.tasks_count, deptMax) + '%' }"
          >
            <span class="tv-dep-bar-num">{{ d.tasks_count }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
// Горизонтальные бары задач по отделам (топ-5).
import { computed } from 'vue'
import { num, barPercent } from '../tvFormat.js'

const props = defineProps({
  slide: { type: Object, required: true },
  departments: { type: Array, default: () => [] },
})

const deptList = computed(() =>
  [...props.departments].sort((a, b) => num(b.tasks_count) - num(a.tasks_count)).slice(0, 5))
const deptMax = computed(() => Math.max(1, ...deptList.value.map(d => num(d.tasks_count))))
</script>

<style scoped>
.tv-deps-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 1.6vmin, 20px);
  min-height: 0;
}

.tv-deps {
  flex: 1;
  display: grid;
  grid-template-rows: repeat(auto-fit, 1fr);
  gap: clamp(8px, 1vmin, 14px);
  min-height: 0;
}

.tv-dep-bar {
  display: grid;
  grid-template-columns: minmax(120px, 26%) 1fr;
  gap: clamp(12px, 1.6vmin, 20px);
  align-items: center;
  animation: tv-row-in 0.6s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
  animation-delay: var(--row-delay, 0ms);
}

.tv-dep-bar-label {
  font-size: clamp(15px, 1.9vmin, 22px);
  font-weight: 700;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tv-dep-bar-track {
  height: clamp(28px, 4vmin, 50px);
  background: color-mix(in oklch, var(--color-surface-high) 70%, transparent);
  border-radius: 12px;
  overflow: hidden;
  position: relative;
}

.tv-dep-bar-fill {
  height: 100%;
  background: linear-gradient(90deg,
    color-mix(in oklch, var(--color-tertiary) 60%, transparent),
    var(--color-tertiary));
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding-right: clamp(10px, 1.4vmin, 18px);
  color: var(--color-on-tertiary-container);
  font-weight: 900;
  font-size: clamp(14px, 1.8vmin, 22px);
  width: 0;
  animation: tv-bar-fill 0.9s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
  animation-delay: calc(var(--row-delay, 0ms) + 200ms);
  font-variant-numeric: tabular-nums;
}
</style>
