<template>
  <Select
    :model-value="modelValue"
    :options="options"
    option-label="label"
    option-value="value"
    class="acc-select"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <template #option="{ option }">
      <div class="acc-option">
        <span class="acc-option-label">{{ option.label }}</span>
        <span class="acc-option-hint">{{ option.hint }}</span>
      </div>
    </template>
  </Select>
</template>

<script setup>
/* Уровень доступа — один и тот же выбор у всех трёх способов поделиться
   (ссылка, человек, компания), поэтому он вынесен отдельным компонентом.
   Значения — зеркало domain/access.go. */
import Select from 'primevue/select'

defineProps({
  modelValue: { type: String, default: 'view' },
})
defineEmits(['update:modelValue'])

const options = [
  { value: 'view', label: 'Просмотр', hint: 'Смотреть, выгружать и печатать QR-коды' },
  { value: 'edit', label: 'Редактирование', hint: 'Плюс вести записи' },
  { value: 'admin', label: 'Администрирование', hint: 'Плюс менять структуру реестра' },
]
</script>

<style scoped>
.acc-select { min-width: 190px; }
.acc-option { display: flex; flex-direction: column; gap: 2px; }
.acc-option-label { font-size: 14px; }
.acc-option-hint { font-size: 12px; color: var(--color-text-dim); }
</style>
