<template>
  <label class="switch-row">
    <span class="switch-text">
      <span v-if="icon" class="material-symbols-outlined">{{ icon }}</span>
      <span>
        <strong>{{ title }}</strong>
        <small v-if="hint">{{ hint }}</small>
      </span>
    </span>
    <input
      type="checkbox"
      class="switch"
      :checked="modelValue"
      :disabled="disabled"
      @change="emit('update:modelValue', $event.target.checked)"
    />
  </label>
</template>

<script setup>
defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, required: true },
  hint: { type: String, default: '' },
  icon: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])
</script>

<style scoped>
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  background: var(--color-surface-container);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background 0.12s;
}

.switch-row:hover { background: var(--color-surface-high); }

.switch-text {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.switch-text .material-symbols-outlined {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 20px;
  flex: none;
}

.switch-text strong { display: block; font-size: 14px; color: var(--color-on-surface); }
.switch-text small { display: block; font-size: 12px; color: var(--color-on-surface-variant); }

.switch {
  appearance: none;
  width: 44px;
  height: 24px;
  min-width: 44px;
  max-width: 44px;
  min-height: 24px;
  max-height: 24px;
  border-radius: var(--radius-full);
  background: var(--color-surface-highest, var(--color-surface-high));
  border: 2px solid var(--color-outline, var(--color-outline-variant));
  box-sizing: border-box;
  position: relative;
  cursor: pointer;
  outline: none;
  transition: background 0.18s, border-color 0.18s;
  flex: none;
}

.switch::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 4px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--color-outline, var(--color-on-surface-variant));
  transform: translateY(-50%);
  transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1),
    background 0.2s, width 0.2s, height 0.2s, left 0.2s;
}

.switch:checked {
  background: var(--color-primary);
  border-color: var(--color-primary);
}

.switch:checked::after {
  width: 16px;
  height: 16px;
  left: 24px;
  background: var(--color-on-primary);
}

.switch:disabled { opacity: 0.5; cursor: default; }
</style>
