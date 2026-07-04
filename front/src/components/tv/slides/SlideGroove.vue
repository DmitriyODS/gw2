<template>
  <div class="tv-groove-wrap">
    <div class="tv-stage-eyebrow">
      <span style="font-size: 1.2em">👾</span>
      <span>{{ slide.heroEyebrow }}</span>
    </div>
    <div v-if="pets.length === 0" class="tv-stage-empty">
      Грувики ещё не вылупились — загляните в «Мой Groove»
    </div>
    <ol v-else class="tv-groove-list">
      <li
        v-for="(p, i) in pets"
        :key="p.user_id"
        class="tv-groove-item"
        :class="{ 'is-leader': i === 0 }"
        :style="{ '--row-delay': i * 80 + 'ms' }"
      >
        <span class="tv-groove-rank" :class="{ 'is-leader': i === 0 }">
          <span v-if="i === 0" class="tv-fire material-symbols-outlined">local_fire_department</span>
          <span class="tv-groove-rank-num">{{ i + 1 }}</span>
        </span>
        <span class="tv-groove-pet">
          <span class="tv-groove-emoji" :class="{ sick: p.sick }">{{ petEmoji(p) }}</span>
          <span v-if="p.hat" class="tv-groove-hat">{{ SHOP_ITEMS[p.hat]?.emoji || '' }}</span>
          <span v-if="p.sick" class="tv-groove-sick">🤒</span>
        </span>
        <span class="tv-groove-names">
          <span class="tv-groove-petname">{{ p.name }}</span>
          <span class="tv-groove-owner">{{ p.user?.fio }}</span>
        </span>
        <span class="tv-groove-stage">{{ PET_STAGES[p.stage] || '' }}</span>
        <div class="tv-groove-bar">
          <div class="tv-groove-bar-fill" :style="{ '--bar-width': barPercent(p.xp, xpMax) + '%' }"></div>
        </div>
        <span class="tv-groove-xp"><TvCount :value="p.xp" format="int" /> XP</span>
      </li>
    </ol>
  </div>
</template>

<script setup>
// Зал славы Грувиков: топ питомцев по XP.
import { computed } from 'vue'
import TvCount from '../TvCount.vue'
import { num, barPercent } from '../tvFormat.js'
import { petEmoji, PET_STAGES, SHOP_ITEMS } from '@/utils/groove.js'

const props = defineProps({
  slide: { type: Object, required: true },
  groove: { type: Object, default: null }, // ответ getGrooveTv(): {pets, raid}
})

const pets = computed(() => (props.groove?.pets || []).slice(0, 5))
const xpMax = computed(() => Math.max(1, ...pets.value.map(p => num(p.xp))))
</script>

<style scoped>
.tv-groove-wrap {
  display: flex;
  flex-direction: column;
  gap: clamp(8px, 1.5vmin, 16px);
  height: 100%;
  justify-content: center;
}
.tv-groove-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: clamp(6px, 1.2vmin, 14px);
}
.tv-groove-item {
  display: grid;
  grid-template-columns: clamp(28px, 3.4vmin, 44px) clamp(44px, 6vmin, 72px) minmax(0, 1.4fr) auto minmax(0, 1fr) auto;
  align-items: center;
  gap: clamp(8px, 1.4vmin, 18px);
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: clamp(12px, 1.6vmin, 18px);
  padding: clamp(6px, 1vmin, 12px) clamp(10px, 1.6vmin, 20px);
  animation: tv-groove-row-in 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) both;
  animation-delay: var(--row-delay, 0ms);
}
.tv-groove-item.is-leader {
  border-color: color-mix(in oklch, var(--color-warning) 55%, transparent);
}
/* Своя (более короткая) версия въезда строк — не путать с общей tv-row-in. */
@keyframes tv-groove-row-in {
  from { opacity: 0; transform: translateX(-18px); }
  to { opacity: 1; transform: translateX(0); }
}
.tv-groove-rank {
  font-size: clamp(14px, 2vmin, 24px);
  font-weight: 800;
  text-align: center;
  color: var(--color-text-dim);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}
.tv-groove-rank.is-leader { color: var(--color-warning); }
.tv-groove-rank.is-leader .tv-groove-rank-num { font-size: 1.1em; }
.tv-groove-rank .tv-fire { position: static; }
.tv-groove-pet {
  position: relative;
  width: clamp(44px, 6vmin, 72px);
  height: clamp(44px, 6vmin, 72px);
  border-radius: 50%;
  background: var(--color-primary-container);
  display: grid;
  place-items: center;
}
.tv-groove-emoji { font-size: clamp(22px, 3.2vmin, 40px); line-height: 1; }
.tv-groove-emoji.sick { filter: grayscale(0.55) brightness(0.92); }
.tv-groove-hat {
  position: absolute;
  top: clamp(-10px, -1.2vmin, -6px);
  right: -2px;
  font-size: clamp(14px, 2vmin, 24px);
  transform: rotate(12deg);
}
.tv-groove-sick {
  position: absolute;
  bottom: -4px;
  left: -4px;
  font-size: clamp(13px, 1.8vmin, 22px);
}
.tv-groove-names { display: flex; flex-direction: column; min-width: 0; }
.tv-groove-petname {
  font-size: clamp(13px, 1.9vmin, 24px);
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tv-groove-owner {
  font-size: clamp(10px, 1.3vmin, 16px);
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tv-groove-stage {
  font-size: clamp(10px, 1.4vmin, 17px);
  font-weight: 700;
  padding: clamp(2px, 0.5vmin, 6px) clamp(8px, 1.2vmin, 16px);
  border-radius: 999px;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  white-space: nowrap;
}
.tv-groove-bar {
  height: clamp(8px, 1.1vmin, 14px);
  border-radius: 999px;
  background: var(--color-surface-high);
  overflow: hidden;
}
.tv-groove-bar-fill {
  height: 100%;
  border-radius: inherit;
  background: var(--color-primary);
  width: var(--bar-width, 0%);
  animation: tv-bar-fill 0.9s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}
.tv-groove-xp {
  font-size: clamp(12px, 1.7vmin, 22px);
  font-weight: 800;
  white-space: nowrap;
}
@media (max-aspect-ratio: 1/1) {
  .tv-groove-item { grid-template-columns: clamp(24px, 3vmin, 36px) clamp(40px, 5vmin, 56px) minmax(0, 1fr) auto; }
  .tv-groove-bar, .tv-groove-xp { display: none; }
}
</style>
