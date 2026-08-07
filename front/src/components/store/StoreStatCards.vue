<template>
  <!-- Три карточки состояния подписки: тариф, хранилище, токены ИИ. -->
  <!-- Карточек ровно три и они делят ширину поровну, поэтому сетка своя:
       auto-fill AppGrid оставил бы хвост ряда пустым. -->
  <div class="stat-grid">
    <AppCard title="Моя подписка" class="stat">
      <div class="stat-body stat-body--plan">
        <span class="plan-tag">тариф</span>
        <strong class="plan-name">{{ planName }}</strong>
        <span v-if="until" class="stat-note">действует до {{ until }}</span>
        <span v-else-if="isFree" class="stat-note">бесплатный тариф</span>
      </div>
      <AppButton label="Управление подпиской" block @click="$emit('manage', 'plan')" />
    </AppCard>

    <AppCard title="Хранилище" class="stat">
      <div class="stat-body stat-body--metric">
        <span class="metric-cap">занято</span>
        <p class="metric">
          {{ formatBytes(storageUsed) }} из <b>{{ storageLimit < 0 ? '∞' : formatBytes(storageLimit) }}</b>
        </p>
        <div class="bar"><span :style="{ width: pct(storageUsed, storageLimit) }" /></div>
      </div>
      <AppButton label="Управление местом" block @click="$emit('manage', 'storage')" />
    </AppCard>

    <AppCard title="ИИ возможности" class="stat">
      <div class="stat-body stat-body--metric">
        <span class="metric-cap">использовано</span>
        <p class="metric">
          {{ formatCount(tokensUsed) }} из <b>{{ formatCount(tokensLimit) }} токенов</b>
        </p>
        <div class="bar"><span :style="{ width: pct(tokensUsed, tokensLimit) }" /></div>
      </div>
      <AppButton label="Управление лимитами" block @click="$emit('manage', 'tokens')" />
    </AppCard>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
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
.stat-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

/* Табло — высокая карточка: цифра посередине, кнопка прижата к низу. */
.stat { min-height: 200px; }

.stat-note { font-size: 0.85rem; color: var(--color-text-dim); }

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

/* Раскладка считается от ширины ОКНА раздела (панель AppPage — container). */
@container (max-width: 900px) {
  .stat-grid { grid-template-columns: 1fr; }
  .stat { min-height: 0; }
}

/* Дубль для старого WebView, который не знает @container. */
@media (max-width: 900px) {
  .stat-grid { grid-template-columns: 1fr; }
}
</style>
