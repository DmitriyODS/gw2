<template>
  <AppDialog
    :model-value="modelValue"
    title="О разделе"
    size="sm"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
  >
    <AppStack :gap="10">
      <AppRow title="Раздел" :hint="name" inline />
      <AppRow title="Версия" :hint="version" inline />
      <AppRow v-if="updatedAt" title="Обновлён" :hint="updatedAt" inline />
    </AppStack>
  </AppDialog>
</template>

<script setup>
/* Карточка «О разделе»: своя версия у КАЖДОГО раздела, а не у приложения.
   Разделы переписываются поодиночке, и по этой цифре видно, какое поколение
   раздела перед человеком; к выпуску платформы (data/changelog.json) она
   отношения не имеет.

   Сведения раздел объявляет в реестре приложений (desktop/apps.js, поле
   `about`) — оттуда их берёт и заголовок окна, и сам раздел. */
import { computed } from 'vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  name: { type: String, default: '' },
  version: { type: String, default: '' },
  /** Дата обновления раздела в формате ГГГГ-ММ-ДД. */
  date: { type: String, default: '' },
})
defineEmits(['update:modelValue'])

const updatedAt = computed(() => {
  if (!props.date) return ''
  const d = new Date(props.date)
  return isNaN(d) ? '' : d.toLocaleDateString('ru-RU', {
    day: '2-digit', month: 'long', year: 'numeric',
  })
})
</script>
