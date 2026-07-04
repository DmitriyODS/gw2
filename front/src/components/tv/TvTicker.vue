<template>
  <footer class="tv-ticker">
    <span class="tv-ticker-mark">
      <span class="tv-ticker-dot"></span>
      ЛЕНТА
    </span>
    <div class="tv-ticker-viewport">
      <div class="tv-ticker-track" :style="{ animationDuration: tickerDuration + 's' }">
        <span v-for="(item, i) in itemsX2" :key="i" class="tv-ticker-item">
          <span class="tv-ticker-bullet">●</span>{{ item }}
        </span>
      </div>
    </div>
  </footer>
</template>

<script setup>
// Бегущая строка: собирает живые фразы из статистики всех периодов.
import { computed } from 'vue'
import { num, sumHours, plural, formatHoursShort } from './tvFormat.js'

const props = defineProps({
  commonByPeriod: { type: Object, default: () => ({}) },
  extendedByPeriod: { type: Object, default: () => ({}) },
  hoursPerDay: { type: Number, default: 8 },
})

function fmtH(v) { return formatHoursShort(v, props.hoursPerDay) }

const items = computed(() => {
  const out = []
  const day = props.commonByPeriod['day']
  const week = props.commonByPeriod['week']
  const month = props.commonByPeriod['month']
  const dayExt = props.extendedByPeriod['day']
  const weekExt = props.extendedByPeriod['week']

  if (day?.tasks) {
    out.push(`Сегодня закрыто ${day.tasks.closed} ${plural(day.tasks.closed, 'задача', 'задачи', 'задач')}`)
    out.push(`Сегодня поступило ${day.tasks.received} ${plural(day.tasks.received, 'задача', 'задачи', 'задач')}`)
  }
  const dayLeader = (day?.tasks_by_employees || []).slice().sort((a, b) => num(b.total_hours) - num(a.total_hours))[0]
  if (dayLeader) out.push(`Лидер дня — ${dayLeader.fio}, ${fmtH(dayLeader.total_hours)}`)

  const dayDept = (dayExt?.by_departments || []).slice().sort((a, b) => num(b.tasks_count) - num(a.tasks_count))[0]
  if (dayDept) out.push(`Активный отдел дня — ${dayDept.name}`)

  if (week?.tasks) {
    out.push(`За неделю закрыто ${week.tasks.closed} ${plural(week.tasks.closed, 'задача', 'задачи', 'задач')}`)
  }
  const weekHours = sumHours(week?.tasks_by_employees)
  if (weekHours > 0) out.push(`Команда отработала ${fmtH(weekHours)} за неделю`)

  const weekTopType = (weekExt?.by_unit_types || []).slice().sort((a, b) => num(b.total_hours) - num(a.total_hours))[0]
  if (weekTopType) out.push(`Главный тип работ недели — «${weekTopType.name}»`)

  if (month?.tasks) {
    const monthHours = sumHours(month?.tasks_by_employees)
    if (monthHours > 0) out.push(`За месяц команда наработала ${fmtH(monthHours)}`)
    out.push(`Месяц: ${month.tasks.received} поступило, ${month.tasks.closed} закрыто`)
  }
  if (!out.length) out.push('Загружаем данные…')
  return out
})

// Дублируем дорожку, чтобы анимация была бесшовной.
const itemsX2 = computed(() => [...items.value, ...items.value])

// Скорость прокрутки — пропорционально количеству пунктов.
const tickerDuration = computed(() => Math.max(20, items.value.length * 6))
</script>

<style scoped>
.tv-ticker {
  display: flex;
  align-items: center;
  gap: clamp(12px, 1.6vmin, 20px);
  padding: clamp(8px, 1.2vmin, 14px) clamp(20px, 2.6vmin, 36px);
  background: color-mix(in oklch, var(--color-surface) 80%, transparent);
  border-top: 1px solid var(--color-outline-dim);
  overflow: hidden;
}

.tv-ticker-mark {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: clamp(11px, 1.2vmin, 14px);
  font-weight: 800;
  letter-spacing: 0.16em;
  color: var(--color-error);
  text-transform: uppercase;
  flex-shrink: 0;
}

.tv-ticker-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-error);
  animation: tv-ticker-pulse 1.6s ease-out infinite;
}

@keyframes tv-ticker-pulse {
  0%   { box-shadow: 0 0 0 0 color-mix(in oklch, var(--color-error) 65%, transparent); }
  100% { box-shadow: 0 0 0 12px color-mix(in oklch, var(--color-error) 0%, transparent); }
}

.tv-ticker-viewport {
  flex: 1;
  overflow: hidden;
  min-width: 0;
  mask-image: linear-gradient(90deg, transparent 0%, black 4%, black 96%, transparent 100%);
}

.tv-ticker-track {
  display: inline-flex;
  gap: clamp(28px, 4vmin, 60px);
  white-space: nowrap;
  animation: tv-ticker-scroll linear infinite;
  will-change: transform;
}

@keyframes tv-ticker-scroll {
  from { transform: translateX(0); }
  to   { transform: translateX(-50%); }
}

.tv-ticker-item {
  display: inline-flex;
  align-items: center;
  gap: clamp(8px, 1vmin, 12px);
  font-size: clamp(14px, 1.6vmin, 20px);
  color: var(--color-text);
  font-weight: 600;
}

.tv-ticker-bullet {
  color: var(--color-primary);
  font-size: 8px;
}
</style>
