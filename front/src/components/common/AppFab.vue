<template>
  <Teleport to="body">
    <button
      v-if="visible"
      class="fab float-fade"
      :class="[
        `fab--${tone}`,
        { 'fab--collapsed': collapsed, 'fab--icon-only': !label, 'float-hidden': floatingHidden },
      ]"
      :data-tutorial="tutorial || null"
      :aria-label="ariaLabel || label || icon"
      @click="$emit('click', $event)"
    >
      <span class="material-symbols-outlined">{{ icon }}</span>
      <span v-if="label" class="fab-label">{{ label }}</span>
    </button>
  </Teleport>
</template>

<script setup>
import { onMounted } from 'vue'
import { floatingHidden, installFloatingHide } from '@/composables/useFloatingHide.js'

onMounted(installFloatingHide)

defineProps({
  icon: { type: String, default: 'add' },
  label: { type: String, default: '' },
  collapsed: { type: Boolean, default: false },
  visible: { type: Boolean, default: true },
  ariaLabel: { type: String, default: '' },
  tutorial: { type: String, default: '' },
  /* primary | tertiary */
  tone: { type: String, default: 'primary' },
})
defineEmits(['click'])
</script>

<style scoped>
@media (max-width: 768px) {
  .fab {
    position: fixed;
    right: 16px;
    bottom: calc(64px + 16px + env(safe-area-inset-bottom, 0px));
    height: 56px;
    min-width: 56px;
    padding: 0 22px 0 18px;
    border: none;
    border-radius: 28px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 10px;
    font: inherit;
    font-size: 15px;
    font-weight: 650;
    letter-spacing: 0.2px;
    z-index: 150;
    transition: padding 0.26s cubic-bezier(0.34, 1.36, 0.64, 1),
                min-width 0.26s cubic-bezier(0.34, 1.36, 0.64, 1),
                border-radius 0.26s cubic-bezier(0.34, 1.36, 0.64, 1),
                background 0.15s, box-shadow 0.2s, transform 0.12s,
                opacity 0.22s ease;
  }

  .fab--primary {
    background: var(--grad-primary);
    color: var(--color-on-primary);
    box-shadow:
      0 6px 16px color-mix(in oklch, var(--color-primary) 38%, transparent),
      0 2px 6px color-mix(in oklch, var(--color-primary) 20%, transparent);
  }
  .fab--primary:active {
    transform: scale(0.96);
    filter: brightness(1.06);
  }

  .fab--tertiary {
    background: var(--color-tertiary-container);
    color: var(--color-on-tertiary-container);
    box-shadow:
      0 6px 16px color-mix(in oklch, var(--color-tertiary) 38%, transparent),
      0 2px 6px color-mix(in oklch, var(--color-tertiary) 20%, transparent);
  }
  .fab--tertiary:active {
    transform: scale(0.96);
    background: color-mix(in oklch, var(--color-tertiary) 22%, var(--color-tertiary-container));
  }

  .fab .material-symbols-outlined {
    font-size: 24px;
    font-variation-settings: 'FILL' 0, 'wght' 500;
  }

  .fab-label {
    white-space: nowrap;
    overflow: hidden;
    max-width: 160px;
    opacity: 1;
    transition: max-width 0.26s cubic-bezier(0.4, 0, 0.2, 1),
                opacity 0.18s ease;
  }

  .fab--collapsed,
  .fab--icon-only {
    padding: 0;
    min-width: 56px;
    width: 56px;
    border-radius: 50%;
    justify-content: center;
    gap: 0;
  }

  .fab--collapsed .fab-label {
    max-width: 0;
    opacity: 0;
  }
}

@media (min-width: 769px) {
  .fab { display: none; }
}
</style>
