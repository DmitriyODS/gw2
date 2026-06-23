<template>
  <div class="unit-overlay">
    <div class="unit-modal">
      <button class="minimize-btn" title="Свернуть" @click="minimize">
        <span class="material-symbols-outlined">remove</span>
      </button>
      <p class="unit-header">
        Текущий юнит от {{ formatDate(unit.datetime_start) }}, {{ formatTime(unit.datetime_start) }}
      </p>
      <h2 class="unit-name">{{ unit.name }}</h2>
      <div class="unit-task-pill">
        <span class="material-symbols-outlined">task</span>
        {{ unit.task_name || `Задача #${unit.task_id}` }}
      </div>
      <p class="unit-status">В работе</p>
      <p class="unit-timer">{{ elapsedDisplay }}</p>
      <div class="unit-actions">
        <button v-if="unit.task_id" class="show-task-btn" @click="showTask = true">
          <span class="material-symbols-outlined">open_in_full</span>
          Показать задачу
        </button>
        <button class="stop-btn" @click="stop" :disabled="stopping">
          <span class="material-symbols-outlined">check</span>
          Завершить
        </button>
      </div>
    </div>

    <TaskFloatWindow
      v-if="showTask && unit.task_id"
      :task-id="unit.task_id"
      :task-name="unit.task_name"
      @close="showTask = false"
    />
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useUnitsStore } from '@/stores/units.js'
import { useActiveUnit } from '@/composables/useActiveUnit.js'
import { useElapsed } from '@/composables/useElapsed.js'
import TaskFloatWindow from '@/components/tasks/TaskFloatWindow.vue'

const unitsStore = useUnitsStore()
const { stopping, stop, minimize } = useActiveUnit()

const showTask = ref(false)
const unit = computed(() => unitsStore.activeUnit)

const { display: elapsedDisplay } = useElapsed(() => unit.value?.datetime_start)

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

function formatTime(d) {
  if (!d) return '—'
  return new Date(d).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.unit-overlay {
  position: fixed;
  inset: 0;
  background: color-mix(in oklch, var(--color-bg) 40%, transparent);
  backdrop-filter: blur(4px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
}

.unit-modal {
  position: relative;
  background: var(--gw-surface);
  border-radius: 16px;
  box-shadow: var(--shadow-xl);
  padding: 40px 48px;
  width: 480px;
  max-width: 95vw;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 12px;
}

.minimize-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--gw-text-secondary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.minimize-btn:hover {
  background: var(--gw-primary-light);
  color: var(--gw-primary);
}

.unit-header {
  font-size: 13px;
  color: var(--gw-primary);
  font-weight: 500;
  margin: 0;
}

.unit-name {
  font-size: 24px;
  font-weight: 700;
  color: var(--gw-text);
  margin: 0;
  line-height: 1.3;
}

.unit-task-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--gw-primary-light);
  color: var(--gw-primary);
  border-radius: 20px;
  padding: 6px 14px;
  font-size: 13px;
  font-weight: 500;
}

.unit-task-pill .material-symbols-outlined {
  font-size: 16px;
}

.unit-status {
  font-size: 13px;
  color: var(--gw-text-secondary);
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.unit-timer {
  font-size: 40px;
  font-weight: 700;
  color: var(--gw-text);
  margin: 8px 0;
  font-variant-numeric: tabular-nums;
}

.unit-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: center;
  margin-top: 8px;
}

.show-task-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: var(--gw-primary-light);
  color: var(--gw-primary);
  border: none;
  border-radius: 10px;
  padding: 12px 24px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s, transform 0.1s;
}

.show-task-btn:hover {
  opacity: 0.88;
  transform: translateY(-1px);
}

.show-task-btn .material-symbols-outlined {
  font-size: 20px;
}

.stop-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: var(--gw-accent);
  color: var(--color-on-secondary);
  border: none;
  border-radius: 10px;
  padding: 12px 28px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s, transform 0.1s;
}

.stop-btn:hover:not(:disabled) {
  opacity: 0.88;
  transform: translateY(-1px);
}

.stop-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.stop-btn .material-symbols-outlined {
  font-size: 20px;
}
</style>
