<template>
  <!-- Тумблер — та же строка настройки (`SettingRow`), только управление у неё
       компактное и остаётся справа при любой ширине. Нажимается вся строка. -->
  <SettingRow
    :title="title"
    :hint="hint"
    :disabled="disabled"
    clickable
    inline
    @click="toggle"
  >
    <span class="switch" :class="{ on: modelValue }" role="switch" :aria-checked="modelValue" />
  </SettingRow>
</template>

<script setup>
import SettingRow from './SettingRow.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, required: true },
  hint: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue'])

function toggle() {
  if (!props.disabled) emit('update:modelValue', !props.modelValue)
}
</script>

<style scoped>
.switch {
  position: relative;
  width: 44px;
  min-width: 44px;
  max-width: 44px;
  height: 24px;
  min-height: 24px;
  max-height: 24px;
  box-sizing: border-box;
  border: 2px solid var(--color-outline, var(--color-outline-variant));
  border-radius: var(--radius-full);
  background: var(--color-surface-highest, var(--color-surface-high));
  transition: background 0.18s, border-color 0.18s;
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

.switch.on {
  background: var(--color-primary);
  border-color: var(--color-primary);
}

.switch.on::after {
  width: 16px;
  height: 16px;
  left: 24px;
  background: var(--color-on-primary);
}
</style>
