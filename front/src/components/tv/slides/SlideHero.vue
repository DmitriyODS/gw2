<template>
  <div class="tv-hero">
    <div class="tv-stage-eyebrow">
      <span class="material-symbols-outlined" :style="{ color: toneColor(slide.tone) }">
        {{ slide.heroIcon }}
      </span>
      <span>{{ slide.heroEyebrow }}</span>
    </div>
    <div class="tv-hero-glow" :style="{ '--glow': toneColor(slide.tone) }">
      <div class="tv-hero-number" :style="{ color: toneColor(slide.tone) }">
        <TvCount :value="heroValue(slide.heroKey)" :format="slide.heroFormat || 'int'" />
      </div>
    </div>
    <div class="tv-hero-caption">{{ slide.heroCaption }}</div>
    <div v-if="secondaries.length" class="tv-hero-secondaries">
      <div v-for="(s, i) in secondaries" :key="i" class="tv-hero-sec">
        <div class="tv-hero-sec-label">{{ s.label }}</div>
        <div class="tv-hero-sec-value" :style="{ color: s.tone ? toneColor(s.tone) : '' }">
          <TvCount :value="s.value" :format="s.format || 'int'" :prefix="s.prefix || ''" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
// Слайд «одно большое число»: heroKey из common.tasks или total_hours.
import { computed } from 'vue'
import TvCount from '../TvCount.vue'
import { num, toneColor } from '../tvFormat.js'

const props = defineProps({
  slide: { type: Object, required: true },
  common: { type: Object, default: null },
  totalHours: { type: Number, default: 0 },
})

function heroValue(key) {
  if (key === 'total_hours') return props.totalHours
  const t = props.common?.tasks
  return t ? num(t[key]) : 0
}

const secondaries = computed(() =>
  (props.slide.secondaries || []).map(s => ({
    label: s.label,
    value: heroValue(s.key),
    format: s.format || 'int',
    tone: s.tone,
    prefix: s.prefix || '',
  })))
</script>

<style scoped>
.tv-hero {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: clamp(10px, 1.4vmin, 18px);
  text-align: center;
}

.tv-hero-glow {
  position: relative;
  padding: clamp(20px, 3vmin, 40px) clamp(40px, 6vmin, 80px);
}

.tv-hero-glow::before {
  content: '';
  position: absolute;
  inset: -10%;
  background: radial-gradient(circle, color-mix(in oklch, var(--glow) 28%, transparent), transparent 65%);
  filter: blur(8px);
  z-index: 0;
  pointer-events: none;
}

.tv-hero-number {
  position: relative;
  z-index: 1;
  font-size: clamp(72px, 16vmin, 240px);
  font-weight: 900;
  line-height: 0.9;
  letter-spacing: -0.04em;
  font-variant-numeric: tabular-nums;
}

.tv-hero-caption {
  font-size: clamp(14px, 1.8vmin, 22px);
  color: var(--color-text-dim);
  max-width: 560px;
  line-height: 1.4;
}

.tv-hero-secondaries {
  display: flex;
  gap: clamp(20px, 3vmin, 50px);
  margin-top: clamp(8px, 1.2vmin, 18px);
}

.tv-hero-sec {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: center;
}

.tv-hero-sec-label {
  font-size: clamp(11px, 1.2vmin, 14px);
  color: var(--color-text-dim);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-weight: 700;
}

.tv-hero-sec-value {
  font-size: clamp(28px, 4vmin, 56px);
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}
</style>
