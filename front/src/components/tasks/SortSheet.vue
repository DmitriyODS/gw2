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

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['close'])

const tasksStore = useTasksStore()

const sorts = [
  { label: 'Последняя активность', value: 'last_activity', icon: 'history' },
  { label: 'Дата создания',        value: 'created_at',    icon: 'calendar_today' },
  { label: 'Дата поступления',     value: 'received_at',   icon: 'inbox' },
  { label: 'Срок исполнения',      value: 'deadline',      icon: 'event' },
]

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
  background: rgba(0, 0, 0, 0.4);
  z-index: 399;
}

.sort-sheet {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: var(--gw-surface);
  border-top: 1px solid var(--gw-border);
  border-radius: 20px 20px 0 0;
  z-index: 400;
  padding: 12px 16px calc(16px + env(safe-area-inset-bottom, 0px));
  display: flex;
  flex-direction: column;
  gap: 6px;
  transform: translateY(105%);
  transition: transform 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}

.sort-sheet--open {
  transform: translateY(0);
}

.sort-handle {
  width: 36px;
  height: 4px;
  border-radius: 2px;
  background: var(--gw-border);
  margin: 0 auto 12px;
  flex-shrink: 0;
}

.sort-sheet-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--gw-text-secondary);
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
  gap: 12px;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--gw-text);
  font-size: 15px;
  cursor: pointer;
  text-align: left;
  transition: background 0.12s, color 0.12s;
}

.sort-btn:hover {
  background: var(--gw-primary-light);
  color: var(--gw-primary);
}

.sort-btn.active {
  background: var(--gw-primary-light);
  color: var(--gw-primary);
  border-color: color-mix(in oklch, var(--gw-primary) 30%, transparent);
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
  color: var(--gw-primary);
}

.reset-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 8px;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid var(--gw-border);
  background: transparent;
  color: var(--gw-text);
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
