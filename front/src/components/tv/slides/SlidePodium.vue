<template>
  <div class="tv-podium-wrap">
    <div class="tv-stage-eyebrow">
      <span class="material-symbols-outlined" style="color: var(--color-warning)">workspace_premium</span>
      <span>{{ slide.heroEyebrow }}</span>
    </div>
    <div v-if="podiumList.length === 0" class="tv-stage-empty">
      Пока никто не работал
    </div>
    <div v-else class="tv-podium">
      <!-- Визуальный порядок: 2-е, 1-е, 3-е место -->
      <div
        v-for="place in podiumOrder"
        :key="place"
        class="tv-podium-col"
        :class="['tv-podium-col--' + place, { 'tv-podium-col--empty': !podiumList[place - 1] }]"
      >
        <template v-if="podiumList[place - 1]">
          <div class="tv-podium-medal">
            <span v-if="place === 1" class="tv-fire material-symbols-outlined">local_fire_department</span>
            <span class="tv-podium-place">{{ place }}</span>
          </div>
          <div class="tv-podium-avatar-wrap">
            <img class="tv-podium-avatar" :src="avatarOf(podiumList[place - 1].user_id)" alt="" />
          </div>
          <div class="tv-podium-fio">{{ podiumList[place - 1].fio }}</div>
          <div class="tv-podium-hours">
            <TvCount :value="podiumList[place - 1].total_hours" format="hours" />
          </div>
          <div class="tv-podium-base">
            <div class="tv-podium-base-num">{{ place }}</div>
          </div>
        </template>
        <template v-else>
          <div class="tv-podium-place-empty">{{ place }}</div>
          <div class="tv-podium-empty-text">—</div>
          <div class="tv-podium-base">
            <div class="tv-podium-base-num">{{ place }}</div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
// Пьедестал топ-3 сотрудников по часам за период.
import { computed } from 'vue'
import TvCount from '../TvCount.vue'
import { num } from '../tvFormat.js'

const props = defineProps({
  slide: { type: Object, required: true },
  employees: { type: Array, default: () => [] },
  avatarOf: { type: Function, required: true },
})

const podiumList = computed(() =>
  [...props.employees].sort((a, b) => num(b.total_hours) - num(a.total_hours)).slice(0, 3))
const podiumOrder = [2, 1, 3]
</script>

<style scoped>
.tv-podium-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: clamp(14px, 1.6vmin, 22px);
  min-height: 0;
}

.tv-podium {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: clamp(10px, 1.4vmin, 18px);
  align-items: end;
  min-height: 0;
}

.tv-podium-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: clamp(8px, 1vmin, 14px);
  text-align: center;
  min-width: 0;
  animation: tv-podium-rise 0.7s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
}

.tv-podium-col--1 { animation-delay: 0.25s; }
.tv-podium-col--2 { animation-delay: 0.1s; }
.tv-podium-col--3 { animation-delay: 0.4s; }

@keyframes tv-podium-rise {
  from { opacity: 0; transform: translateY(40px); }
  to   { opacity: 1; transform: translateY(0); }
}

.tv-podium-medal {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: clamp(36px, 4vmin, 56px);
  height: clamp(36px, 4vmin, 56px);
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  position: relative;
}

.tv-podium-col--1 .tv-podium-medal {
  background: color-mix(in oklch, var(--color-warning) 90%, white);
  color: var(--color-on-warning);
  box-shadow: 0 0 24px color-mix(in oklch, var(--color-warning) 50%, transparent);
}
.tv-podium-col--2 .tv-podium-medal {
  background: color-mix(in oklch, var(--color-outline) 30%, var(--color-surface-high));
  color: var(--color-text);
}
.tv-podium-col--3 .tv-podium-medal {
  background: color-mix(in oklch, var(--color-tertiary) 50%, var(--color-surface-high));
  color: var(--color-on-tertiary-container);
}

.tv-podium-place {
  font-size: clamp(16px, 2vmin, 24px);
  font-weight: 900;
}

.tv-podium-avatar-wrap {
  width: clamp(64px, 10vmin, 140px);
  height: clamp(64px, 10vmin, 140px);
  border-radius: 50%;
  border: clamp(3px, 0.4vmin, 5px) solid var(--color-primary);
  overflow: hidden;
  background: var(--color-surface-high);
  flex-shrink: 0;
}

.tv-podium-col--1 .tv-podium-avatar-wrap {
  border-color: var(--color-warning);
  width: clamp(80px, 13vmin, 170px);
  height: clamp(80px, 13vmin, 170px);
  box-shadow: 0 0 36px color-mix(in oklch, var(--color-warning) 35%, transparent);
}

.tv-podium-col--2 .tv-podium-avatar-wrap { border-color: var(--color-outline); }
.tv-podium-col--3 .tv-podium-avatar-wrap { border-color: var(--color-tertiary); }

.tv-podium-avatar {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.tv-podium-fio {
  font-size: clamp(14px, 1.8vmin, 22px);
  font-weight: 700;
  color: var(--color-text);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tv-podium-hours {
  font-size: clamp(16px, 2.4vmin, 30px);
  font-weight: 800;
  color: var(--color-primary);
  font-variant-numeric: tabular-nums;
}

.tv-podium-col--1 .tv-podium-hours { color: var(--color-warning); }
.tv-podium-col--3 .tv-podium-hours { color: var(--color-tertiary); }

.tv-podium-base {
  width: 100%;
  background: color-mix(in oklch, var(--color-primary) 14%, var(--color-surface-low));
  border-radius: 12px 12px 0 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 900;
  font-size: clamp(28px, 4vmin, 56px);
  color: color-mix(in oklch, var(--color-text) 35%, transparent);
  height: clamp(36px, 5vmin, 70px);
  margin-top: clamp(4px, 0.8vmin, 10px);
}

.tv-podium-col--1 .tv-podium-base {
  background: color-mix(in oklch, var(--color-warning) 22%, var(--color-surface-low));
  height: clamp(58px, 8vmin, 110px);
}
.tv-podium-col--2 .tv-podium-base {
  background: color-mix(in oklch, var(--color-outline) 30%, var(--color-surface-low));
  height: clamp(46px, 6vmin, 88px);
}
.tv-podium-col--3 .tv-podium-base {
  background: color-mix(in oklch, var(--color-tertiary) 22%, var(--color-surface-low));
  height: clamp(34px, 4.5vmin, 66px);
}

.tv-podium-place-empty {
  font-size: clamp(28px, 4vmin, 60px);
  font-weight: 900;
  color: var(--color-outline);
  margin: clamp(20px, 3vmin, 40px) 0;
}

.tv-podium-empty-text {
  color: var(--color-text-dim);
  font-size: clamp(14px, 1.6vmin, 18px);
}
</style>
