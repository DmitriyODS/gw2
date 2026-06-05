<template>
  <AppDialog
    :model-value="visible"
    :tone="dangerConfirm ? 'danger' : 'primary'"
    :icon="dangerConfirm ? 'warning' : 'help'"
    :title="header"
    :subtitle="message"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: confirmLabel || 'Подтвердить' },
    ]"
    @update:model-value="(v) => !v && $emit('cancel')"
    @cancel="$emit('cancel')"
    @confirm="$emit('confirm')"
  />
</template>

<script setup>
import AppDialog from './AppDialog.vue'

defineProps({
  visible: { type: Boolean, default: false },
  header: { type: String, default: 'Подтверждение' },
  message: { type: String, default: '' },
  confirmLabel: { type: String, default: 'Подтвердить' },
  dangerConfirm: { type: Boolean, default: false },
})

defineEmits(['confirm', 'cancel'])
</script>
