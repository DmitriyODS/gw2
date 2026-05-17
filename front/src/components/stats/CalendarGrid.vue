<template>
  <div class="calendar-grid">
    <div
      v-for="day in data"
      :key="day.date"
      class="calendar-cell"
    >
      <div class="cell-date">{{ formatCellDate(day.date) }}</div>
      <div class="cell-stat">Поступило: <span class="stat-val">{{ day.received }}</span></div>
      <div class="cell-stat">Закрыто: <span class="stat-val closed">{{ day.closed }}</span></div>
      <div class="cell-stat">Время: <span class="stat-val hours">{{ formatHours(day.total_hours) }}</span></div>
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
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  padding: 10px;
  background: var(--gw-bg);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.cell-date {
  font-size: 13px;
  font-weight: 700;
  color: var(--gw-primary);
  margin-bottom: 4px;
}

.cell-stat {
  font-size: 12px;
  color: var(--gw-text-secondary);
  display: flex;
  gap: 4px;
}

.stat-val {
  font-weight: 600;
  color: var(--gw-text);
}

.stat-val.closed {
  color: var(--color-error);
}

.stat-val.hours {
  color: var(--gw-accent);
}

.empty-calendar {
  grid-column: 1 / -1;
  text-align: center;
  padding: 32px;
  color: var(--gw-text-secondary);
  font-size: 14px;
}

@media (max-width: 480px) {
  .calendar-grid {
    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    gap: 6px;
  }

  .calendar-cell {
    padding: 8px;
  }

  .cell-stat {
    font-size: 11px;
  }
}
</style>
