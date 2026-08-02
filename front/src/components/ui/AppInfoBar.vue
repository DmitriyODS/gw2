<template>
  <div class="infobar" :class="[`tone-${tone}`, { inline }]" role="status">
    <span class="material-symbols-outlined bar-icon">{{ icon || defaultIcon }}</span>

    <div class="bar-text">
      <strong v-if="title" class="bar-title">{{ title }}</strong>
      <span v-if="message || slots.default" class="bar-msg"><slot>{{ message }}</slot></span>
    </div>

    <div v-if="slots.actions" class="bar-actions"><slot name="actions" /></div>

    <button v-if="closable" class="bar-x" type="button" aria-label="Закрыть" @click="$emit('close')">
      <span class="material-symbols-outlined">close</span>
    </button>
  </div>
</template>

<script setup>
/* Полоса-сообщение внутри раздела: ошибка загрузки, предупреждение о лимите,
   подсказка о состоянии. Заменила разнобой из `.error-block`, `.state-block`,
   `.gw-banner` и локальных плашек — они отличались не смыслом, а вёрсткой. */
import { computed, useSlots } from 'vue'

const props = defineProps({
  tone: {
    type: String,
    default: 'info',
    validator: (v) => ['info', 'success', 'warning', 'error'].includes(v),
  },
  title: { type: String, default: '' },
  message: { type: String, default: '' },
  icon: { type: String, default: '' },
  closable: { type: Boolean, default: false },
  /** Компактная строка внутри карточки, а не отдельный блок раздела. */
  inline: { type: Boolean, default: false },
})

defineEmits(['close'])
const slots = useSlots()

const defaultIcon = computed(() => ({
  info: 'info',
  success: 'check_circle',
  warning: 'warning',
  error: 'error',
}[props.tone]))
</script>

<style scoped>
.infobar {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 16px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface-high);
  color: var(--color-text);
  font-size: 0.88rem;
  line-height: 1.45;
}

.infobar.inline { padding: 10px 12px; border-radius: var(--radius-md); font-size: 0.83rem; }

.bar-icon { font-size: 21px; flex-shrink: 0; }
.bar-text { display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0; }
.bar-title { font-size: 0.92rem; font-weight: 600; }
.bar-msg { color: inherit; opacity: 0.9; overflow-wrap: anywhere; }

.bar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.bar-x {
  display: grid;
  place-items: center;
  width: 26px; min-width: 26px; max-width: 26px;
  height: 26px; min-height: 26px; max-height: 26px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: none;
  color: inherit;
  opacity: 0.7;
  cursor: pointer;
}

.bar-x:hover { opacity: 1; background: color-mix(in oklch, currentColor 12%, transparent); }
.bar-x .material-symbols-outlined { font-size: 17px; }

.tone-info { border-color: color-mix(in oklch, var(--color-primary) 28%, var(--acrylic-border)); }
.tone-info .bar-icon { color: var(--color-primary); }

.tone-success {
  background: var(--color-success-container);
  border-color: transparent;
  color: var(--color-on-success-container);
}

.tone-warning {
  background: var(--color-warning-container);
  border-color: transparent;
  color: var(--color-on-warning-container);
}

.tone-error {
  background: var(--color-error-container);
  border-color: transparent;
  color: var(--color-on-error-container);
}

/* Узкая полоса: действия уезжают под текст и растягиваются на всю ширину. */
@container (max-width: 460px) {
  .infobar { flex-wrap: wrap; }
  .bar-actions { width: 100%; }
  .bar-actions > :deep(*) { flex: 1; }
}

@media (max-width: 560px) {
  .infobar { flex-wrap: wrap; }
  .bar-actions { width: 100%; }
  .bar-actions > :deep(*) { flex: 1; }
}
</style>
