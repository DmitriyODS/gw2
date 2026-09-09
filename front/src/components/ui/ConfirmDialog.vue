<template>
  <!-- Подтверждение — короткий вопрос в одну-две строки, поэтому размер `sm`:
       на широком экране `md` растягивался вдвое, и кнопки «Отмена»/«Удалить»
       разъезжались по разным краям полосы. -->
  <AppDialog
    :model-value="visible"
    size="sm"
    :tone="dangerConfirm ? 'danger' : 'primary'"
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
import AppDialog from '@/components/ui/AppDialog.vue'

defineProps({
  visible: { type: Boolean, default: false },
  header: { type: String, default: 'Подтверждение' },
  message: { type: String, default: '' },
  confirmLabel: { type: String, default: 'Подтвердить' },
  dangerConfirm: { type: Boolean, default: false },
})

defineEmits(['confirm', 'cancel'])
</script>
