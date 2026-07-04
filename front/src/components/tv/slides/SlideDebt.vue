<template>
  <div class="tv-debt">
    <div class="tv-stage-eyebrow">
      <span class="material-symbols-outlined" style="color: var(--color-warning)">assignment_late</span>
      <span>{{ slide.heroEyebrow }}</span>
    </div>
    <div class="tv-debt-glow" :class="{ 'is-heavy': isHeavy }">
      <div class="tv-debt-number" :class="{ 'is-heavy': isHeavy }">
        <TvCount :value="debt" format="int" />
      </div>
    </div>
    <div class="tv-debt-caption">задач ждут дольше срока</div>
    <div class="tv-debt-note">Разгружаем хвост вместе — каждая закрытая задача уменьшает это число</div>
    <div class="tv-debt-secondaries">
      <div class="tv-debt-sec">
        <div class="tv-debt-sec-label">Закрыто за период</div>
        <div class="tv-debt-sec-value tone-success"><TvCount :value="closed" prefix="−" /></div>
      </div>
      <div class="tv-debt-sec">
        <div class="tv-debt-sec-label">В работе</div>
        <div class="tv-debt-sec-value tone-tertiary"><TvCount :value="remaining" /></div>
      </div>
    </div>
  </div>
</template>

<script setup>
// «Фокус недели»: сколько задач ждут дольше срока. Подача нейтральная —
// не доска позора, а общая цель команды разгрузить хвост.
import { computed } from 'vue'
import TvCount from '../TvCount.vue'
import { num } from '../tvFormat.js'

const props = defineProps({
  slide: { type: Object, required: true },
  debt: { type: Number, default: 0 },
  closed: { type: Number, default: 0 },
  remaining: { type: Number, default: 0 },
})

// Небольшой долг — warning, ощутимый — error.
const isHeavy = computed(() => num(props.debt) >= 10)
</script>

<style scoped>
.tv-debt {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: clamp(10px, 1.4vmin, 18px);
  text-align: center;
}

.tv-debt-glow {
  position: relative;
  padding: clamp(20px, 3vmin, 40px) clamp(40px, 6vmin, 80px);
}

.tv-debt-glow::before {
  content: '';
  position: absolute;
  inset: -10%;
  background: radial-gradient(circle, color-mix(in oklch, var(--color-warning) 26%, transparent), transparent 65%);
  filter: blur(8px);
  z-index: 0;
  pointer-events: none;
}

.tv-debt-glow.is-heavy::before {
  background: radial-gradient(circle, color-mix(in oklch, var(--color-error) 26%, transparent), transparent 65%);
}

.tv-debt-number {
  position: relative;
  z-index: 1;
  font-size: clamp(72px, 16vmin, 240px);
  font-weight: 900;
  line-height: 0.9;
  letter-spacing: -0.04em;
  font-variant-numeric: tabular-nums;
  color: var(--color-warning);
}

.tv-debt-number.is-heavy { color: var(--color-error); }

.tv-debt-caption {
  font-size: clamp(16px, 2vmin, 26px);
  font-weight: 700;
  color: var(--color-text);
}

.tv-debt-note {
  font-size: clamp(13px, 1.6vmin, 19px);
  color: var(--color-text-dim);
  max-width: 560px;
  line-height: 1.4;
}

.tv-debt-secondaries {
  display: flex;
  gap: clamp(20px, 3vmin, 50px);
  margin-top: clamp(8px, 1.2vmin, 18px);
}

.tv-debt-sec {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: center;
}

.tv-debt-sec-label {
  font-size: clamp(11px, 1.2vmin, 14px);
  color: var(--color-text-dim);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-weight: 700;
}

.tv-debt-sec-value {
  font-size: clamp(28px, 4vmin, 56px);
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.tv-debt-sec-value.tone-success  { color: var(--color-success); }
.tv-debt-sec-value.tone-tertiary { color: var(--color-tertiary); }
</style>
