<template>
  <div
    class="app-tabs"
    :class="[`v-${variant}`, `align-${align}`, { full: fullWidth, dense }]"
    role="tablist"
  >
    <button
      v-for="t in items"
      :key="t.value"
      class="app-tab"
      :class="{ active: t.value === modelValue }"
      type="button"
      role="tab"
      :aria-selected="t.value === modelValue"
      :disabled="t.disabled"
      @click="select(t)"
    >
      <span v-if="t.icon" class="material-symbols-outlined">{{ t.icon }}</span>
      <span v-if="t.label" class="app-tab-label">{{ t.label }}</span>
      <span v-if="t.badge" class="app-tab-badge">{{ t.badge }}</span>
    </button>
  </div>
</template>

<script setup>
/* Единственный переключатель вкладок платформы: свёл `PillTabs` и
   `SegmentedTabs`, которые различались только оформлением дорожки.

   variant: solid — главный переключатель режима раздела (активная вкладка на
   фирменном градиенте); tint — второстепенный, внутри карточки или тулбара
   (активная вкладка тинтованной пилюлей). */
import { computed } from 'vue'

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  /** [{ value, label, icon?, badge?, disabled? }] — `key` принимается как синоним value. */
  tabs: { type: Array, required: true },
  variant: { type: String, default: 'solid', validator: (v) => ['solid', 'tint'].includes(v) },
  align: { type: String, default: 'start', validator: (v) => ['start', 'center'].includes(v) },
  fullWidth: { type: Boolean, default: false },
  dense: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'change'])

// `key` — наследие PillTabs: вкладки половины разделов описаны через него.
const items = computed(() => props.tabs.map((t) => ({ ...t, value: t.value ?? t.key })))

function select(t) {
  if (t.disabled || t.value === props.modelValue) return
  emit('update:modelValue', t.value)
  emit('change', t.value)
}
</script>

<style scoped>
.app-tabs {
  display: inline-flex;
  gap: 3px;
  align-self: flex-start;
  max-width: 100%;
  padding: 4px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  /* Вкладок может быть больше, чем влезает в ряд, — горизонтальный скролл
     вместо переноса или обрезки. */
  overflow-x: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.app-tabs::-webkit-scrollbar { display: none; }

.app-tabs.full { display: flex; align-self: stretch; }
.app-tabs.align-center { justify-content: center; }
.app-tabs.full .app-tab { flex: 1; }

.app-tab {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 36px;
  padding: 8px 18px;
  border: none;
  border-radius: var(--radius-full);
  background: none;
  color: var(--color-text-dim);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.18s, color 0.18s, box-shadow 0.18s;
}

.app-tabs.dense .app-tab { min-height: 32px; padding: 6px 12px; font-size: 13px; }

.app-tab .material-symbols-outlined {
  font-size: 18px;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 20;
}

.app-tab:hover:not(.active):not(:disabled) { color: var(--color-text); }
.app-tab:disabled { opacity: 0.45; cursor: default; }

.v-solid .app-tab.active {
  background: var(--grad-primary);
  color: var(--color-on-primary);
  box-shadow: var(--shadow-sm);
}

.v-tint .app-tab.active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.app-tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: var(--radius-full);
  background: var(--color-error);
  color: var(--color-on-error);
  font-size: 11px;
  font-weight: 700;
}

.v-solid .app-tab.active .app-tab-badge { background: var(--color-surface); color: var(--color-primary); }
.v-tint .app-tab.active .app-tab-badge { background: var(--color-primary); color: var(--color-on-primary); }

/* Подписи на телефоне НЕ прячем (иконки без текста читаются плохо) — ужимаем
   типографику и отступы. */
@media (max-width: 768px) {
  .app-tab { min-height: 44px; padding: 10px 12px; font-size: 13px; }
}

@media (max-width: 480px) {
  .app-tab { padding: 10px 8px; gap: 5px; font-size: 12px; }
}
</style>
