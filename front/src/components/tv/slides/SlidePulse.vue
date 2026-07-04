<template>
  <div class="tv-pulse-wrap">
    <div class="tv-stage-eyebrow">
      <span class="material-symbols-outlined" style="color: var(--color-success)">monitor_heart</span>
      <span>{{ slide.heroEyebrow }}</span>
    </div>
    <div v-if="days.length === 0" class="tv-stage-empty">Нет данных за период</div>
    <template v-else>
      <div class="tv-pulse-legend">
        <span class="tv-pulse-key"><i class="tv-pulse-dot tv-pulse-dot--received"></i>Поступило</span>
        <span class="tv-pulse-key"><i class="tv-pulse-dot tv-pulse-dot--closed"></i>Закрыто</span>
      </div>
      <div class="tv-pulse-chart">
        <div
          v-for="(d, i) in days"
          :key="d.date"
          class="tv-pulse-day"
          :style="{ '--row-delay': i * 70 + 'ms' }"
        >
          <div class="tv-pulse-bars">
            <div class="tv-pulse-bar tv-pulse-bar--received" :style="{ '--bar-h': heightPct(d.received) }">
              <span v-if="num(d.received)" class="tv-pulse-num">{{ d.received }}</span>
            </div>
            <div class="tv-pulse-bar tv-pulse-bar--closed" :style="{ '--bar-h': heightPct(d.closed) }">
              <span v-if="num(d.closed)" class="tv-pulse-num">{{ d.closed }}</span>
            </div>
          </div>
          <div class="tv-pulse-label">{{ dayLabel(d.date) }}</div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
// «Пульс недели»: вертикальные бары по дням — поступило vs закрыто,
// чтобы был виден баланс входящего и закрытого потока.
import { computed } from 'vue'
import { num } from '../tvFormat.js'

const props = defineProps({
  slide: { type: Object, required: true },
  calendar: { type: Array, default: () => [] }, // extended.calendar периода
})

const days = computed(() => props.calendar || [])
const maxVal = computed(() =>
  Math.max(1, ...days.value.flatMap(d => [num(d.received), num(d.closed)])))

function heightPct(v) {
  const n = num(v)
  // Нулевой день — тонкий «пенёк», чтобы ось читалась.
  if (!n) return '2%'
  return Math.max(8, Math.round((n / maxVal.value) * 100)) + '%'
}

function dayLabel(dateStr) {
  const d = new Date(dateStr)
  if (Number.isNaN(d.getTime())) return dateStr
  return d.toLocaleDateString('ru-RU', { weekday: 'short' })
}
</script>

<style scoped>
.tv-pulse-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 1.6vmin, 20px);
  min-height: 0;
}

.tv-pulse-legend {
  display: flex;
  gap: clamp(16px, 2.4vmin, 32px);
}

.tv-pulse-key {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: clamp(12px, 1.5vmin, 17px);
  font-weight: 700;
  color: var(--color-text-dim);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.tv-pulse-dot {
  width: clamp(10px, 1.2vmin, 14px);
  height: clamp(10px, 1.2vmin, 14px);
  border-radius: 4px;
}

.tv-pulse-dot--received { background: var(--color-primary); }
.tv-pulse-dot--closed   { background: var(--color-success); }

.tv-pulse-chart {
  flex: 1;
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: 1fr;
  gap: clamp(8px, 1.4vmin, 20px);
  align-items: stretch;
  min-height: 0;
  padding-top: clamp(20px, 2.6vmin, 34px); /* место под числа над барами */
}

.tv-pulse-day {
  display: flex;
  flex-direction: column;
  gap: clamp(6px, 0.8vmin, 10px);
  min-height: 0;
  animation: tv-row-in 0.6s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
  animation-delay: var(--row-delay, 0ms);
}

.tv-pulse-bars {
  flex: 1;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  gap: clamp(4px, 0.6vmin, 8px);
  min-height: 0;
  border-bottom: 2px solid var(--color-outline-dim);
}

.tv-pulse-bar {
  position: relative;
  width: clamp(14px, 2.4vmin, 36px);
  height: 0;
  border-radius: clamp(4px, 0.6vmin, 8px) clamp(4px, 0.6vmin, 8px) 0 0;
  animation: tv-pulse-grow 0.8s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
  animation-delay: calc(var(--row-delay, 0ms) + 150ms);
}

@keyframes tv-pulse-grow {
  from { height: 0; }
  to   { height: var(--bar-h, 0%); }
}

.tv-pulse-bar--received {
  background: linear-gradient(180deg,
    var(--color-primary),
    color-mix(in oklch, var(--color-primary) 65%, transparent));
}

.tv-pulse-bar--closed {
  background: linear-gradient(180deg,
    var(--color-success),
    color-mix(in oklch, var(--color-success) 65%, transparent));
}

.tv-pulse-num {
  position: absolute;
  top: calc(clamp(16px, 2vmin, 26px) * -1);
  left: 50%;
  transform: translateX(-50%);
  font-size: clamp(11px, 1.5vmin, 18px);
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-dim);
}

.tv-pulse-label {
  text-align: center;
  font-size: clamp(11px, 1.4vmin, 16px);
  font-weight: 700;
  color: var(--color-text-dim);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}
</style>
