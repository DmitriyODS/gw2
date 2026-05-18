<template>
  <div class="unit-overlay">
    <div class="unit-modal">
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
      <button class="stop-btn" @click="handleStop" :disabled="stopping">
        <span class="material-symbols-outlined">check</span>
        Завершить
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const unitsStore = useUnitsStore()
const notifications = useNotificationsStore()

const stopping = ref(false)
let timer = null

const unit = computed(() => unitsStore.activeUnit)

// Реактивный счётчик — обновляем каждую секунду
const tick = ref(0)

onMounted(() => {
  timer = setInterval(() => { tick.value++ }, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

// Используем tick чтобы computed пересчитывался каждую секунду
const elapsedDisplay = computed(() => {
  // eslint-disable-next-line no-unused-expressions
  tick.value // зависимость для реактивности
  if (!unit.value) return '—'
  return formatDuration(unit.value.datetime_start, null)
})

function formatDuration(start) {
  const totalSec = Math.max(0, Math.floor((Date.now() - new Date(start)) / 1000))
  const h = Math.floor(totalSec / 3600)
  const m = Math.floor((totalSec % 3600) / 60)
  const s = totalSec % 60
  const ss = String(s).padStart(2, '0')
  const mm = String(m).padStart(2, '0')
  if (h > 0) return `${h} ч ${mm} мин ${ss} сек`
  if (m > 0) return `${m} мин ${ss} сек`
  return `${s} сек`
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

function formatTime(d) {
  if (!d) return '—'
  return new Date(d).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

async function handleStop() {
  stopping.value = true
  try {
    await unitsStore.stop()
    notifications.success('Юнит успешно завершён')
  } catch (e) {
    notifications.error(e?.message || 'Не удалось завершить юнит')
  } finally {
    stopping.value = false
  }
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
  margin-top: 8px;
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
