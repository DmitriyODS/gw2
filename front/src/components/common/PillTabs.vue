<template>
  <!-- Пилюли-вкладки: стеклянная дорожка, активная вкладка — заливка
       primary-container. Общий элемент нового оформления разделов. -->
  <div class="pt" :class="[`align-${align}`, { compact }]" role="tablist">
    <button
      v-for="tab in tabs"
      :key="tab.key"
      class="pt-tab"
      :class="{ active: tab.key === modelValue }"
      role="tab"
      type="button"
      :aria-selected="tab.key === modelValue"
      :disabled="tab.disabled"
      @click="$emit('update:modelValue', tab.key)"
    >
      <span v-if="tab.icon" class="material-symbols-outlined">{{ tab.icon }}</span>
      <span class="pt-label">{{ tab.label }}</span>
      <span v-if="tab.badge" class="pt-badge">{{ tab.badge }}</span>
    </button>
  </div>
</template>

<script setup>
defineProps({
  /** [{ key, label, icon?, badge?, disabled? }] */
  tabs: { type: Array, required: true },
  modelValue: { type: [String, Number], default: '' },
  align: { type: String, default: 'center' }, // center | start
  compact: { type: Boolean, default: false },
})
defineEmits(['update:modelValue'])
</script>

<style scoped>
.pt {
  display: flex;
  gap: 4px;
  padding: 6px;
  border: 1px solid var(--acrylic-border);
  border-radius: 999px;
  background: var(--acrylic-card-bg);
  overflow-x: auto;
  scrollbar-width: none;
}

.pt::-webkit-scrollbar { display: none; }

.pt.align-center { justify-content: center; }
.pt.align-start { justify-content: flex-start; }

.pt-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border: none;
  border-radius: 999px;
  background: none;
  color: var(--color-text-dim);
  font-size: 0.9rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.18s ease, color 0.18s ease;
}

.pt.compact .pt-tab { padding: 7px 14px; font-size: 0.84rem; }

.pt-tab .material-symbols-outlined { font-size: 20px; }

.pt-tab:hover:not(.active):not(:disabled) {
  background: var(--color-surface-high);
  color: var(--color-text);
}

.pt-tab.active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.pt-tab:disabled { opacity: 0.45; cursor: default; }

.pt-badge {
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 0.72rem;
  font-weight: 700;
}

.pt-tab.active .pt-badge {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
</style>
