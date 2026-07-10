<template>
  <AppDialog
    :model-value="modelValue"
    title="Теги задач"
    subtitle="Общий справочник компании — теги видят все сотрудники"
    icon="sell"
    tone="primary"
    size="sm"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <!-- Создание -->
    <div class="tagm-create">
      <input
        v-model="newName"
        class="tagm-input"
        type="text"
        maxlength="64"
        placeholder="Новый тег…"
        @keydown.enter.prevent="create"
      />
      <div class="tagm-colors">
        <button
          v-for="c in TASK_COLORS"
          :key="c.id"
          class="tagm-color"
          :class="{ active: newColor === c.id }"
          :style="{ background: `var(--tag-${c.id}-accent)` }"
          :title="c.label"
          :aria-label="c.label"
          type="button"
          @click="newColor = c.id"
        />
      </div>
      <button class="btn-glass tagm-add" :disabled="busy || !newName.trim()" @click="create">
        <span class="material-symbols-outlined">add</span>
        Создать
      </button>
    </div>

    <!-- Список -->
    <EmptyState
      v-if="!tags.length"
      icon="sell" size="sm" tone="soft"
      title="Тегов пока нет"
      subtitle="Создайте первый — например, «Срочно» или «Багфикс»."
    />
    <ul v-else class="tagm-list">
      <li v-for="t in tags" :key="t.id" class="tagm-row" :style="rowStyle(t)">
        <span class="stage-chip-dot tagm-dot" :style="{ background: `var(--tag-${t.color}-accent)` }" />
        <template v-if="editingId === t.id">
          <input
            v-model="editName"
            class="tagm-input tagm-input--edit"
            type="text"
            maxlength="64"
            @keydown.enter.prevent="saveEdit(t)"
            @keydown.esc="editingId = null"
          />
          <button class="tagm-icon" title="Сохранить" @click="saveEdit(t)">
            <span class="material-symbols-outlined">check</span>
          </button>
        </template>
        <template v-else>
          <span class="tagm-name">{{ t.name }}</span>
          <button class="tagm-icon" title="Переименовать" @click="startEdit(t)">
            <span class="material-symbols-outlined">edit</span>
          </button>
          <button class="tagm-icon danger" title="Удалить" @click="askDelete(t)">
            <span class="material-symbols-outlined">delete</span>
          </button>
        </template>
      </li>
    </ul>

    <ConfirmDialog
      :visible="!!deleting"
      header="Удалить тег?"
      :message="`Тег «${deleting?.name}» будет снят со всех задач компании.`"
      confirm-label="Удалить"
      :danger-confirm="true"
      @confirm="doDelete"
      @cancel="deleting = null"
    />
  </AppDialog>
</template>

<script setup>
import { ref } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { createTag, updateTag, deleteTag } from '@/api/tasks.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { TASK_COLORS } from '@/utils/taskColors.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  tags: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue', 'changed'])

const notify = useNotificationsStore()
const busy = ref(false)
const newName = ref('')
const newColor = ref('blue')
const editingId = ref(null)
const editName = ref('')
const deleting = ref(null)

function rowStyle(t) {
  return { background: `var(--tag-${t.color}-surface)` }
}

async function run(action, successMsg) {
  busy.value = true
  try {
    await action()
    if (successMsg) notify.success(successMsg)
    emit('changed')
  } catch (e) {
    notify.error(e?.message || 'Не удалось сохранить тег')
  } finally {
    busy.value = false
  }
}

const create = () => {
  const name = newName.value.trim()
  if (!name) return
  run(async () => {
    await createTag(name, newColor.value)
    newName.value = ''
  }, 'Тег создан')
}

function startEdit(t) {
  editingId.value = t.id
  editName.value = t.name
}

const saveEdit = (t) => {
  const name = editName.value.trim()
  if (!name || name === t.name) {
    editingId.value = null
    return
  }
  run(async () => {
    await updateTag(t.id, { name })
    editingId.value = null
  })
}

function askDelete(t) {
  deleting.value = t
}

const doDelete = () => {
  const t = deleting.value
  deleting.value = null
  if (t) run(() => deleteTag(t.id), 'Тег удалён')
}
</script>

<style scoped>
.tagm-create {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding-bottom: 14px;
  margin-bottom: 14px;
  border-bottom: 1px solid var(--color-outline-dim);
}

.tagm-input {
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit;
  font-size: 13.5px;
  padding: 9px 12px;
}
.tagm-input:focus { outline: none; border-color: var(--color-primary); }

.tagm-colors { display: flex; gap: 8px; flex-wrap: wrap; }
.tagm-color {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  transition: transform 0.12s, border-color 0.12s;
}
.tagm-color.active {
  border-color: var(--color-text);
  transform: scale(1.12);
}

.tagm-add { align-self: flex-start; }

.tagm-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }
.tagm-row {
  display: flex;
  align-items: center;
  gap: 8px;
  border-radius: var(--radius-md);
  padding: 8px 12px;
}
.tagm-dot { width: 9px; height: 9px; border-radius: 50%; flex-shrink: 0; }
.tagm-name {
  flex: 1;
  min-width: 0;
  font-size: 13.5px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tagm-input--edit { flex: 1; min-width: 0; padding: 5px 9px; }

.tagm-icon {
  border: none;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: inline-flex;
  padding: 4px;
  border-radius: var(--radius-full);
}
.tagm-icon:hover { color: var(--color-text); }
.tagm-icon.danger:hover { color: var(--color-error); }
.tagm-icon .material-symbols-outlined { font-size: 17px; }
</style>
