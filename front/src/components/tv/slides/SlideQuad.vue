<template>
  <div class="tv-quad-wrap">
    <div class="tv-stage-eyebrow">
      <span class="material-symbols-outlined" style="color: var(--color-secondary)">view_quilt</span>
      <span>{{ slide.heroEyebrow }}</span>
    </div>
    <div class="tv-quad">
      <div class="tv-quad-tile tone-primary">
        <div class="tv-quad-icon"><span class="material-symbols-outlined">inbox</span></div>
        <div class="tv-quad-num"><TvCount :value="common?.tasks?.received ?? 0" format="int" prefix="+" /></div>
        <div class="tv-quad-label">поступило</div>
      </div>
      <div class="tv-quad-tile tone-success">
        <div class="tv-quad-icon"><span class="material-symbols-outlined">task_alt</span></div>
        <div class="tv-quad-num"><TvCount :value="common?.tasks?.closed ?? 0" format="int" prefix="−" /></div>
        <div class="tv-quad-label">закрыто</div>
      </div>
      <div class="tv-quad-tile tone-tertiary">
        <div class="tv-quad-icon"><span class="material-symbols-outlined">hourglass_top</span></div>
        <div class="tv-quad-num"><TvCount :value="common?.tasks?.remaining ?? 0" format="int" /></div>
        <div class="tv-quad-label">в работе</div>
      </div>
      <div class="tv-quad-tile tone-secondary">
        <div class="tv-quad-icon"><span class="material-symbols-outlined">schedule</span></div>
        <div class="tv-quad-num"><TvCount :value="totalHours" format="hours" /></div>
        <div class="tv-quad-label">часы команды</div>
      </div>
    </div>
  </div>
</template>

<script setup>
// «Период одной картой»: четыре больших KPI-плитки.
import TvCount from '../TvCount.vue'

defineProps({
  slide: { type: Object, required: true },
  common: { type: Object, default: null },
  totalHours: { type: Number, default: 0 },
})
</script>

<style scoped>
.tv-quad-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 1.6vmin, 20px);
  min-height: 0;
}

.tv-quad {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  grid-template-rows: repeat(2, 1fr);
  gap: clamp(12px, 1.6vmin, 20px);
  min-height: 0;
}

.tv-quad-tile {
  border-radius: clamp(14px, 1.8vmin, 22px);
  padding: clamp(16px, 2vmin, 28px);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: clamp(6px, 0.8vmin, 12px);
  position: relative;
  overflow: hidden;
  animation: tv-tile-in 0.65s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
}

.tv-quad-tile:nth-child(1) { animation-delay: 80ms; }
.tv-quad-tile:nth-child(2) { animation-delay: 180ms; }
.tv-quad-tile:nth-child(3) { animation-delay: 280ms; }
.tv-quad-tile:nth-child(4) { animation-delay: 380ms; }

@keyframes tv-tile-in {
  from { opacity: 0; transform: scale(0.92); }
  to   { opacity: 1; transform: scale(1); }
}

.tv-quad-tile.tone-primary   { background: var(--color-primary-container);   color: var(--color-on-primary-container); }
.tv-quad-tile.tone-secondary { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.tv-quad-tile.tone-tertiary  { background: var(--color-tertiary-container);  color: var(--color-on-tertiary-container); }
.tv-quad-tile.tone-success   { background: var(--color-success-container);   color: var(--color-on-success-container); }

.tv-quad-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: clamp(36px, 4vmin, 56px);
  height: clamp(36px, 4vmin, 56px);
  border-radius: 50%;
  background: color-mix(in oklch, currentColor 14%, transparent);
}

.tv-quad-icon .material-symbols-outlined {
  font-size: clamp(22px, 2.6vmin, 32px);
}

.tv-quad-num {
  font-size: clamp(40px, 7.4vmin, 110px);
  font-weight: 900;
  line-height: 0.9;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
}

.tv-quad-label {
  font-size: clamp(13px, 1.6vmin, 18px);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  opacity: 0.85;
}
</style>
