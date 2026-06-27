<template>
  <AppDialog
    :model-value="modelValue"
    :title="mode === 'export' ? 'Что включить в копию' : 'Что восстановить'"
    :icon="mode === 'export' ? 'download' : 'restore'"
    :tone="mode === 'import' ? 'danger' : 'primary'"
    size="md"
    :busy="busy"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: mode === 'export' ? 'Скачать' : 'Восстановить', disabled: !selected.length },
    ]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
    @confirm="onConfirm"
  >
    <div class="bsd">
      <p class="bsd-note">
        <template v-if="mode === 'export'">Выберите разделы для выгрузки. По умолчанию — все.</template>
        <template v-else>Выбранные разделы будут <b>полностью заменены</b> данными из архива. Восстанавливаются только разделы, которые есть в файле.</template>
      </p>
      <div class="bsd-bulk">
        <button type="button" class="bsd-link" @click="selectAll">Выбрать всё</button>
        <button type="button" class="bsd-link" @click="clearAll">Снять всё</button>
      </div>
      <label v-for="s in sections" :key="s.key" class="bsd-row">
        <Checkbox v-model="selected" :value="s.key" />
        <span class="bsd-main">
          <span class="bsd-title">{{ s.label }}</span>
          <span class="bsd-desc">{{ s.desc }}</span>
        </span>
      </label>
    </div>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import Checkbox from 'primevue/checkbox'
import AppDialog from '@/components/common/AppDialog.vue'
import { BACKUP_SECTIONS, ALL_SECTION_KEYS } from '@/utils/backupSections.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  mode: { type: String, default: 'export' }, // 'export' | 'import'
  busy: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'confirm'])

const sections = BACKUP_SECTIONS
const selected = ref([...ALL_SECTION_KEYS])

// При каждом открытии — сбрасываем выбор на «все».
watch(() => props.modelValue, (open) => { if (open) selected.value = [...ALL_SECTION_KEYS] })

function selectAll() { selected.value = [...ALL_SECTION_KEYS] }
function clearAll() { selected.value = [] }
function onConfirm() {
  if (!selected.value.length) return
  emit('confirm', [...selected.value])
}
</script>

<style scoped>
.bsd { display: flex; flex-direction: column; gap: 10px; }
.bsd-note { margin: 0; font-size: 14px; color: var(--color-text-dim); }
.bsd-bulk { display: flex; gap: 12px; }
.bsd-link { border: none; background: none; cursor: pointer; padding: 0; color: var(--color-primary); font-weight: 600; font-size: 13px; }
.bsd-row { display: flex; align-items: flex-start; gap: 12px; padding: 10px 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); cursor: pointer; }
.bsd-row:hover { background: var(--color-surface-high); }
.bsd-main { display: flex; flex-direction: column; min-width: 0; }
.bsd-title { font-size: 14px; font-weight: 600; color: var(--color-text); }
.bsd-desc { font-size: 12px; color: var(--color-text-dim); }
</style>
