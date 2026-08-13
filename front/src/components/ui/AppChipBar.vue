<template>
  <div class="chipbar" role="group" :aria-label="ariaLabel">
    <AppChip
      v-if="allLabel"
      :label="allLabel"
      :count="allCount"
      interactive
      :size="size"
      :tone="modelValue ? 'neutral' : tone"
      :selected="!modelValue"
      @click="pick('')"
    />
    <AppChip
      v-for="opt in normalized"
      :key="opt.value"
      :label="opt.label"
      :icon="opt.icon"
      :count="opt.count"
      interactive
      :size="size"
      :tone="modelValue === opt.value ? tone : 'neutral'"
      :selected="modelValue === opt.value"
      @click="pick(opt.value)"
    />
  </div>
</template>

<script setup>
/* Ряд чипов-фильтров: теги реестра, разделы ленты, метки заметок.

   Выбор ОДИН: повторный клик по активному чипу снимает фильтр — это и есть
   возврат к «Все», поэтому отдельная кнопка сброса не нужна. Значения —
   строки: чипы приходят из пользовательских данных (варианты поля, теги),
   где id может и не быть. */
import { computed } from 'vue'
import AppChip from './AppChip.vue'

const props = defineProps({
  /** Строки либо { value, label, count?, icon? }. */
  options: { type: Array, default: () => [] },
  /** Выбранное значение; '' — фильтр снят. */
  modelValue: { type: String, default: '' },
  /** Подпись чипа «всё»; пустая строка убирает его. */
  allLabel: { type: String, default: 'Все' },
  allCount: { type: [Number, String], default: null },
  size: { type: String, default: 'md', validator: (v) => ['sm', 'md'].includes(v) },
  tone: { type: String, default: 'primary' },
  ariaLabel: { type: String, default: 'Фильтр' },
})
const emit = defineEmits(['update:modelValue'])

const normalized = computed(() => props.options
  .map((o) => (typeof o === 'string' ? { value: o, label: o } : o))
  .filter((o) => o && o.value !== undefined && o.value !== ''))

function pick(value) {
  emit('update:modelValue', props.modelValue === value ? '' : value)
}
</script>

<style scoped>
.chipbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
</style>
