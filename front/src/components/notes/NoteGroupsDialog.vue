<template>
  <AppDialog
    :model-value="modelValue"
    title="Группы заметки" icon="folder" size="sm"
    :busy="busy"
    :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Сохранить' }]"
    @confirm="save" @cancel="close" @update:model-value="(v) => !v && close()"
  >
    <p class="ng-note">
      Заметка может входить в несколько групп, одну или ни одной — в списке она
      видна в каждой выбранной группе и всегда во «Все».
    </p>
    <div v-if="groups.length" class="ng-chips">
      <button
        v-for="g in groups"
        :key="g.id"
        class="ng-chip"
        :class="{ active: selected.has(g.id) }"
        @click="toggle(g.id)"
      >
        <span class="material-symbols-outlined">{{ selected.has(g.id) ? 'check' : 'folder' }}</span>
        {{ g.name }}
      </button>
    </div>
    <p v-else class="ng-empty">Групп пока нет — создайте их в списке заметок.</p>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { useNotesStore } from '@/stores/notes.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { setNoteGroups } from '@/api/notes.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  noteId: { type: [Number, String, null], default: null },
  groupIds: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue', 'saved'])

const store = useNotesStore()
const notif = useNotificationsStore()
const groups = ref([])
const selected = ref(new Set())
const busy = ref(false)

watch(() => props.modelValue, async (open) => {
  if (!open) return
  selected.value = new Set(props.groupIds)
  if (!store.groups.length) await store.fetchGroups({ silent: true })
  groups.value = store.groups
})

function toggle(id) {
  const next = new Set(selected.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selected.value = next
}

async function save() {
  busy.value = true
  try {
    const n = await setNoteGroups(props.noteId, [...selected.value])
    emit('saved', n)
    close()
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить группы')
  } finally {
    busy.value = false
  }
}

function close() { emit('update:modelValue', false) }
</script>

<style scoped>
.ng-note { margin: 0 0 12px; font-size: 13px; color: var(--color-text-dim); line-height: 1.5; }
.ng-empty { margin: 0; font-size: 13px; color: var(--color-text-dim); }
.ng-chips { display: flex; flex-wrap: wrap; gap: 8px; }
.ng-chip {
  display: inline-flex; align-items: center; gap: 6px; height: 34px; padding: 0 14px;
  border: 1px solid var(--color-outline-variant); border-radius: var(--radius-full);
  background: var(--color-surface); color: var(--color-text);
  font-size: 13px; font-weight: 600; cursor: pointer;
}
.ng-chip .material-symbols-outlined { font-size: 17px; color: var(--color-text-dim); }
.ng-chip:hover { background: color-mix(in oklch, var(--color-primary) 8%, var(--color-surface)); }
.ng-chip.active {
  background: color-mix(in oklch, var(--color-primary) 14%, var(--color-surface));
  border-color: color-mix(in oklch, var(--color-primary) 30%, transparent);
  color: var(--color-primary);
}
.ng-chip.active .material-symbols-outlined { color: var(--color-primary); }
</style>
