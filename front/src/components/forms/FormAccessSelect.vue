<template>
  <Select
    :model-value="modelValue"
    class="fas"
    :options="ACCESS_LEVELS"
    option-label="label"
    option-value="value"
    :disabled="disabled"
    @update:model-value="$emit('update:modelValue', $event)"
  />
</template>

<script setup>
/* Уровень доступа к форме: «Заполнить» (это же и назначение), «Видеть ответы»,
   «Редактировать». Набор — зеркало domain/access.go. */
import Select from 'primevue/select'
import { ACCESS_LEVELS } from '@/utils/formFields.js'

defineProps({
  modelValue: { type: String, default: 'respond' },
  disabled: { type: Boolean, default: false },
})
defineEmits(['update:modelValue'])
</script>

<style scoped>
.fas { width: 180px; }
</style>
