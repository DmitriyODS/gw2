<template>
  <!-- Три карточки состояния подписки: тариф, хранилище, токены ИИ. -->
  <section class="gw-card stat-wrap">
    <div class="stat-grid">
      <article class="stat">
        <p class="stat-label">Моя подписка</p>
        <div class="stat-body stat-body--plan">
          <span class="plan-tag">тариф</span>
          <strong class="plan-name">{{ planName }}</strong>
          <span v-if="until" class="gw-sub">действует до {{ until }}</span>
          <span v-else-if="isFree" class="gw-sub">бесплатный тариф</span>
        </div>
        <button class="stat-action" type="button" @click="$emit('manage', 'plan')">
          Управление подпиской
        </button>
      </article>

      <article class="stat">
        <p class="stat-label">Хранилище</p>
        <div class="stat-body stat-body--metric">
          <span class="metric-cap">занято</span>
          <p class="metric">
            {{ formatBytes(storageUsed) }} из <b>{{ storageLimit < 0 ? '∞' : formatBytes(storageLimit) }}</b>
          </p>
          <div class="bar"><span :style="{ width: pct(storageUsed, storageLimit) }" /></div>
        </div>
        <button class="stat-action" type="button" @click="$emit('manage', 'storage')">
          Управление местом
        </button>
      </article>

      <article class="stat">
        <p class="stat-label">ИИ возможности</p>
        <div class="stat-body stat-body--metric">
          <span class="metric-cap">использовано</span>
          <p class="metric">
            {{ formatCount(tokensUsed) }} из <b>{{ formatCount(tokensLimit) }} токенов</b>
          </p>
          <div class="bar"><span :style="{ width: pct(tokensUsed, tokensLimit) }" /></div>
        </div>
        <button class="stat-action" type="button" @click="$emit('manage', 'tokens')">
          Управление лимитами
        </button>
      </article>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import { formatBytes, formatCount, formatUntil, usageRatio } from '@/utils/money.js'

const props = defineProps({
  planName: { type: String, default: 'Джун' },
  isFree: { type: Boolean, default: true },
  expiresAt: { type: String, default: null },
  storageUsed: { type: Number, default: 0 },
  storageLimit: { type: Number, default: -1 },
  tokensUsed: { type: Number, default: 0 },
  tokensLimit: { type: Number, default: 0 },
})

defineEmits(['manage'])

const until = computed(() => formatUntil(props.expiresAt))

function pct(used, limit) {
  return `${Math.round(usageRatio(used, limit) * 100)}%`
}
</script>

<style scoped>
.stat-wrap { padding: 14px; }

.stat-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

.stat {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 200px;
  padding: 18px;
  border-radius: var(--radius-lg);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
}

.stat-label {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--color-text-dim);
}

.stat-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  justify-content: center;
  gap: 8px;
  min-height: 0;
}

.stat-body--metric { align-items: flex-end; text-align: right; }

.plan-tag {
  align-self: flex-start;
  padding: 4px 12px;
  border-radius: 999px;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 0.75rem;
  font-weight: 600;
}

.plan-name {
  font-size: clamp(1.8rem, 4cqw, 2.6rem);
  font-weight: 800;
  line-height: 1.05;
  color: var(--color-primary);
}

.metric-cap {
  font-size: 1.05rem;
  color: var(--color-text-dim);
}

.metric {
  margin: 0;
  font-size: clamp(1.1rem, 2.4cqw, 1.6rem);
  font-weight: 500;
}

.metric b {
  font-weight: 800;
  color: var(--color-primary);
}

.bar {
  width: 100%;
  height: 12px;
  border-radius: 999px;
  background: var(--color-surface-variant);
  overflow: hidden;
}

.bar span {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--color-primary);
  transition: width 0.3s ease;
}

.stat-action {
  padding: 11px 16px;
  border: none;
  border-radius: 999px;
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font-size: 0.9rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s ease;
}

.stat-action:hover { background: var(--glass-hover-bg, var(--glass-bg)), var(--acrylic-card-bg); }

/* Раскладка считается от ширины ОКНА раздела (.gw-shell — container). */
@container (max-width: 900px) {
  .stat-grid { grid-template-columns: 1fr; }
  .stat { min-height: 0; }
}

/* Дубль для старого WebView, который не знает @container. */
@media (max-width: 900px) {
  .stat-grid { grid-template-columns: 1fr; }
}
</style>
