<template>
  <DatePicker
    :model-value="modelValue"
    @update:model-value="$emit('update:modelValue', $event)"
    selection-mode="range"
    date-format="dd.mm.yy"
    show-icon
    :show-button-bar="true"
    placeholder="Выберите период"
  />
</template>

<script setup>
import DatePicker from 'primevue/datepicker'

defineProps({
  modelValue: {
    type: Array,
    default: null
  }
})

defineEmits(['update:modelValue'])
</script>

<style scoped>
:deep(.p-datepicker) {
  border-radius: var(--gw-radius);
  border-color: var(--gw-border);
}

:deep(.p-datepicker:not(.p-disabled):hover) {
  border-color: var(--gw-primary);
}
</style>
