<template>
  <div class="calendar-grid">
    <div
      v-for="day in data"
      :key="day.date"
      class="calendar-cell"
    >
      <div class="cell-date">{{ formatCellDate(day.date) }}</div>
      <div class="cell-stats">
        <div class="cell-stat">
          <span class="material-symbols-outlined stat-ic">north_east</span>
          <span class="stat-val">{{ day.received }}</span>
        </div>
        <div class="cell-stat">
          <span class="material-symbols-outlined stat-ic closed">task_alt</span>
          <span class="stat-val closed">{{ day.closed }}</span>
        </div>
        <div class="cell-stat hours">
          <span class="material-symbols-outlined stat-ic hours">schedule</span>
          <span class="stat-val hours">{{ formatHours(day.total_hours) }}</span>
        </div>
      </div>
    </div>
    <div v-if="!data || data.length === 0" class="empty-calendar">
      Нет данных за выбранный период
    </div>
  </div>
</template>

<script setup>
import { formatHours } from '@/utils/time.js'

defineProps({
  data: {
    type: Array,
    default: () => []
  }
})

function formatCellDate(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' })
}
</script>

<style scoped>
.calendar-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 8px;
}

.calendar-cell {
  border-radius: var(--radius-lg, 16px);
  padding: 12px;
  background: var(--color-surface-high);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cell-date {
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.01em;
  color: var(--color-primary);
}

.cell-stats {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.cell-stat {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-dim);
}

.stat-ic {
  font-size: 16px;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 20;
  color: var(--color-success);
}

.stat-ic.closed {
  color: var(--color-error);
}

.stat-ic.hours {
  color: var(--color-tertiary);
}

.stat-val {
  font-weight: 700;
  font-size: 13px;
  color: var(--color-text);
}

.stat-val.closed {
  color: var(--color-error);
}

.stat-val.hours {
  color: var(--color-tertiary);
}

.empty-calendar {
  grid-column: 1 / -1;
  text-align: center;
  padding: 32px;
  color: var(--color-text-dim);
  font-size: 14px;
}

@media (max-width: 768px) {
  .calendar-grid {
    grid-template-columns: repeat(auto-fill, minmax(132px, 1fr));
    gap: 8px;
  }
}

@media (max-width: 480px) {
  .calendar-grid {
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .calendar-cell {
    padding: 12px;
  }

  .cell-date {
    font-size: 14px;
  }
}

@media (max-width: 360px) {
  .calendar-grid {
    grid-template-columns: 1fr;
  }
}
</style>
