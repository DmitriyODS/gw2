<template>
  <Teleport to="body">
    <div v-if="visible" class="sort-backdrop" @click="$emit('close')" />
    <div class="sort-sheet" :class="{ 'sort-sheet--open': visible }">
      <div class="sort-handle" />
      <h4 class="sort-sheet-title">Сортировка</h4>
      <div class="sort-options">
        <button
          v-for="s in sorts"
          :key="s.value"
          class="sort-btn"
          :class="{ active: tasksStore.filters.sort === s.value }"
          @click="select(s.value)"
        >
          <span class="material-symbols-outlined">{{ s.icon }}</span>
          <span class="sort-btn-label">{{ s.label }}</span>
          <span v-if="tasksStore.filters.sort === s.value" class="material-symbols-outlined sort-check">check</span>
        </button>
      </div>
      <button class="reset-btn" @click="resetAll">
        <span class="material-symbols-outlined">restart_alt</span>
        Сбросить сортировку
      </button>
    </div>
  </Teleport>
</template>

<script setup>
import { useTasksStore } from '@/stores/tasks.js'
import { TASK_SORTS } from '@/components/tasks/taskSorts.js'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['close'])

const tasksStore = useTasksStore()

const sorts = TASK_SORTS

function select(value) {
  tasksStore.setFilter('sort', value)
  emit('close')
}

function resetAll() {
  tasksStore.resetFilters()
  emit('close')
}
</script>

<style scoped>
.sort-backdrop {
  position: fixed;
  inset: 0;
  background: var(--color-scrim);
  z-index: 399;
}

.sort-sheet {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border-top: 1px solid var(--color-outline-dim);
  border-radius: 24px 24px 0 0;
  z-index: 400;
  padding: 12px 16px calc(16px + env(safe-area-inset-bottom, 0px));
  display: flex;
  flex-direction: column;
  gap: 6px;
  transform: translateY(105%);
  transition: transform 0.32s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 -8px 24px color-mix(in oklch, var(--color-scrim) 60%, transparent);
}

.sort-sheet--open {
  transform: translateY(0);
}

.sort-handle {
  width: 36px;
  height: 4px;
  border-radius: 2px;
  background: var(--color-outline-dim);
  margin: 0 auto 12px;
  flex-shrink: 0;
}

.sort-sheet-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text-dim);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin: 0 0 6px;
  flex-shrink: 0;
}

.sort-options {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sort-btn {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 14px;
  border-radius: var(--radius-lg);
  border: 1px solid transparent;
  background: transparent;
  color: var(--color-text);
  font-size: 15px;
  cursor: pointer;
  text-align: left;
  min-height: 52px;
  transition: background 0.12s, color 0.12s;
}

.sort-btn:active {
  background: color-mix(in oklch, var(--color-primary) 12%, transparent);
}

.sort-btn:hover {
  background: var(--color-surface-high);
}

.sort-btn.active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  border-color: color-mix(in oklch, var(--color-primary) 30%, transparent);
  font-weight: 600;
}

.sort-btn .material-symbols-outlined {
  font-size: 20px;
  flex-shrink: 0;
}

.sort-btn-label {
  flex: 1;
}

.sort-check {
  font-size: 18px !important;
  color: var(--color-primary);
}

.reset-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 8px;
  padding: 13px 14px;
  border-radius: var(--radius-full);
  border: 1px solid var(--color-outline-dim);
  background: transparent;
  color: var(--color-text);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.12s, color 0.12s, border-color 0.12s;
}

.reset-btn:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  border-color: color-mix(in oklch, var(--color-error) 40%, var(--color-outline-dim));
}

.reset-btn .material-symbols-outlined {
  font-size: 18px;
}
</style>
