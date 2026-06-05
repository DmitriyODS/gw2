<template>
  <div class="period-control">
    <div ref="displayRef" class="period-display" @click="openPicker">
      <span class="material-symbols-outlined">calendar_month</span>
      {{ period.displayLabel.value }}
    </div>

    <Teleport to="body">
      <div v-if="showPicker" class="period-picker" :style="pickerStyle">
        <DatePicker
          v-model="customRange"
          selection-mode="range"
          date-format="dd.mm.yy"
          inline
          @update:model-value="onCustomRange"
        />
      </div>
    </Teleport>

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
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import DatePicker from 'primevue/datepicker'
import { useStatsPeriod } from '@/composables/useStatsPeriod.js'

const emit = defineEmits(['change'])

const period = useStatsPeriod()
const showPicker = ref(false)
const customRange = ref(null)
const displayRef = ref(null)
const pickerPos = ref({ top: 0, left: 0 })

function openPicker() {
  const rect = displayRef.value?.getBoundingClientRect()
  if (rect) {
    const pickerWidth = 580
    const left = Math.max(8, Math.min(rect.left, window.innerWidth - pickerWidth - 8))
    pickerPos.value = { top: rect.bottom + 8, left }
  }
  showPicker.value = !showPicker.value
}

function onDocClick(e) {
  if (!showPicker.value) return
  if (displayRef.value?.contains(e.target)) return
  const pickerEl = document.querySelector('.period-picker')
  if (pickerEl && pickerEl.contains(e.target)) return
  showPicker.value = false
}

onMounted(() => document.addEventListener('mousedown', onDocClick, true))
onUnmounted(() => document.removeEventListener('mousedown', onDocClick, true))

const pickerStyle = computed(() => ({
  top: `${pickerPos.value.top}px`,
  left: `${pickerPos.value.left}px`,
}))

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
  padding: 4px 0;
  position: relative;
}

.period-display {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  min-height: 44px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  cursor: pointer;
  background: var(--color-surface);
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  transition: border-color 0.15s, background 0.15s;
  user-select: none;
  white-space: nowrap;
}

.period-display:hover {
  border-color: var(--color-primary);
  background: var(--color-surface-high);
}

.period-display .material-symbols-outlined {
  font-size: 18px;
  color: var(--color-primary);
}

.period-picker {
  position: fixed;
  z-index: 1001;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-xl, 20px);
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
  display: inline-flex;
  background: var(--color-surface-high);
  border-radius: var(--radius-full);
  padding: 4px;
  gap: 2px;
}

.mode-btn {
  padding: 8px 14px;
  min-height: 36px;
  background: transparent;
  border: none;
  border-radius: var(--radius-full);
  color: var(--color-text-dim);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}

.mode-btn:hover:not(.active) {
  color: var(--color-text);
}

.mode-btn.active {
  background: var(--color-surface);
  color: var(--color-primary);
  font-weight: 700;
  box-shadow: var(--shadow-sm);
}

.period-shift {
  display: flex;
  align-items: center;
  gap: 4px;
}

.period-btn {
  width: 40px;
  height: 40px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
  line-height: 1;
  padding: 0;
}

.period-btn:hover:not(:disabled) {
  background: var(--color-primary);
  border-color: var(--color-primary);
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
  padding: 10px 16px;
  min-height: 40px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}

.all-time-btn:hover:not(.active) {
  background: var(--color-surface-high);
  border-color: var(--color-primary);
}

.all-time-btn.active {
  background: var(--color-tertiary-container);
  border-color: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  font-weight: 700;
}

.all-time-btn .material-symbols-outlined {
  font-size: 18px;
}

@media (max-width: 768px) {
  .period-control {
    gap: 10px;
    padding: 4px 0;
  }

  .period-display {
    padding: 10px 14px;
    font-size: 13px;
    width: 100%;
    justify-content: center;
  }

  .period-buttons {
    gap: 8px;
    width: 100%;
    justify-content: space-between;
  }

  .period-modes {
    flex: 1;
    justify-content: center;
  }

  .mode-btn {
    flex: 1;
    padding: 10px 8px;
    min-height: 40px;
    font-size: 13px;
  }

  .period-btn {
    width: 40px;
    height: 40px;
  }

  .all-time-btn {
    padding: 10px 14px;
    font-size: 13px;
    min-height: 40px;
  }
}

@media (max-width: 480px) {
  .all-time-btn .material-symbols-outlined {
    margin: 0;
  }
}
</style>
