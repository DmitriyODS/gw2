<template>
  <div class="period-control">
    <div class="period-display" @click="showPicker = !showPicker">
      <span class="material-symbols-outlined">calendar_month</span>
      {{ period.displayLabel.value }}
    </div>

    <div v-if="showPicker" class="period-picker-overlay" @click.self="showPicker = false">
      <div class="period-picker">
        <DatePicker
          v-model="customRange"
          selection-mode="range"
          date-format="dd.mm.yy"
          inline
          @update:model-value="onCustomRange"
        />
      </div>
    </div>

    <div class="period-buttons">
      <div class="period-modes">
        <button
          v-for="m in modes"
          :key="m.value"
          class="mode-btn"
          :class="{ active: period.mode.value === m.value }"
          @click="period.selectMode(m.value)"
        >
          {{ m.label }}
        </button>
      </div>

      <div class="period-shift">
        <button class="period-btn" @click="period.shift(-1)" :disabled="!canShift" title="Назад">
          <span class="material-symbols-outlined">chevron_left</span>
        </button>
        <button class="period-btn" @click="period.shift(1)" :disabled="!canShift" title="Вперёд">
          <span class="material-symbols-outlined">chevron_right</span>
        </button>
      </div>

      <button
        class="all-time-btn"
        :class="{ active: period.mode.value === 'all' }"
        @click="period.setAllTime()"
        title="Показать все задачи за весь срок"
      >
        <span class="material-symbols-outlined">all_inclusive</span>
        Весь срок
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import DatePicker from 'primevue/datepicker'
import { useStatsPeriod } from '@/composables/useStatsPeriod.js'

const emit = defineEmits(['change'])

const period = useStatsPeriod()
const showPicker = ref(false)
const customRange = ref(null)

const modes = [
  { value: 'day', label: 'День' },
  { value: 'week', label: 'Неделя' },
  { value: 'month', label: 'Месяц' },
  { value: 'year', label: 'Год' },
]

// Сдвиг имеет смысл только для регулярных периодов (не «весь срок»/«произвольный»).
const canShift = computed(() => ['day', 'week', 'month', 'year'].includes(period.mode.value))

function onCustomRange(val) {
  if (Array.isArray(val) && val[0] && val[1]) {
    period.setCustom(val[0], val[1])
    showPicker.value = false
  }
}

watch(
  [period.fromStr, period.toStr],
  ([from, to]) => {
    emit('change', { from, to })
  },
  { immediate: true }
)
</script>

<style scoped>
.period-control {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  padding: 12px 0;
  position: relative;
}

.period-display {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border: 1px solid var(--gw-border);
  border-radius: 20px;
  cursor: pointer;
  background: var(--gw-surface);
  font-size: 14px;
  color: var(--gw-text);
  transition: border-color 0.15s, background 0.15s;
  user-select: none;
  white-space: nowrap;
}

.period-display:hover {
  border-color: var(--gw-primary);
  background: var(--gw-bg);
}

.period-display .material-symbols-outlined {
  font-size: 18px;
  color: var(--gw-primary);
}

.period-picker-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: flex-start;
  justify-content: flex-start;
}

.period-picker {
  position: absolute;
  top: 56px;
  left: 0;
  z-index: 1001;
  background: var(--gw-surface);
  border: 1px solid var(--gw-border);
  border-radius: var(--gw-radius);
  box-shadow: var(--shadow-lg);
  padding: 8px;
}

.period-buttons {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.period-modes {
  display: flex;
  border: 1px solid var(--gw-border);
  border-radius: 10px;
  overflow: hidden;
}

.mode-btn {
  padding: 7px 14px;
  background: var(--gw-surface);
  border: none;
  border-right: 1px solid var(--gw-border);
  color: var(--gw-text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.mode-btn:last-child {
  border-right: none;
}

.mode-btn:hover:not(.active) {
  background: var(--gw-bg);
  color: var(--gw-text);
}

.mode-btn.active {
  background: var(--gw-primary);
  color: var(--color-on-primary);
  font-weight: 600;
}

.period-shift {
  display: flex;
  align-items: center;
  gap: 4px;
}

.period-btn {
  width: 32px;
  height: 32px;
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  background: var(--gw-surface);
  color: var(--gw-text);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
  line-height: 1;
  padding: 0;
}

.period-btn:hover:not(:disabled) {
  background: var(--gw-primary);
  border-color: var(--gw-primary);
  color: var(--color-on-primary);
}

.period-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.period-btn .material-symbols-outlined {
  font-size: 20px;
}

.all-time-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  border: 1px solid var(--gw-border);
  border-radius: 10px;
  background: var(--gw-surface);
  color: var(--gw-text);
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}

.all-time-btn:hover:not(.active) {
  background: var(--gw-bg);
  border-color: var(--gw-primary);
}

.all-time-btn.active {
  background: var(--color-tertiary);
  border-color: var(--color-tertiary);
  color: var(--color-on-tertiary);
  font-weight: 600;
}

.all-time-btn .material-symbols-outlined {
  font-size: 18px;
}

@media (max-width: 768px) {
  .period-control {
    gap: 10px;
    padding: 8px 0;
  }

  .period-display {
    padding: 7px 12px;
    font-size: 13px;
  }

  .period-buttons {
    gap: 8px;
  }

  .mode-btn {
    padding: 6px 11px;
    font-size: 12px;
  }

  .period-btn {
    width: 30px;
    height: 30px;
  }

  .all-time-btn {
    padding: 6px 11px;
    font-size: 12px;
  }
}
</style>
