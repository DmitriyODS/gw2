<template>
  <div class="tv-aside-card" :class="'tone-' + (slide.asideTone || 'primary')">
    <div class="tv-aside-eyebrow">
      <span class="material-symbols-outlined">{{ slide.asideIcon || 'auto_awesome' }}</span>
      {{ slide.asideTitle || 'Контекст' }}
    </div>
    <div v-if="content" class="tv-aside-body">
      <div v-if="content.headline" class="tv-aside-headline">{{ content.headline }}</div>
      <div v-if="content.value != null" class="tv-aside-value">
        <TvCount :value="content.value" :format="content.format || 'int'" :prefix="content.prefix || ''" />
      </div>
      <div v-if="content.sub" class="tv-aside-sub">{{ content.sub }}</div>

      <!-- Спарклайн -->
      <div v-if="content.sparkline?.length" class="tv-spark">
        <svg viewBox="0 0 100 40" preserveAspectRatio="none">
          <polyline class="tv-spark-line" :points="sparklinePoints(content.sparkline)" />
          <polygon class="tv-spark-area" :points="sparklineArea(content.sparkline)" />
        </svg>
      </div>
    </div>
    <div v-else class="tv-aside-body">
      <div class="tv-aside-headline">—</div>
    </div>
  </div>
</template>

<script setup>
// Контекстная карточка справа от сцены: заголовок-тон + значение + спарклайн.
import TvCount from './TvCount.vue'
import { num } from './tvFormat.js'

defineProps({
  slide: { type: Object, required: true },
  content: { type: Object, default: null },
})

function sparklinePoints(arr) {
  if (!arr || arr.length === 0) return ''
  const max = Math.max(1, ...arr)
  const n = arr.length
  return arr.map((v, i) => {
    const x = (i / Math.max(1, n - 1)) * 100
    const y = 38 - (num(v) / max) * 34
    return `${x.toFixed(2)},${y.toFixed(2)}`
  }).join(' ')
}

function sparklineArea(arr) {
  const pts = sparklinePoints(arr)
  if (!pts) return ''
  return `0,40 ${pts} 100,40`
}
</script>

<style scoped>
.tv-aside-card {
  flex: 1;
  border-radius: clamp(18px, 2.4vmin, 28px);
  padding: clamp(18px, 2.2vmin, 28px);
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  display: flex;
  flex-direction: column;
  gap: clamp(10px, 1.4vmin, 18px);
  position: relative;
  overflow: hidden;
  min-height: 0;
}

.tv-aside-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 100% 0%,
    var(--aside-glow, color-mix(in oklch, var(--color-primary) 22%, transparent)),
    transparent 60%);
  pointer-events: none;
}

.tv-aside-card.tone-primary   { --aside-glow: color-mix(in oklch, var(--color-primary) 22%, transparent); }
.tv-aside-card.tone-secondary { --aside-glow: color-mix(in oklch, var(--color-secondary) 22%, transparent); }
.tv-aside-card.tone-tertiary  { --aside-glow: color-mix(in oklch, var(--color-tertiary) 22%, transparent); }
.tv-aside-card.tone-success   { --aside-glow: color-mix(in oklch, var(--color-success) 24%, transparent); }
.tv-aside-card.tone-warning   { --aside-glow: color-mix(in oklch, var(--color-warning) 24%, transparent); }
.tv-aside-card.tone-error     { --aside-glow: color-mix(in oklch, var(--color-error) 24%, transparent); }

.tv-aside-eyebrow {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: clamp(11px, 1.3vmin, 14px);
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

.tv-aside-eyebrow .material-symbols-outlined {
  font-size: clamp(16px, 1.8vmin, 22px);
  color: var(--color-primary);
}

.tv-aside-card.tone-secondary .tv-aside-eyebrow .material-symbols-outlined { color: var(--color-secondary); }
.tv-aside-card.tone-tertiary  .tv-aside-eyebrow .material-symbols-outlined { color: var(--color-tertiary); }
.tv-aside-card.tone-success   .tv-aside-eyebrow .material-symbols-outlined { color: var(--color-success); }
.tv-aside-card.tone-warning   .tv-aside-eyebrow .material-symbols-outlined { color: var(--color-warning); }
.tv-aside-card.tone-error     .tv-aside-eyebrow .material-symbols-outlined { color: var(--color-error); }

.tv-aside-body {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: clamp(8px, 1.2vmin, 14px);
  flex: 1;
  min-height: 0;
}

.tv-aside-headline {
  font-size: clamp(16px, 2.2vmin, 26px);
  font-weight: 700;
  color: var(--color-text);
  line-height: 1.2;
  word-break: break-word;
}

.tv-aside-value {
  font-size: clamp(40px, 6.4vmin, 96px);
  font-weight: 900;
  line-height: 0.95;
  letter-spacing: -0.03em;
  color: var(--color-primary);
  font-variant-numeric: tabular-nums;
}

.tv-aside-card.tone-secondary .tv-aside-value { color: var(--color-secondary); }
.tv-aside-card.tone-tertiary  .tv-aside-value { color: var(--color-tertiary); }
.tv-aside-card.tone-success   .tv-aside-value { color: var(--color-success); }
.tv-aside-card.tone-warning   .tv-aside-value { color: var(--color-warning); }
.tv-aside-card.tone-error     .tv-aside-value { color: var(--color-error); }

.tv-aside-sub {
  font-size: clamp(12px, 1.4vmin, 16px);
  color: var(--color-text-dim);
}

.tv-spark {
  flex: 1;
  min-height: clamp(60px, 8vmin, 110px);
  margin-top: auto;
}

.tv-spark svg { width: 100%; height: 100%; display: block; }

.tv-spark-line {
  fill: none;
  stroke: currentColor;
  stroke-width: 1.6;
  vector-effect: non-scaling-stroke;
  color: var(--color-primary);
  stroke-linecap: round;
  stroke-linejoin: round;
}

.tv-aside-card.tone-secondary .tv-spark-line { color: var(--color-secondary); }
.tv-aside-card.tone-tertiary  .tv-spark-line { color: var(--color-tertiary); }
.tv-aside-card.tone-success   .tv-spark-line { color: var(--color-success); }

.tv-spark-area {
  fill: currentColor;
  color: var(--color-primary);
  opacity: 0.18;
}

.tv-aside-card.tone-secondary .tv-spark-area { color: var(--color-secondary); }
.tv-aside-card.tone-tertiary  .tv-spark-area { color: var(--color-tertiary); }
.tv-aside-card.tone-success   .tv-spark-area { color: var(--color-success); }
</style>
