<template>
  <AppDialog
    :model-value="modelValue"
    title="Теги заметки" icon="sell" size="sm"
    :busy="saving"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: 'Сохранить' },
    ]"
    @confirm="save" @cancel="close" @update:model-value="(v) => !v && close()"
  >
    <div class="nt">
      <p v-if="!store.tags.length" class="nt-empty">
        Тегов пока нет. Создайте первый — им можно помечать заметки и фильтровать.
      </p>
      <div v-else class="nt-list">
        <button
          v-for="t in store.tags"
          :key="t.id"
          type="button"
          class="nt-chip"
          :class="{ active: selected.has(t.id) }"
          :style="chipStyle(t)"
          @click="toggle(t.id)"
        >
          <span class="material-symbols-outlined">{{ selected.has(t.id) ? 'check' : 'sell' }}</span>
          {{ t.name }}
        </button>
      </div>

      <form class="nt-create" @submit.prevent="quickCreate">
        <ColorSwatchPicker v-model="newColor" aria-label="Цвет тега" />
        <div class="nt-create-row">
          <input v-model="newName" class="nt-input" maxlength="60" placeholder="Новый тег" />
          <button class="btn-glass" type="submit" :disabled="!newName.trim() || creating">
            <span class="material-symbols-outlined">add</span>
          </button>
        </div>
      </form>
    </div>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import ColorSwatchPicker from '@/components/common/ColorSwatchPicker.vue'
import { useNotesStore } from '@/stores/notes.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  noteId: { type: [Number, String, null], default: null },
  tagIds: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue', 'saved'])

const store = useNotesStore()
const notif = useNotificationsStore()
const selected = ref(new Set())
const newName = ref('')
const newColor = ref('')
const creating = ref(false)
const saving = ref(false)

watch(() => props.modelValue, (open) => {
  if (!open) return
  selected.value = new Set(props.tagIds)
  newName.value = ''
  newColor.value = ''
  if (!store.tags.length) store.fetchTags()
})

function chipStyle(t) {
  if (!t.color) return {}
  return {
    '--chip-bg': `var(--tag-${t.color}-surface)`,
    '--chip-bd': `var(--tag-${t.color}-border)`,
    '--chip-fg': `var(--tag-${t.color}-accent)`,
  }
}

function toggle(id) {
  const s = new Set(selected.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  selected.value = s
}

async function quickCreate() {
  const name = newName.value.trim()
  if (!name) return
  creating.value = true
  try {
    const t = await store.createTag(name, newColor.value)
    toggle(t.id)
    newName.value = ''
    newColor.value = ''
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать тег')
  } finally {
    creating.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await store.setNoteTags(props.noteId, [...selected.value])
    emit('saved')
    close()
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить теги')
  } finally {
    saving.value = false
  }
}

function close() { emit('update:modelValue', false) }
</script>

<style scoped>
.nt { display: flex; flex-direction: column; gap: 16px; }
.nt-empty { margin: 0; font-size: 13px; color: var(--color-text-dim); }
.nt-list { display: flex; flex-wrap: wrap; gap: 8px; }
.nt-chip {
  display: inline-flex; align-items: center; gap: 5px;
  height: 34px; padding: 0 12px;
  border: 1px solid var(--chip-bd, var(--color-outline-dim));
  border-radius: var(--radius-full);
  background: var(--chip-bg, var(--color-surface));
  color: var(--chip-fg, var(--color-text));
  font: inherit; font-size: 13px; font-weight: 600; cursor: pointer;
  opacity: 0.6;
}
.nt-chip.active { opacity: 1; box-shadow: 0 0 0 1.5px var(--chip-fg, var(--color-primary)); }
.nt-chip .material-symbols-outlined { font-size: 16px; }
.nt-create { display: flex; flex-direction: column; gap: 10px; padding-top: 8px; border-top: 1px solid var(--color-outline-dim); }
.nt-create-row { display: flex; gap: 8px; }
.nt-input { flex: 1; min-width: 0; height: 40px; padding: 0 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--color-surface); color: var(--color-text); font: inherit; outline: none; }
.nt-input:focus { border-color: var(--color-primary); }
.nt-create-row .btn-glass { width: 44px; padding: 0; display: grid; place-items: center; }
</style>
