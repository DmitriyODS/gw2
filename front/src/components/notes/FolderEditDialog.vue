<template>
  <AppDialog
    :model-value="modelValue"
    :title="folder ? 'Переименовать папку' : 'Новая папка'"
    :icon="folder ? 'drive_file_rename_outline' : 'create_new_folder'"
    size="sm"
    :busy="saving"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: folder ? 'Сохранить' : 'Создать' },
    ]"
    @confirm="save" @cancel="close" @update:model-value="(v) => !v && close()"
  >
    <div class="fe">
      <input
        ref="nameEl"
        v-model="name"
        class="fe-input"
        maxlength="200"
        placeholder="Название папки"
        @keydown.enter="save"
      />
      <div class="fe-color">
        <span class="fe-label">Цвет</span>
        <ColorSwatchPicker v-model="color" aria-label="Цвет папки" />
      </div>
    </div>
  </AppDialog>
</template>

<script setup>
import { nextTick, ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import ColorSwatchPicker from '@/components/common/ColorSwatchPicker.vue'
import { useNotesStore } from '@/stores/notes.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  folder: { type: Object, default: null }, // null — создание
  parentId: { type: [Number, null], default: null },
})
const emit = defineEmits(['update:modelValue', 'saved'])

const store = useNotesStore()
const notif = useNotificationsStore()
const name = ref('')
const color = ref('')
const saving = ref(false)
const nameEl = ref(null)

watch(() => props.modelValue, (open) => {
  if (!open) return
  name.value = props.folder?.name || ''
  color.value = props.folder?.color || ''
  nextTick(() => nameEl.value?.focus())
})

async function save() {
  const n = name.value.trim()
  if (!n) return
  saving.value = true
  try {
    const f = props.folder
      ? await store.renameFolder(props.folder.id, n, color.value)
      : await store.createFolder(n, props.parentId, color.value)
    emit('saved', f)
    close()
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить папку')
  } finally {
    saving.value = false
  }
}

function close() { emit('update:modelValue', false) }
</script>

<style scoped>
.fe { display: flex; flex-direction: column; gap: 16px; }
.fe-input { height: 44px; padding: 0 14px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--color-surface); color: var(--color-text); font: inherit; font-size: 15px; outline: none; }
.fe-input:focus { border-color: var(--color-primary); }
.fe-color { display: flex; flex-direction: column; gap: 8px; }
.fe-label { font-size: 12.5px; font-weight: 600; color: var(--color-text-dim); }
</style>
