<template>
  <AppDialog
    :model-value="modelValue"
    title="Папки с чатами"
    subtitle="Раскладывайте чаты по папкам — как в Telegram"
    icon="folder_open"
    tone="tertiary"
    size="md"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="fm-body">
      <ul v-if="list.length" class="fm-list">
        <li
          v-for="(f, idx) in list"
          :key="f.id"
          class="fm-item"
          draggable="true"
          :class="{ dragging: dragIndex === idx, over: overIndex === idx }"
          @dragstart="onDragStart(idx, $event)"
          @dragover.prevent="onDragOver(idx)"
          @drop.prevent="onDrop(idx)"
          @dragend="onDragEnd"
        >
          <span class="fm-handle material-symbols-outlined" title="Перетащите для порядка">drag_indicator</span>
          <span class="fm-emoji">
            <EmojiGlyph v-if="f.emoji" :char="f.emoji" class="fm-emoji-glyph" />
            <span v-else class="material-symbols-outlined">folder</span>
          </span>
          <div class="fm-info">
            <div class="fm-title">{{ f.title }}</div>
            <div class="fm-sub">{{ folderSummary(f) }}</div>
          </div>
          <button class="fm-act" title="Изменить" @click="edit(f)">
            <span class="material-symbols-outlined">edit</span>
          </button>
          <button class="fm-act danger" title="Удалить" @click="askDelete(f)">
            <span class="material-symbols-outlined">delete</span>
          </button>
        </li>
      </ul>
      <EmptyState
        v-else
        icon="create_new_folder"
        title="Папок пока нет"
        subtitle="Создайте первую — и группируйте чаты так, как удобно."
      />

      <button class="fm-new btn-glass" @click="create">
        <span class="material-symbols-outlined">add</span>
        Новая папка
      </button>
    </div>
  </AppDialog>

  <FolderEditDialog v-model="editOpen" :folder="editing" />

  <AppDialog
    v-model="deleteOpen"
    tone="danger"
    icon="delete"
    title="Удалить папку?"
    :subtitle="deleting ? `Папка «${deleting.title}» будет удалена. Чаты и переписка останутся на месте.` : ''"
    size="sm"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: 'Удалить', tone: 'danger', icon: 'delete' },
    ]"
    @confirm="confirmDelete"
  />
</template>

<script setup>
import { ref, computed } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import FolderEditDialog from './FolderEditDialog.vue'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotificationsStore } from '@/stores/notifications.js'

defineProps({ modelValue: { type: Boolean, default: false } })
defineEmits(['update:modelValue'])

const messenger = useMessengerStore()
const list = computed(() => messenger.folders)

const editOpen = ref(false)
const editing = ref(null)
const deleteOpen = ref(false)
const deleting = ref(null)

function create() { editing.value = null; editOpen.value = true }
function edit(f) { editing.value = f; editOpen.value = true }

function askDelete(f) { deleting.value = f; deleteOpen.value = true }
async function confirmDelete() {
  if (!deleting.value) return
  try {
    await messenger.deleteFolderAction(deleting.value.id)
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось удалить папку')
  }
}

function folderSummary(f) {
  const parts = []
  const n = f.conversation_ids?.length || 0
  if (n) parts.push(`${n} чат${plural(n)}`)
  if (f.include_personal) parts.push('личные')
  if (f.include_groups) parts.push('группы')
  if (f.include_unread) parts.push('непрочитанные')
  return parts.length ? parts.join(' · ') : 'Пустая папка'
}
function plural(n) {
  const d = n % 10, dd = n % 100
  if (d === 1 && dd !== 11) return ''
  if (d >= 2 && d <= 4 && (dd < 10 || dd >= 20)) return 'а'
  return 'ов'
}

// ── Drag-and-drop порядка ──
const dragIndex = ref(-1)
const overIndex = ref(-1)
function onDragStart(idx, e) {
  dragIndex.value = idx
  e.dataTransfer.effectAllowed = 'move'
}
function onDragOver(idx) { overIndex.value = idx }
function onDrop(idx) {
  const from = dragIndex.value
  if (from === -1 || from === idx) return
  const ids = list.value.map(f => f.id)
  const [moved] = ids.splice(from, 1)
  ids.splice(idx, 0, moved)
  messenger.reorderFoldersAction(ids).catch(() => {})
}
function onDragEnd() { dragIndex.value = -1; overIndex.value = -1 }
</script>

<style scoped>
.fm-body { display: flex; flex-direction: column; gap: 12px; }

.fm-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 6px; }

.fm-item {
  display: flex; align-items: center; gap: 10px;
  padding: 8px 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
}
.fm-item.dragging { opacity: 0.5; }
.fm-item.over { border-color: var(--color-primary); }

.fm-handle { color: var(--color-text-dim); cursor: grab; font-size: 20px; flex-shrink: 0; }

.fm-emoji {
  width: 36px; height: 36px; border-radius: var(--radius-sm);
  flex-shrink: 0; display: grid; place-items: center;
  background: var(--color-tertiary-container); color: var(--color-on-tertiary-container);
}
.fm-emoji-glyph { font-size: 20px; line-height: 1; }
.fm-emoji .material-symbols-outlined { font-size: 20px; font-variation-settings: 'FILL' 1; }

.fm-info { flex: 1; min-width: 0; }
.fm-title { font-size: 14px; font-weight: 600; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.fm-sub { font-size: 12px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.fm-act {
  width: 34px; height: 34px; min-height: 0;
  flex-shrink: 0;
  border: none; border-radius: 50%;
  background: transparent; color: var(--color-text-dim);
  cursor: pointer; display: grid; place-items: center;
  transition: background 0.15s, color 0.15s;
}
.fm-act:hover { background: var(--color-surface-high); color: var(--color-text); }
.fm-act.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.fm-act .material-symbols-outlined { font-size: 18px; }

.fm-new {
  align-self: flex-start;
  display: inline-flex; align-items: center; gap: 8px;
}
.fm-new .material-symbols-outlined { font-size: 20px; }
</style>
