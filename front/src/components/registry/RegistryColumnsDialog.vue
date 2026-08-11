<template>
  <AppDialog
    :model-value="modelValue"
    title="Колонки таблицы"
    size="sm"
    :actions="[{ kind: 'cancel', label: 'Готово' }]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
  >
    <AppStack :gap="2">
      <label v-for="f in fields" :key="f.id" class="rc-row">
        <Checkbox
          :model-value="visible.includes(f.id)"
          binary
          @update:model-value="$emit('toggle', f.id)"
        />
        <span>{{ f.label }}</span>
      </label>
    </AppStack>
  </AppDialog>
</template>

<script setup>
/* Какие поля показывать колонками таблицы (в узкой раскладке — строками
   карточки). Настройка личная и хранится у вызывающего — см.
   composables/useRegistryColumns.js. */
import Checkbox from 'primevue/checkbox'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppStack from '@/components/ui/AppStack.vue'

defineProps({
  modelValue: { type: Boolean, default: false },
  /** Все поля реестра. */
  fields: { type: Array, default: () => [] },
  /** Идентификаторы показанных полей. */
  visible: { type: Array, default: () => [] },
})

defineEmits(['update:modelValue', 'toggle'])
</script>

<style scoped>
.rc-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 8px;
  border-radius: var(--radius-md);
  font-size: 14px;
  cursor: pointer;
}

.rc-row:hover { background: var(--color-surface-high); }
</style>
