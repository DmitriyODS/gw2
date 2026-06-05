<template>
  <AppDialog
    :model-value="modelValue"
    tone="danger"
    icon="delete"
    size="sm"
    :title="title"
    :subtitle="text"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: 'Удалить', icon: 'delete' },
    ]"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="confirm"
  >
    <label v-if="canForAll" class="ds-check" :class="{ active: forAll }">
      <input type="checkbox" v-model="forAll" />
      <span class="ds-check-box">
        <span class="material-symbols-outlined">{{ forAll ? 'check_box' : 'check_box_outline_blank' }}</span>
      </span>
      <span class="ds-check-label">Удалить также у {{ otherName || 'собеседника' }}</span>
    </label>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: 'Удалить?' },
  text: { type: String, default: '' },
  canForAll: { type: Boolean, default: true },
  otherName: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'confirm'])

const forAll = ref(false)

watch(() => props.modelValue, (v) => {
  if (v) forAll.value = false
})

function confirm() {
  emit('confirm', { scope: forAll.value ? 'all' : 'me' })
  emit('update:modelValue', false)
}
</script>

<style scoped>
/* Чекбокс «удалить также у собеседника» — единственный кастомный элемент тела,
   шапка/кнопки/маска идут из AppDialog. */
.ds-check {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  cursor: pointer;
  user-select: none;
  width: 100%;
  margin: 8px 0 4px;
  transition: background 0.15s, border-color 0.15s;
}

.ds-check.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.ds-check input { display: none; }

.ds-check-box {
  display: inline-flex;
  align-items: center;
  color: var(--color-primary);
}

.ds-check-box .material-symbols-outlined { font-size: 22px; }

.ds-check-label {
  font-size: 14px;
  font-weight: 500;
  text-align: left;
}
</style>
