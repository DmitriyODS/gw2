<template>
  <div
    class="seg-tabs"
    :class="{
      'seg-tabs--full-width': fullWidth,
      'seg-tabs--dense': dense,
    }"
    role="tablist"
  >
    <button
      v-for="t in tabs"
      :key="t.value"
      type="button"
      class="seg-tab"
      :class="{ active: t.value === modelValue }"
      :data-tutorial="t.tutorial || null"
      role="tab"
      :aria-selected="t.value === modelValue"
      @click="select(t)"
    >
      <span v-if="t.icon" class="material-symbols-outlined">{{ t.icon }}</span>
      <span v-if="t.label" class="seg-tab-label">{{ t.label }}</span>
      <span v-if="t.badge" class="seg-tab-badge">{{ t.badge }}</span>
    </button>
  </div>
</template>

<script setup>
const props = defineProps({
  modelValue: { type: [String, Number], required: true },
  /* [{ value, label, icon?, badge?, tutorial? }] */
  tabs: { type: Array, required: true },
  fullWidth: { type: Boolean, default: false },
  /* На мобиле скрывает подписи (для is-compact в Tasks) */
  dense: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'change'])

function select(t) {
  if (t.value === props.modelValue) return
  emit('update:modelValue', t.value)
  emit('change', t.value)
}
</script>

<style scoped>
.seg-tabs {
  display: inline-flex;
  gap: 2px;
  background: var(--color-surface-high);
  border-radius: var(--radius-full);
  padding: 4px;
  align-self: flex-start;
  max-width: 100%;
}

.seg-tabs--full-width {
  align-self: stretch;
  display: flex;
}

.seg-tab {
  appearance: none;
  border: none;
  background: transparent;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 18px;
  min-height: 36px;
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-dim);
  border-radius: var(--radius-full);
  transition: background 0.18s, color 0.18s, box-shadow 0.18s;
  white-space: nowrap;
}

.seg-tabs--full-width .seg-tab { flex: 1; }

.seg-tab .material-symbols-outlined {
  font-size: 18px;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 20;
}

.seg-tab:hover:not(.active) { color: var(--color-text); }

.seg-tab.active {
  background: var(--color-surface);
  color: var(--color-primary);
  box-shadow: var(--shadow-sm);
}

.seg-tab-badge {
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

.seg-tab.active .seg-tab-badge {
  background: var(--color-primary);
  color: var(--color-on-primary);
}

@media (max-width: 768px) {
  .seg-tab {
    padding: 10px 8px;
    min-height: 44px;
    font-size: 13px;
  }
  .seg-tab-label { font-size: 13px; }
}

@media (max-width: 480px) {
  .seg-tab-label { font-size: 12px; }
  .seg-tabs--dense .seg-tab-label { display: none; }
}

@media (max-width: 360px) {
  .seg-tab-label { display: none; }
}
</style>
