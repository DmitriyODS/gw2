<template>
  <div class="tv-ranking-wrap">
    <div class="tv-stage-eyebrow">
      <span class="material-symbols-outlined" style="color: var(--color-primary)">leaderboard</span>
      <span>{{ slide.heroEyebrow }}</span>
    </div>
    <div v-if="rankingList.length === 0" class="tv-stage-empty">За период работа не велась</div>
    <ol v-else class="tv-ranking">
      <li
        v-for="(e, i) in rankingList"
        :key="e.user_id || e.fio + i"
        class="tv-ranking-item"
        :class="{ 'is-leader': i === 0 }"
        :style="{ '--row-delay': i * 80 + 'ms' }"
      >
        <span class="tv-ranking-rank">
          <span v-if="i === 0" class="tv-fire material-symbols-outlined">local_fire_department</span>
          <span v-else>{{ i + 1 }}</span>
        </span>
        <img class="tv-ranking-avatar" :src="avatarOf(e.user_id)" alt="" />
        <span class="tv-ranking-fio">{{ e.fio }}</span>
        <div class="tv-ranking-bar">
          <div
            class="tv-ranking-bar-fill"
            :style="{ '--bar-width': barPercent(e.total_hours, rankingMax) + '%' }"
          ></div>
        </div>
        <span class="tv-ranking-value">
          <TvCount :value="e.total_hours" format="hours" />
        </span>
      </li>
    </ol>
  </div>
</template>

<script setup>
// Рейтинг топ-5 сотрудников по часам с барами.
import { computed } from 'vue'
import TvCount from '../TvCount.vue'
import { num, barPercent } from '../tvFormat.js'

const props = defineProps({
  slide: { type: Object, required: true },
  employees: { type: Array, default: () => [] },
  avatarOf: { type: Function, required: true },
})

const rankingList = computed(() =>
  [...props.employees].sort((a, b) => num(b.total_hours) - num(a.total_hours)).slice(0, 5))
const rankingMax = computed(() => Math.max(1, ...rankingList.value.map(e => num(e.total_hours))))
</script>

<style scoped>
.tv-ranking-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 1.6vmin, 20px);
  min-height: 0;
}

.tv-ranking {
  list-style: none;
  padding: 0;
  margin: 0;
  flex: 1;
  display: grid;
  grid-template-rows: repeat(5, 1fr);
  gap: clamp(6px, 0.8vmin, 12px);
  min-height: 0;
}

.tv-ranking-item {
  display: grid;
  grid-template-columns: clamp(40px, 4.6vmin, 60px)
                         clamp(40px, 5.4vmin, 70px)
                         minmax(0, 1.4fr)
                         minmax(0, 2.4fr)
                         auto;
  gap: clamp(10px, 1.4vmin, 18px);
  align-items: center;
  padding: clamp(6px, 0.8vmin, 12px) clamp(10px, 1.4vmin, 18px);
  background: color-mix(in oklch, var(--color-surface-high) 60%, transparent);
  border-radius: clamp(12px, 1.4vmin, 18px);
  animation: tv-row-in 0.6s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
  animation-delay: var(--row-delay, 0ms);
}

.tv-ranking-item.is-leader {
  background: color-mix(in oklch, var(--color-warning) 16%, var(--color-surface));
  box-shadow: 0 0 20px color-mix(in oklch, var(--color-warning) 18%, transparent) inset;
}

.tv-ranking-rank {
  font-size: clamp(20px, 2.6vmin, 32px);
  font-weight: 900;
  color: var(--color-primary);
  text-align: center;
  position: relative;
}

.tv-ranking-item.is-leader .tv-ranking-rank {
  color: var(--color-warning);
}

.tv-ranking-rank .tv-fire {
  position: static;
  font-size: clamp(26px, 3vmin, 40px);
}

.tv-ranking-avatar {
  width: clamp(40px, 5.4vmin, 70px);
  height: clamp(40px, 5.4vmin, 70px);
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--color-primary);
}

.tv-ranking-item.is-leader .tv-ranking-avatar { border-color: var(--color-warning); }

.tv-ranking-fio {
  font-size: clamp(15px, 2vmin, 24px);
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tv-ranking-bar {
  height: clamp(10px, 1.4vmin, 16px);
  background: var(--color-outline-dim);
  border-radius: 999px;
  overflow: hidden;
}

.tv-ranking-bar-fill {
  height: 100%;
  background: linear-gradient(90deg,
    color-mix(in oklch, var(--color-primary) 80%, transparent),
    var(--color-primary));
  border-radius: 999px;
  width: 0;
  animation: tv-bar-fill 0.9s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
  animation-delay: calc(var(--row-delay, 0ms) + 200ms);
}

.tv-ranking-item.is-leader .tv-ranking-bar-fill {
  background: linear-gradient(90deg,
    color-mix(in oklch, var(--color-warning) 80%, transparent),
    var(--color-warning));
}

.tv-ranking-value {
  font-size: clamp(15px, 2vmin, 24px);
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  color: var(--color-text);
  white-space: nowrap;
}
</style>
