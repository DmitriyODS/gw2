<template>
  <div class="tv-kpi" :class="'tone-' + tone">
    <div class="tv-kpi-row">
      <span class="tv-kpi-ico material-symbols-outlined">{{ icon }}</span>
      <div class="tv-kpi-label">{{ label }}</div>
    </div>
    <div class="tv-kpi-value">
      <TvCount :value="Number(value) || 0" :format="format" :prefix="prefix" />
    </div>
  </div>
</template>

<script setup>
// Плитка KPI-рейла слева: иконка + подпись + анимированное значение.
import TvCount from './TvCount.vue'

defineProps({
  tone:   { type: String, default: 'primary' },
  icon:   { type: String, required: true },
  label:  { type: String, required: true },
  value:  { type: [Number, String], default: 0 },
  format: { type: String, default: 'int' },
  prefix: { type: String, default: '' },
})
</script>

<style scoped>
.tv-kpi {
  border-radius: clamp(14px, 1.6vmin, 22px);
  padding: clamp(12px, 1.6vmin, 20px);
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 8px;
  min-height: 0;
  position: relative;
  overflow: hidden;
}

.tv-kpi::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 100% 0%, var(--kpi-glow, transparent), transparent 65%);
  opacity: 0.6;
  pointer-events: none;
}

.tv-kpi.tone-primary   { --kpi-glow: color-mix(in oklch, var(--color-primary) 20%, transparent); }
.tv-kpi.tone-secondary { --kpi-glow: color-mix(in oklch, var(--color-secondary) 20%, transparent); }
.tv-kpi.tone-tertiary  { --kpi-glow: color-mix(in oklch, var(--color-tertiary) 20%, transparent); }
.tv-kpi.tone-success   { --kpi-glow: color-mix(in oklch, var(--color-success) 22%, transparent); }
.tv-kpi.tone-warning   { --kpi-glow: color-mix(in oklch, var(--color-warning) 22%, transparent); }
.tv-kpi.tone-error     { --kpi-glow: color-mix(in oklch, var(--color-error) 22%, transparent); }

.tv-kpi-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.tv-kpi-ico {
  font-size: clamp(20px, 2.2vmin, 26px);
  color: var(--color-text-dim);
}

.tv-kpi.tone-primary   .tv-kpi-ico { color: var(--color-primary); }
.tv-kpi.tone-secondary .tv-kpi-ico { color: var(--color-secondary); }
.tv-kpi.tone-tertiary  .tv-kpi-ico { color: var(--color-tertiary); }
.tv-kpi.tone-success   .tv-kpi-ico { color: var(--color-success); }
.tv-kpi.tone-warning   .tv-kpi-ico { color: var(--color-warning); }
.tv-kpi.tone-error     .tv-kpi-ico { color: var(--color-error); }

.tv-kpi-label {
  font-size: clamp(11px, 1.3vmin, 15px);
  color: var(--color-text-dim);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-weight: 700;
}

.tv-kpi-value {
  font-size: clamp(28px, 4.2vmin, 56px);
  font-weight: 800;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
}

.tv-kpi.tone-primary   .tv-kpi-value { color: var(--color-primary); }
.tv-kpi.tone-secondary .tv-kpi-value { color: var(--color-secondary); }
.tv-kpi.tone-tertiary  .tv-kpi-value { color: var(--color-tertiary); }
.tv-kpi.tone-success   .tv-kpi-value { color: var(--color-success); }
.tv-kpi.tone-warning   .tv-kpi-value { color: var(--color-warning); }
.tv-kpi.tone-error     .tv-kpi-value { color: var(--color-error); }
</style>
