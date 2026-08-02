<template>
  <!-- Тумблер — та же строка, только управление компактное и остаётся справа при
       любой ширине. Нажимается вся строка. -->
  <AppRow
    :title="title"
    :hint="hint"
    :icon="icon"
    :disabled="disabled"
    :dense="dense"
    :plain="plain"
    clickable
    inline
    @click="toggle"
  >
    <template v-if="$slots.hint" #hint><slot name="hint" /></template>
    <template v-if="$slots.lead" #lead><slot name="lead" /></template>

    <AppSwitch :model-value="modelValue" :disabled="disabled" @update:model-value="emit('update:modelValue', $event)" />
  </AppRow>
</template>

<script setup>
import AppRow from './AppRow.vue'
import AppSwitch from './AppSwitch.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, required: true },
  hint: { type: String, default: '' },
  icon: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  dense: { type: Boolean, default: false },
  plain: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue'])

function toggle() {
  if (!props.disabled) emit('update:modelValue', !props.modelValue)
}
</script>
