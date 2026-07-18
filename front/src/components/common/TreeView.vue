<template>
  <ul class="tv" :class="{ 'tv-root': level === 0 }">
    <li v-for="node in nodes" :key="node.id" class="tv-li">
      <div
        class="tv-row"
        :class="{ active: node.id === selectedId, shared: node.shared, 'tv-drop': dropId === node.id }"
        :style="{ paddingLeft: `${8 + level * 14}px` }"
        role="treeitem"
        :aria-selected="node.id === selectedId"
        :aria-expanded="hasChildren(node) ? isOpen(node.id) : undefined"
        tabindex="0"
        :draggable="!!node.owner_is_me"
        @click="$emit('select', node)"
        @keydown.enter="$emit('select', node)"
        @contextmenu.prevent="$emit('context', { node, event: $event })"
        @dragstart="onDragStart(node, $event)"
        @dragover.prevent="onOver(node)"
        @dragleave="onLeave(node)"
        @drop.prevent="onDrop(node, $event)"
      >
        <button
          v-if="hasChildren(node)"
          type="button"
          class="tv-toggle"
          :aria-label="isOpen(node.id) ? 'Свернуть' : 'Развернуть'"
          @click.stop="toggle(node.id)"
        >
          <span class="material-symbols-outlined" :class="{ open: isOpen(node.id) }">chevron_right</span>
        </button>
        <span v-else class="tv-toggle-spacer" />

        <span class="tv-icon material-symbols-outlined" :style="folderColor(node)">
          {{ node.shared ? 'folder_shared' : (isOpen(node.id) && hasChildren(node) ? 'folder_open' : 'folder') }}
        </span>
        <span class="tv-name">{{ node.name }}</span>
        <span v-if="node.shared_by_me && !node.shared" class="tv-badge material-symbols-outlined" title="Вы поделились">share</span>
        <span v-if="node.notes_count" class="tv-count">{{ node.notes_count }}</span>
      </div>

      <TreeView
        v-if="hasChildren(node) && isOpen(node.id)"
        :nodes="node.children"
        :selected-id="selectedId"
        :expanded="expanded"
        :level="level + 1"
        @select="$emit('select', $event)"
        @toggle="$emit('toggle', $event)"
        @context="$emit('context', $event)"
        @node-dragstart="$emit('node-dragstart', $event)"
        @node-drop="$emit('node-drop', $event)"
      />
    </li>
  </ul>
</template>

<script setup>
import { ref } from 'vue'

defineOptions({ name: 'TreeView' })

const props = defineProps({
  nodes: { type: Array, default: () => [] },
  selectedId: { type: [Number, null], default: null },
  expanded: { type: Object, default: () => new Set() }, // Set раскрытых id
  level: { type: Number, default: 0 },
})
const emit = defineEmits(['select', 'toggle', 'context', 'node-dragstart', 'node-drop'])

const dropId = ref(null)

const hasChildren = (n) => Array.isArray(n.children) && n.children.length > 0
const isOpen = (id) => props.expanded.has(id)
function toggle(id) { emit('toggle', id) }

function folderColor(node) {
  if (!node.color) return {}
  return { color: `var(--tag-${node.color}-accent)` }
}

function onDragStart(node, e) {
  if (!node.owner_is_me) { e.preventDefault(); return }
  emit('node-dragstart', node)
  e.dataTransfer.effectAllowed = 'move'
  try { e.dataTransfer.setData('text/plain', `folder:${node.id}`) } catch { /* Safari */ }
}
function onOver(node) { if (node.owner_is_me) dropId.value = node.id }
function onLeave(node) { if (dropId.value === node.id) dropId.value = null }
function onDrop(node) {
  dropId.value = null
  if (node.owner_is_me) emit('node-drop', node)
}
</script>

<style scoped>
.tv { list-style: none; margin: 0; padding: 0; }
.tv-row {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 7px 8px;
  border-radius: var(--radius-md);
  cursor: pointer;
  color: var(--color-text);
  user-select: none;
}
.tv-row:hover { background: var(--color-surface-high); }
.tv-row.active { background: color-mix(in oklch, var(--color-primary) 14%, transparent); color: var(--color-primary); }
.tv-row.shared { color: var(--color-tertiary); }
.tv-row.active.shared { background: color-mix(in oklch, var(--color-tertiary) 14%, transparent); }
.tv-row.tv-drop { background: color-mix(in oklch, var(--color-primary) 22%, transparent); box-shadow: inset 0 0 0 1.5px var(--color-primary); }
.tv-row:focus-visible { outline: 2px solid var(--color-primary); outline-offset: -2px; }

.tv-toggle {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  display: grid;
  place-items: center;
  border: none;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  padding: 0;
}
.tv-toggle .material-symbols-outlined { font-size: 18px; transition: transform 0.15s ease; }
.tv-toggle .material-symbols-outlined.open { transform: rotate(90deg); }
.tv-toggle-spacer { width: 20px; flex-shrink: 0; }

.tv-icon { font-size: 19px; flex-shrink: 0; color: var(--color-text-dim); }
.tv-row.active .tv-icon, .tv-row.shared .tv-icon { color: inherit; }
.tv-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 600;
}
.tv-badge { font-size: 14px; color: var(--color-tertiary); flex-shrink: 0; }
.tv-count {
  flex-shrink: 0;
  min-width: 20px;
  padding: 0 6px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  font-size: 11px;
  font-weight: 700;
  text-align: center;
}
.tv-row.active .tv-count { background: color-mix(in oklch, var(--color-primary) 22%, transparent); color: var(--color-primary); }
</style>
