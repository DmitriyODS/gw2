<template>
  <nav ref="navEl" class="bc" aria-label="Навигация по папкам">
    <button
      type="button"
      class="bc-item bc-root"
      :class="{ 'bc-drop': dropIndex === -1 }"
      @click="$emit('navigate', -1)"
      @dragover.prevent="onOver(-1)"
      @dragleave="onLeave(-1)"
      @drop.prevent="onDrop(-1)"
    >
      <span class="material-symbols-outlined">{{ rootIcon }}</span>
      <span class="bc-label">{{ rootLabel }}</span>
    </button>
    <template v-for="(item, i) in items" :key="item.id">
      <span class="material-symbols-outlined bc-sep">chevron_right</span>
      <button
        type="button"
        class="bc-item"
        :class="{ current: i === items.length - 1, 'bc-drop': dropIndex === i }"
        @click="$emit('navigate', i)"
        @dragover.prevent="onOver(i)"
        @dragleave="onLeave(i)"
        @drop.prevent="onDrop(i)"
      >
        <span class="bc-label">{{ item.name }}</span>
      </button>
    </template>
  </nav>
</template>

<script setup>
import { nextTick, ref, watch } from 'vue'

const props = defineProps({
  items: { type: Array, default: () => [] },
  rootLabel: { type: String, default: 'Все заметки' },
  rootIcon: { type: String, default: 'home' },
})
const emit = defineEmits(['navigate', 'drop-item'])

// Строка крошек прокручиваемая (на узких экранах путь длиннее ширины) — при
// смене пути доматываем к концу, чтобы текущая папка всегда была видна.
const navEl = ref(null)
watch(() => props.items.length, () => {
  nextTick(() => { if (navEl.value) navEl.value.scrollLeft = navEl.value.scrollWidth })
})

// Подсветка целевой крошки при перетаскивании (drag-move в родителя).
const dropIndex = ref(null)
function onOver(i) { dropIndex.value = i }
function onLeave(i) { if (dropIndex.value === i) dropIndex.value = null }
function onDrop(i) {
  dropIndex.value = null
  emit('drop-item', i) // i=-1 — корень
}
</script>

<style scoped>
.bc {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-wrap: nowrap;
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}
.bc::-webkit-scrollbar { display: none; }
.bc-item { flex-shrink: 0; }
.bc-sep { flex-shrink: 0; }
.bc-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  max-width: 220px;
  padding: 5px 10px;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text-dim);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}
.bc-item:hover { background: var(--color-surface-high); color: var(--color-text); }
.bc-item.current { color: var(--color-text); cursor: default; }
.bc-item.bc-drop { background: color-mix(in oklch, var(--color-primary) 18%, transparent); color: var(--color-primary); }
.bc-label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bc-item .material-symbols-outlined { font-size: 18px; }
.bc-sep { color: var(--color-text-dim); font-size: 18px; opacity: 0.6; }
.bc-root .material-symbols-outlined { color: var(--color-primary); }
</style>
