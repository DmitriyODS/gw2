<template>
  <AppDialog
    :model-value="modelValue"
    title="Переместить" icon="drive_file_move" size="sm"
    :busy="moving"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: 'Переместить' },
    ]"
    @confirm="confirm" @cancel="close" @update:model-value="(v) => !v && close()"
  >
    <div class="mv">
      <p class="mv-hint">Выберите папку назначения:</p>
      <button
        type="button"
        class="mv-root"
        :class="{ active: target === null }"
        @click="target = null"
      >
        <span class="material-symbols-outlined">home</span>
        <span>В корень (без папки)</span>
        <span v-if="target === null" class="material-symbols-outlined mv-check">check</span>
      </button>
      <div class="mv-tree">
        <TreeView
          :nodes="nodes"
          :selected-id="target"
          :expanded="expanded"
          @select="onSelect"
          @toggle="onToggle"
        />
      </div>
      <p v-if="!nodes.length" class="mv-empty">Других папок нет — доступен только корень.</p>
    </div>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import TreeView from '@/components/common/TreeView.vue'
import { useNotesStore } from '@/stores/notes.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  itemType: { type: String, default: 'note' }, // note | folder
  itemId: { type: [Number, null], default: null },
})
const emit = defineEmits(['update:modelValue', 'moved'])

const store = useNotesStore()
const auth = useAuthStore()
const notif = useNotificationsStore()
const target = ref(null)
const moving = ref(false)
const expanded = ref(new Set())

// При переносе папки исключаем её саму и всё поддерево (иначе цикл).
const excluded = computed(() => {
  const set = new Set()
  if (props.itemType !== 'folder' || props.itemId == null) return set
  const collect = (id) => {
    set.add(id)
    store.childrenOf(id).forEach((c) => collect(c.id))
  }
  collect(props.itemId)
  return set
})

// Цель переноса — только МОИ папки (в чужие размещённые класть нельзя).
function buildTree(parentId) {
  return store.childrenOf(parentId)
    .filter((f) => !excluded.value.has(f.id) && (f.owner_id == null || f.owner_id === auth.userId))
    .sort((a, b) => (a.position - b.position) || a.name.localeCompare(b.name))
    .map((f) => ({ ...f, owner_is_me: true, children: buildTree(f.id) }))
}
const nodes = computed(() => buildTree(null))

watch(() => props.modelValue, (open) => {
  if (!open) return
  target.value = null
  // Раскрыть все ветки — удобнее выбирать.
  expanded.value = new Set(store.folders.map((f) => f.id))
})

function onSelect(node) { target.value = node.id }
function onToggle(id) {
  const s = new Set(expanded.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  expanded.value = s
}

async function confirm() {
  moving.value = true
  try {
    if (props.itemType === 'folder') await store.moveFolder(props.itemId, target.value)
    else await store.moveNote(props.itemId, target.value)
    emit('moved')
    close()
  } catch (e) {
    notif.error(e?.message || 'Не удалось переместить')
  } finally {
    moving.value = false
  }
}

function close() { emit('update:modelValue', false) }
</script>

<style scoped>
.mv { display: flex; flex-direction: column; gap: 10px; }
.mv-hint { margin: 0; font-size: 13px; color: var(--color-text-dim); }
.mv-root {
  display: flex; align-items: center; gap: 8px;
  padding: 10px 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md);
  background: var(--color-surface); color: var(--color-text); font: inherit; font-weight: 600; cursor: pointer;
}
.mv-root.active { border-color: var(--color-primary); background: color-mix(in oklch, var(--color-primary) 10%, transparent); color: var(--color-primary); }
.mv-root .material-symbols-outlined { font-size: 19px; }
.mv-check { margin-left: auto; }
.mv-tree { max-height: 40vh; overflow-y: auto; }
.mv-empty { margin: 0; font-size: 13px; color: var(--color-text-dim); }
</style>
